package services

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"narrabyte/internal/events"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/models"
	"narrabyte/internal/utils"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// On pourrait lowkey rendre ca plus generique pour n'importe quel client
// Interface pour clients?

type sessionRuntime struct {
	client        *client.LLMClient
	modelKey      string
	modelDisplay  string
	providerID    string
	providerLabel string
	targetBranch  string
}

type ClientService struct {
	context                context.Context
	repoLinks              RepoLinkService
	gitService             *GitService
	keyringService         *KeyringService
	generationSessions     GenerationSessionService
	modelConfigs           ModelConfigService
	sessionMu              sync.RWMutex
	sessionRuntimes        map[string]*sessionRuntime // sessionKey -> runtime
	tabBoundSessions       map[uint]bool              // sessionID -> is bound to a tab
	docsBranchesMu         sync.Mutex
	inProgressDocsBranches map[string]bool
}

func (s *ClientService) Startup(ctx context.Context) error {
	s.context = ctx
	if s.repoLinks == nil {
		return fmt.Errorf("repo link service not configured")
	}
	if s.gitService == nil {
		return fmt.Errorf("git service not configured")
	}
	if s.keyringService == nil {
		return fmt.Errorf("keyring service not configured")
	}
	if s.generationSessions == nil {
		return fmt.Errorf("generation session service not configured")
	}
	if s.modelConfigs == nil {
		return fmt.Errorf("model configuration service not configured")
	}
	return nil
}

func NewClientService(repoLinks RepoLinkService, gitService *GitService, keyringService *KeyringService, genSessions GenerationSessionService, modelConfigs ModelConfigService) *ClientService {
	return &ClientService{
		repoLinks:              repoLinks,
		gitService:             gitService,
		keyringService:         keyringService,
		generationSessions:     genSessions,
		modelConfigs:           modelConfigs,
		sessionRuntimes:        make(map[string]*sessionRuntime),
		tabBoundSessions:       make(map[uint]bool),
		inProgressDocsBranches: make(map[string]bool),
	}
}

type docRepoConfig struct {
	RepoRoot       string
	DocsPath       string
	DocsRelative   string
	SharedWithCode bool
}

type tempDocWorkspace struct {
	repoPath string
	docsPath string
}

func makeSessionKey(sessionID uint) string {
	return fmt.Sprintf("session:%d", sessionID)
}

func resolveSessionKey(sessionKeyOverride string, sessionID uint) string {
	override := strings.TrimSpace(sessionKeyOverride)
	if override != "" {
		return override
	}
	return makeSessionKey(sessionID)
}

func (s *ClientService) ensureDocsBranchAvailable(docRepo *git.Repository, docsBranch string) error {
	if s.isDocsBranchInProgress(docsBranch) {
		suggested, suggestErr := s.suggestAlternativeDocsBranch(docRepo, docsBranch)
		if suggestErr != nil {
			return fmt.Errorf("ERR_DOCS_GENERATION_IN_PROGRESS:%s", docsBranch)
		}
		return fmt.Errorf("ERR_DOCS_GENERATION_IN_PROGRESS_SUGGEST:%s:%s", docsBranch, suggested)
	}

	exists, err := s.gitService.BranchExists(docRepo, docsBranch)
	if err != nil {
		return fmt.Errorf("failed to check documentation branch existence: %w", err)
	}
	if exists {
		suggested, suggestErr := s.suggestAlternativeDocsBranch(docRepo, docsBranch)
		if suggestErr != nil {
			return fmt.Errorf("ERR_DOCS_BRANCH_EXISTS:%s", docsBranch)
		}
		return fmt.Errorf("ERR_DOCS_BRANCH_EXISTS_SUGGEST:%s:%s", docsBranch, suggested)
	}

	return nil
}

// CheckDocsBranchAvailability checks if a docs branch can be created without conflict.
// This should be called before starting a generation to allow the UI to show
// conflict dialogs before transitioning to "running" state.
// Returns nil if the branch is available, or an error with the conflict details.
func (s *ClientService) CheckDocsBranchAvailability(projectID uint, sourceBranch string, docsBranchOverride string) error {
	sourceBranch = strings.TrimSpace(sourceBranch)
	if projectID == 0 || sourceBranch == "" {
		return fmt.Errorf("project id and source branch are required")
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("project not found")
	}

	docRepoPath := strings.TrimSpace(project.DocumentationRepo)
	if docRepoPath == "" {
		return fmt.Errorf("documentation repository is not configured")
	}

	docRepo, err := s.gitService.Open(docRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open documentation repository: %w", err)
	}

	// Determine docs branch name
	docsBranch := strings.TrimSpace(docsBranchOverride)
	if docsBranch == "" {
		docsBranch = fmt.Sprintf("docs/%s", sourceBranch)
	}

	return s.ensureDocsBranchAvailable(docRepo, docsBranch)
}

func (s *ClientService) prepareProjectRepos(projectID uint) (*models.RepoLink, string, *docRepoConfig, error) {
	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return nil, "", nil, err
	}
	if project == nil {
		return nil, "", nil, fmt.Errorf("project not found")
	}

	codeRepoPath := strings.TrimSpace(project.CodebaseRepo)
	docRepoPath := strings.TrimSpace(project.DocumentationRepo)
	if codeRepoPath == "" || docRepoPath == "" {
		return nil, "", nil, fmt.Errorf("project repositories are not configured")
	}
	if !utils.DirectoryExists(codeRepoPath) {
		return nil, "", nil, fmt.Errorf("codebase repository path does not exist: %s", codeRepoPath)
	}
	if !utils.DirectoryExists(docRepoPath) {
		return nil, "", nil, fmt.Errorf("documentation repository path does not exist: %s", docRepoPath)
	}
	if !utils.HasGitRepo(codeRepoPath) {
		return nil, "", nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	if !utils.HasGitRepo(docRepoPath) {
		return nil, "", nil, fmt.Errorf("documentation repository is not a git repository: %s", docRepoPath)
	}

	codeRootAbs, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return nil, "", nil, err
	}
	codeRepoRoot, ok := utils.FindGitRepoRoot(codeRootAbs)
	if !ok {
		return nil, "", nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}

	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
	if err != nil {
		return nil, "", nil, err
	}

	return project, codeRepoRoot, docCfg, nil
}

func (s *ClientService) instantiateLLMClient(modelKey string) (*client.LLMClient, *models.LLMModel, error) {
	if s.context == nil {
		return nil, nil, fmt.Errorf("client service not initialized")
	}
	if s.keyringService == nil {
		return nil, nil, fmt.Errorf("keyring service not configured")
	}

	model, err := s.modelConfigs.GetModel(modelKey)
	if err != nil {
		return nil, nil, err
	}
	if model == nil {
		return nil, nil, fmt.Errorf("model %s not found", modelKey)
	}
	if !model.Enabled {
		return nil, nil, fmt.Errorf("model %s is disabled", model.DisplayName)
	}

	providerID := strings.TrimSpace(model.ProviderID)
	if providerID == "" {
		return nil, nil, fmt.Errorf("model %s is missing provider information", model.DisplayName)
	}

	apiKey, err := s.keyringService.GetApiKey(providerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get API key for %s: %w", providerID, err)
	}
	if apiKey == "" {
		return nil, nil, fmt.Errorf("API key for %s is not configured", providerID)
	}

	var (
		llmClient *client.LLMClient
		createErr error
	)
	switch providerID {
	case "anthropic":
		llmClient, createErr = client.NewClaudeClient(s.context, apiKey, client.ClaudeModelOptions{
			Model:    model.APIName,
			Thinking: model.Thinking != nil && *model.Thinking,
		})
	case "openai":
		llmClient, createErr = client.NewOpenAIClient(s.context, apiKey, client.OpenAIModelOptions{
			Model:           model.APIName,
			ReasoningEffort: model.ReasoningEffort,
		})
	case "gemini":
		llmClient, createErr = client.NewGeminiClient(s.context, apiKey, client.GeminiModelOptions{
			Model:    model.APIName,
			Thinking: model.Thinking != nil && *model.Thinking,
		})
	default:
		return nil, nil, fmt.Errorf("unsupported provider: %s", providerID)
	}

	if createErr != nil {
		return nil, nil, fmt.Errorf("failed to create %s client: %w", providerID, createErr)
	}

	return llmClient, model, nil
}

func (s *ClientService) newSessionRuntime(modelKey string) (*sessionRuntime, *models.LLMModel, error) {
	modelKey = strings.TrimSpace(modelKey)
	if modelKey == "" {
		return nil, nil, fmt.Errorf("model is required")
	}
	llmClient, modelInfo, err := s.instantiateLLMClient(modelKey)
	if err != nil {
		return nil, nil, err
	}
	providerLabel := strings.TrimSpace(modelInfo.ProviderName)
	providerID := strings.TrimSpace(modelInfo.ProviderID)
	if providerLabel == "" {
		providerLabel = providerID
	}
	runtime := &sessionRuntime{
		client:        llmClient,
		modelKey:      modelKey,
		modelDisplay:  modelInfo.DisplayName,
		providerID:    providerID,
		providerLabel: providerLabel,
	}
	return runtime, modelInfo, nil
}

func (s *ClientService) getSessionRuntime(sessionKey string) (*sessionRuntime, bool) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	runtime, ok := s.sessionRuntimes[sessionKey]
	return runtime, ok
}

func (s *ClientService) setSessionRuntime(sessionKey string, runtime *sessionRuntime) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if runtime == nil {
		delete(s.sessionRuntimes, sessionKey)
		return
	}
	if existing, ok := s.sessionRuntimes[sessionKey]; ok && existing != runtime && existing.client != nil {
		existing.client.StopStream()
	}
	s.sessionRuntimes[sessionKey] = runtime
}

func (s *ClientService) deleteSessionRuntime(sessionKey string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if existing, ok := s.sessionRuntimes[sessionKey]; ok && existing != nil && existing.client != nil {
		existing.client.StopStream()
	}
	delete(s.sessionRuntimes, sessionKey)
}

// markDocsBranchInProgress attempts to mark a documentation branch as in-progress.
// Returns an error if the branch is already being generated.
func (s *ClientService) markDocsBranchInProgress(docsBranch string) error {
	s.docsBranchesMu.Lock()
	defer s.docsBranchesMu.Unlock()
	if s.inProgressDocsBranches[docsBranch] {
		return fmt.Errorf("ERR_DOCS_GENERATION_IN_PROGRESS:%s", docsBranch)
	}
	s.inProgressDocsBranches[docsBranch] = true
	return nil
}

// unmarkDocsBranchInProgress removes a documentation branch from the in-progress tracking.
func (s *ClientService) unmarkDocsBranchInProgress(docsBranch string) {
	s.docsBranchesMu.Lock()
	defer s.docsBranchesMu.Unlock()
	delete(s.inProgressDocsBranches, docsBranch)
}

// isDocsBranchInProgress checks if a docs branch is currently being generated.
func (s *ClientService) isDocsBranchInProgress(docsBranch string) bool {
	s.docsBranchesMu.Lock()
	defer s.docsBranchesMu.Unlock()
	return s.inProgressDocsBranches[docsBranch]
}

// suggestAlternativeDocsBranch generates an alternative branch name by appending a numeric suffix.
// It checks both existing branches and in-progress generations to find an available name.
func (s *ClientService) suggestAlternativeDocsBranch(repo *git.Repository, baseName string) (string, error) {
	// Try numeric suffixes starting from 2
	for i := 2; i <= 100; i++ {
		candidate := fmt.Sprintf("%s-%d", baseName, i)

		// Check if already being generated
		if s.isDocsBranchInProgress(candidate) {
			continue
		}

		// Check if branch exists in repository
		exists, err := s.gitService.BranchExists(repo, candidate)
		if err != nil {
			return "", fmt.Errorf("failed to check branch existence: %w", err)
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not find available branch name after 100 attempts")
}

func (s *ClientService) ensureRuntimeFromSessions(ctx context.Context, projectID uint, sourceBranch, targetBranch, sessionKey string) (*sessionRuntime, error) {
	if runtime, ok := s.getSessionRuntime(sessionKey); ok && runtime != nil {
		return runtime, nil
	}

	sessions, err := s.generationSessions.List(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to load generation sessions: %w", err)
	}

	for _, sess := range sessions {
		if strings.TrimSpace(sess.SourceBranch) != sourceBranch {
			continue
		}
		if targetBranch != "" && strings.TrimSpace(sess.TargetBranch) != targetBranch {
			continue
		}
		modelKey := strings.TrimSpace(sess.ModelKey)
		providerID := strings.TrimSpace(sess.Provider)
		if modelKey == "" && providerID != "" {
			if fallback, fbErr := s.findDefaultModelForProvider(providerID); fbErr == nil && fallback != nil {
				modelKey = fallback.Key
			}
		}
		if modelKey == "" {
			continue
		}
		runtime, modelInfo, err := s.newSessionRuntime(modelKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LLM client from session: %w", err)
		}
		runtime.targetBranch = strings.TrimSpace(sess.TargetBranch)
		if runtime.targetBranch == "" {
			runtime.targetBranch = strings.TrimSpace(sess.SourceBranch)
		}
		s.setSessionRuntime(sessionKey, runtime)

		if modelInfo != nil {
			emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Initialized %s via %s from session", modelInfo.DisplayName, runtime.providerLabel))
		}

		if sess.MessagesJSON != "" {
			if loadErr := runtime.client.LoadConversationHistoryJSON(sess.MessagesJSON); loadErr != nil {
				emitSessionWarn(ctx, sessionKey, fmt.Sprintf("Failed to restore conversation history: %v", loadErr))
			} else {
				emitSessionInfo(ctx, sessionKey, "Restored LLM conversation history")
			}
		}
		return runtime, nil
	}

	return nil, fmt.Errorf("LLM client not initialized - please run GenerateDocs first or restore a session")
}

// ensureRuntimeFromSession creates or retrieves a session runtime from a specific session
func (s *ClientService) ensureRuntimeFromSession(ctx context.Context, session *models.GenerationSession, sessionKey string) (*sessionRuntime, error) {
	if runtime, ok := s.getSessionRuntime(sessionKey); ok && runtime != nil {
		return runtime, nil
	}

	modelKey := strings.TrimSpace(session.ModelKey)
	providerID := strings.TrimSpace(session.Provider)
	if modelKey == "" && providerID != "" {
		if fallback, fbErr := s.findDefaultModelForProvider(providerID); fbErr == nil && fallback != nil {
			modelKey = fallback.Key
		}
	}
	if modelKey == "" {
		return nil, fmt.Errorf("session has no model key configured")
	}

	runtime, modelInfo, err := s.newSessionRuntime(modelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client from session: %w", err)
	}
	runtime.targetBranch = strings.TrimSpace(session.TargetBranch)
	if runtime.targetBranch == "" {
		runtime.targetBranch = strings.TrimSpace(session.SourceBranch)
	}
	s.setSessionRuntime(sessionKey, runtime)

	if modelInfo != nil {
		emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Initialized %s via %s from session", modelInfo.DisplayName, runtime.providerLabel))
	}

	if session.MessagesJSON != "" {
		if loadErr := runtime.client.LoadConversationHistoryJSON(session.MessagesJSON); loadErr != nil {
			emitSessionWarn(ctx, sessionKey, fmt.Sprintf("Failed to restore conversation history: %v", loadErr))
		} else {
			emitSessionInfo(ctx, sessionKey, "Restored LLM conversation history")
		}
	}
	return runtime, nil
}

func emitSessionInfo(ctx context.Context, sessionKey string, message string) {
	evt := events.NewInfo(message)
	evt.SessionKey = sessionKey
	events.Emit(ctx, events.LLMEventTool, evt)
}

func emitSessionWarn(ctx context.Context, sessionKey string, message string) {
	evt := events.NewWarn(message)
	evt.SessionKey = sessionKey
	events.Emit(ctx, events.LLMEventTool, evt)
}

func emitSessionError(ctx context.Context, sessionKey string, message string) {
	evt := events.NewError(message)
	evt.SessionKey = sessionKey
	events.Emit(ctx, events.LLMEventTool, evt)
}

func emitSessionDebug(ctx context.Context, sessionKey string, message string) {
	evt := events.NewInfo(message)
	evt.SessionKey = sessionKey
	events.Emit(ctx, events.LLMEventTool, evt)
}

func (s *ClientService) GenerateDocs(projectID uint, sourceBranch string, targetBranch string, modelKey string, userInstructions string, docsBranchOverride string, sessionKeyOverride string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	modelKey = strings.TrimSpace(modelKey)
	docsBranchOverride = strings.TrimSpace(docsBranchOverride)
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}
	if sourceBranch == targetBranch {
		return nil, fmt.Errorf("source and target branches must differ")
	}
	if modelKey == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Determine docs branch name early
	docsBranch := documentationBranchName(sourceBranch)
	if docsBranchOverride != "" {
		docsBranch = docsBranchOverride
	}

	// Check if a session with this docsBranch already exists
	existingSession, err := s.generationSessions.GetByDocsBranch(projectID, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing session: %w", err)
	}
	if existingSession != nil {
		return nil, fmt.Errorf("ERR_SESSION_EXISTS:a session with docsBranch '%s' already exists (ID: %d)", docsBranch, existingSession.ID)
	}

	runtime, modelInfo, err := s.newSessionRuntime(modelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	runtime.targetBranch = targetBranch

	providerID := strings.TrimSpace(modelInfo.ProviderID)

	// Create session record to get ID
	session := &models.GenerationSession{
		ProjectID:    projectID,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Provider:     providerID,
		ModelKey:     runtime.modelKey,
		DocsBranch:   docsBranch,
	}
	session, err = s.generationSessions.Create(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sessionKey := resolveSessionKey(sessionKeyOverride, session.ID)
	s.setSessionRuntime(sessionKey, runtime)

	project, codeRoot, docCfg, err := s.prepareProjectRepos(projectID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(docsBranchOverride) != "" {
		emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
			"GenerateDocs: starting for project %s (%s -> %s) using %s via %s into %s",
			project.ProjectName, targetBranch, sourceBranch, runtime.modelDisplay, runtime.providerLabel, docsBranchOverride,
		))
	} else {
		emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
			"GenerateDocs: starting for project %s (%s -> %s) using %s via %s",
			project.ProjectName, targetBranch, sourceBranch, runtime.modelDisplay, runtime.providerLabel,
		))
	}

	codeRepo, err := s.gitService.Open(codeRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open code repository: %w", err)
	}

	targetHash, err := resolveBranchHash(codeRepo, targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target branch '%s': %w", targetBranch, err)
	}
	sourceHash, err := resolveBranchHash(codeRepo, sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source branch '%s': %w", sourceBranch, err)
	}

	diffText, err := s.gitService.DiffBetweenCommits(codeRepo, targetHash.String(), sourceHash.String())
	if err != nil {
		return nil, fmt.Errorf("failed to compute branch diff: %w", err)
	}
	changedFiles := extractPathsFromDiff(diffText)
	if len(changedFiles) == 0 {
		emitSessionInfo(ctx, sessionKey, "GenerateDocs: no code changes detected between branches")
	}

	docRepo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open documentation repository: %w", err)
	}
	docWorktree, err := docRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to load documentation worktree: %w", err)
	}

	status, err := docWorktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation repo status: %w", err)
	}
	if hasDocsChanges(status, docCfg.DocsRelative) {
		emitSessionWarn(ctx, sessionKey, "Documentation repository has uncommitted changes - these will be preserved")
	}

	var (
		baseHash   plumbing.Hash
		baseBranch string
	)
	if docCfg.SharedWithCode {
		baseHash = sourceHash
		baseBranch = sourceBranch
	} else {
		baseHash, baseBranch, err = resolveDocumentationBase(project, docRepo)
		if err != nil {
			return nil, err
		}
	}

	// PRE-CHECK: prevent silently overwriting an existing docs/<source> branch
	if err := s.ensureDocsBranchAvailable(docRepo, docsBranch); err != nil {
		// Clean up the session we created since we're failing
		_ = s.generationSessions.DeleteByID(session.ID)
		return nil, err
	}

	// Mark this docs branch as in-progress to prevent concurrent generations
	if err := s.markDocsBranchInProgress(docsBranch); err != nil {
		return nil, err
	}
	defer s.unmarkDocsBranchInProgress(docsBranch)

	// Create temporary documentation repository (isolated from working directory)
	tempWorkspace, cleanup, err := createTempDocRepo(ctx, sessionKey, docCfg, docsBranch, baseBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup() // Always cleanup temp directory

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"GenerateDocs: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	))

	streamCtx := runtime.client.StartStream(ctx, sessionKey)
	defer runtime.client.StopStream()

	// Use temporary documentation root for LLM operations
	llmResult, err := runtime.client.GenerateDocs(streamCtx, &client.DocGenerationRequest{
		ProjectName:          project.ProjectName,
		CodebasePath:         codeRoot,
		DocumentationPath:    tempWorkspace.docsPath, // Use temporary workspace scoped to docs
		DocumentationRelPath: docCfg.DocsRelative,
		SourceBranch:         sourceBranch,
		TargetBranch:         targetBranch,
		SourceCommit:         sourceHash.String(),
		Diff:                 diffText,
		ChangedFiles:         changedFiles,
		SpecificInstr:        userInstructions,
	})
	if err != nil {
		return nil, err
	}

	// Propagate changes from temporary repository back to main repository
	files, err := propagateDocChanges(ctx, sessionKey, tempWorkspace, docRepo, docsBranch, docCfg.DocsRelative)
	if err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	branchCreated, err := ensureDocsBranchExists(docRepo, docsBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare documentation branch '%s': %w", docsBranch, err)
	}
	if branchCreated {
		emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Initialized docs branch '%s' from '%s' for diff", docsBranch, baseBranch))
	}

	// Save conversation history to session
	if runtime.client != nil {
		if jsonStr, err := runtime.client.ConversationHistoryJSON(); err == nil {
			_ = s.generationSessions.UpdateByID(session.ID, map[string]interface{}{
				"messages_json":      jsonStr,
				"chat_messages_json": "[]",
			})
		}
	}

	// Generate diff between the new docs branch and its base branch
	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	emitSessionInfo(ctx, sessionKey, "GenerateDocs: completed")

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
		SessionID:      session.ID,
		SessionKey:     sessionKey,
		Branch:         sourceBranch,
		TargetBranch:   targetBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        summary,
	}, nil
}

// RefineDocs applies a user-provided instruction to the documentation branch
// for a given session. It reuses the same toolset as GenerateDocs but focuses
// on targeted edits directed by the user's request.
func (s *ClientService) RefineDocs(sessionID uint, instruction string, sessionKeyOverride string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	if sessionID == 0 {
		return nil, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(instruction) == "" {
		return nil, fmt.Errorf("instruction is required")
	}

	// Load session from database
	session, err := s.generationSessions.GetByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found: %d", sessionID)
	}

	projectID := session.ProjectID
	sourceBranch := strings.TrimSpace(session.SourceBranch)
	docsBranch := strings.TrimSpace(session.DocsBranch)
	sessionKey := resolveSessionKey(sessionKeyOverride, sessionID)

	// Check if this docs branch is already being refined/generated
	if s.isDocsBranchInProgress(docsBranch) {
		return nil, fmt.Errorf("ERR_DOCS_GENERATION_IN_PROGRESS:%s", docsBranch)
	}

	// Mark this docs branch as in-progress to prevent concurrent refinements
	if err := s.markDocsBranchInProgress(docsBranch); err != nil {
		return nil, err
	}
	defer s.unmarkDocsBranchInProgress(docsBranch)

	runtime, err := s.ensureRuntimeFromSession(ctx, session, sessionKey)
	if err != nil {
		return nil, err
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"RefineDocs: starting for project %d (%s)",
		projectID, docsBranch,
	))

	project, codeRoot, docCfg, err := s.prepareProjectRepos(projectID)
	if err != nil {
		return nil, err
	}

	// Open documentation repo and ensure the docs branch exists
	docRepo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open documentation repository: %w", err)
	}

	var (
		baseHash   plumbing.Hash
		baseBranch string
	)
	if docCfg.SharedWithCode {
		baseBranch = sourceBranch
		baseHash, err = resolveBranchHash(docRepo, sourceBranch)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source branch '%s': %w", sourceBranch, err)
		}
	} else {
		baseHash, baseBranch, err = resolveDocumentationBase(project, docRepo)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(runtime.targetBranch) == "" && strings.TrimSpace(baseBranch) != "" {
		runtime.targetBranch = strings.TrimSpace(baseBranch)
	}

	existingChat := s.loadStoredChatMessagesFromSession(session)

	refName := plumbing.NewBranchReferenceName(docsBranch)
	// Ensure the docs branch exists in the main repo; if not, create it off base
	if _, err := docRepo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			// Create docs branch pointing to base commit
			if err := docRepo.Storer.SetReference(plumbing.NewHashReference(refName, baseHash)); err != nil {
				return nil, fmt.Errorf("failed to create documentation branch '%s': %w", docsBranch, err)
			}
			emitSessionInfo(ctx, sessionKey, fmt.Sprintf("RefineDocs: created missing docs branch '%s' from '%s'", docsBranch, baseBranch))
		} else {
			return nil, fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
		}
	}

	// Create a temporary workspace checked out to the current docs branch head
	tempWorkspace, cleanup, err := createTempDocRepoAtBranchHead(ctx, sessionKey, docCfg, docsBranch, baseBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup()

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"RefineDocs: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	))

	streamCtx := runtime.client.StartStream(ctx, sessionKey)
	defer runtime.client.StopStream()

	// Run the refinement agent focused on applying user edits
	llmResult, err := runtime.client.DocRefine(streamCtx, &client.DocRefineRequest{
		ProjectName:          project.ProjectName,
		CodebasePath:         codeRoot,
		DocumentationPath:    tempWorkspace.docsPath,
		DocumentationRelPath: docCfg.DocsRelative,
		SourceBranch:         sourceBranch,
		Instruction:          instruction,
	})
	if err != nil {
		return nil, err
	}

	assistantSummary := ""
	if llmResult != nil {
		assistantSummary = llmResult.Summary
	}
	chatMessages := appendChatMessages(existingChat, instruction, assistantSummary)
	chatMessagesJSON := marshalChatMessages(chatMessages)

	// Propagate changes back to the main documentation repository
	files, err := propagateDocChanges(ctx, sessionKey, tempWorkspace, docRepo, docsBranch, docCfg.DocsRelative)
	if err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	// Update diff between base branch and docs branch for UI preview
	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	// Save conversation history and chat messages to session
	if runtime.client != nil {
		if jsonStr, err := runtime.client.ConversationHistoryJSON(); err == nil {
			_ = s.generationSessions.UpdateByID(sessionID, map[string]interface{}{
				"messages_json":      jsonStr,
				"chat_messages_json": chatMessagesJSON,
			})
		}
	}

	emitSessionInfo(ctx, sessionKey, "RefineDocs: completed")

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
		SessionID:      sessionID,
		SessionKey:     sessionKey,
		Branch:         sourceBranch,
		TargetBranch:   baseBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        summary,
		ChatMessages:   chatMessages,
	}, nil
}

// MergeDocsIntoSource fast-forwards the source code branch to include the latest
// documentation commit generated on docs/<sourceBranch>. Only supported when
// documentation lives within the code repository.
func (s *ClientService) MergeDocsIntoSource(sessionID uint) error {
	ctx := s.context
	if ctx == nil {
		return fmt.Errorf("client service not initialized")
	}
	if sessionID == 0 {
		return fmt.Errorf("session ID is required")
	}

	session, err := s.generationSessions.GetByID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	projectID := session.ProjectID
	sourceBranch := strings.TrimSpace(session.SourceBranch)
	sessionKey := makeSessionKey(sessionID)

	_, _, docCfg, err := s.prepareProjectRepos(projectID)
	if err != nil {
		return err
	}
	if !docCfg.SharedWithCode {
		return fmt.Errorf("documentation repository is separate; merge into source branch is not supported")
	}

	repo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if currently on source branch with uncommitted changes
	currentBranch, err := s.gitService.GetCurrentBranch(docCfg.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	if currentBranch == sourceBranch {
		hasUncommitted, err := s.gitService.HasUncommittedChanges(docCfg.RepoRoot)
		if err != nil {
			return fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}
		if hasUncommitted {
			return fmt.Errorf("ERR_UNCOMMITTED_CHANGES_ON_SOURCE_BRANCH")
		}
	}

	docsBranch := documentationBranchName(sourceBranch)
	docRefName := plumbing.NewBranchReferenceName(docsBranch)
	sourceRefName := plumbing.NewBranchReferenceName(sourceBranch)

	docRef, err := repo.Reference(docRefName, true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("documentation branch '%s' does not exist", docsBranch)
		}
		return fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
	}

	docCommit, err := repo.CommitObject(docRef.Hash())
	if err != nil {
		return fmt.Errorf("failed to load documentation commit: %w", err)
	}
	if docCommit.NumParents() == 0 {
		return fmt.Errorf("documentation branch commit has no parent; cannot merge")
	}

	sourceRef, err := repo.Reference(sourceRefName, true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("source branch '%s' does not exist", sourceBranch)
		}
		return fmt.Errorf("failed to resolve source branch '%s': %w", sourceBranch, err)
	}

	sourceCommit, err := repo.CommitObject(sourceRef.Hash())
	if err != nil {
		return fmt.Errorf("failed to load source commit: %w", err)
	}

	isAncestor, err := sourceCommit.IsAncestor(docCommit)
	if err != nil {
		return fmt.Errorf("failed to verify branch ancestry: %w", err)
	}
	if !isAncestor {
		return fmt.Errorf("source branch '%s' has diverged since documentation was generated", sourceBranch)
	}

	if err := repo.Storer.SetReference(plumbing.NewHashReference(sourceRefName, docRef.Hash())); err != nil {
		return fmt.Errorf("failed to fast-forward source branch '%s': %w", sourceBranch, err)
	}

	if currentBranch == sourceBranch {
		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to load worktree for branch '%s': %w", sourceBranch, err)
		}
		if err := worktree.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: docRef.Hash()}); err != nil {
			return fmt.Errorf("failed to update worktree to documentation commit: %w", err)
		}
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"MergeDocs: fast-forwarded '%s' to include documentation commit %s",
		sourceBranch,
		docRef.Hash().String()[:8],
	))
	return nil
}

func createTempDocWorkspace(ctx context.Context, sessionKey string, cfg *docRepoConfig, branch string, baseBranch string, baseHash plumbing.Hash, checkoutHead bool) (workspace tempDocWorkspace, cleanup func(), err error) {
	if cfg == nil {
		return tempDocWorkspace{}, nil, fmt.Errorf("documentation repository configuration is required")
	}
	repoPath, cleanup := newTempRepoDir(ctx, sessionKey)
	emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Creating temporary docs workspace at %s", repoPath))

	cloneOpts := &git.CloneOptions{
		URL:          cfg.RepoRoot,
		Depth:        1,
		Progress:     nil,
		SingleBranch: true,
	}
	if checkoutHead {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(branch)
	} else if strings.TrimSpace(baseBranch) != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(baseBranch)
	}

	tempRepo, err := git.PlainClone(repoPath, false, cloneOpts)
	if err != nil {
		cleanup()
		repoPath, cleanup = newTempRepoDir(ctx, sessionKey)
		emitSessionWarn(ctx, sessionKey, "Shallow clone failed; retrying with full clone")
		tempRepo, err = git.PlainClone(repoPath, false, &git.CloneOptions{URL: cfg.RepoRoot, Progress: nil})
		if err != nil {
			cleanup()
			return tempDocWorkspace{}, nil, fmt.Errorf("failed to clone repository to temp location: %w", err)
		}
	}

	wt, err := tempRepo.Worktree()
	if err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, fmt.Errorf("failed to get temp repository worktree: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branch)
	if checkoutHead {
		if err := wt.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
			srcRepo, srcOpenErr := git.PlainOpen(cfg.RepoRoot)
			if srcOpenErr != nil {
				cleanup()
				return tempDocWorkspace{}, nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, err)
			}

			var headHash plumbing.Hash
			if srcRef, refErr := srcRepo.Reference(refName, true); refErr == nil {
				headHash = srcRef.Hash()
			} else {
				if baseHash == plumbing.ZeroHash {
					cleanup()
					return tempDocWorkspace{}, nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, err)
				}
				if baseBranch != "" {
					emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Creating docs branch '%s' from base '%s'", branch, baseBranch))
				}
				headHash = baseHash
			}

			if setErr := tempRepo.Storer.SetReference(plumbing.NewHashReference(refName, headHash)); setErr != nil {
				cleanup()
				return tempDocWorkspace{}, nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, err)
			}

			if coErr := wt.Checkout(&git.CheckoutOptions{Branch: refName}); coErr != nil {
				cleanup()
				return tempDocWorkspace{}, nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, coErr)
			}
		}
	} else {
		ref := plumbing.NewHashReference(refName, baseHash)
		if err := tempRepo.Storer.SetReference(ref); err != nil {
			cleanup()
			return tempDocWorkspace{}, nil, fmt.Errorf("failed to create branch '%s' in temp repo: %w", branch, err)
		}

		if err := wt.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
			cleanup()
			return tempDocWorkspace{}, nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, err)
		}
	}

	tempDocsPath := repoPath
	if cfg.DocsRelative != "." {
		tempDocsPath = filepath.Join(repoPath, cfg.DocsRelative)
	}

	if err := copyNarrabyteDir(ctx, sessionKey, cfg.DocsPath, tempDocsPath); err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, err
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Temporary docs workspace ready: branch '%s' at %s", branch, repoPath))
	return tempDocWorkspace{repoPath: repoPath, docsPath: tempDocsPath}, cleanup, nil
}

// createTempDocRepoAtBranchHead clones the documentation repository into a temp directory
// and checks out the specified branch at its current HEAD.
func createTempDocRepoAtBranchHead(ctx context.Context, sessionKey string, cfg *docRepoConfig, branch string, baseBranch string, baseHash plumbing.Hash) (workspace tempDocWorkspace, cleanup func(), err error) {
	return createTempDocWorkspace(ctx, sessionKey, cfg, branch, baseBranch, baseHash, true)
}

func (s *ClientService) findDefaultModelForProvider(provider string) (*models.LLMModel, error) {
	if s.modelConfigs == nil {
		return nil, fmt.Errorf("model configuration service not available")
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, nil
	}
	groups, err := s.modelConfigs.ListModelGroups()
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if strings.TrimSpace(group.ProviderID) != provider {
			continue
		}
		sorted := group.Models
		for i := range sorted {
			if sorted[i].Enabled {
				modelCopy := sorted[i]
				return &modelCopy, nil
			}
		}
	}
	return nil, nil
}

func (s *ClientService) CommitDocs(projectID uint, sessionID uint, files []string) error {
	ctx := s.context
	if ctx == nil {
		return fmt.Errorf("client service not initialized")
	}
	if projectID == 0 {
		return fmt.Errorf("project id is required")
	}
	if sessionID == 0 {
		return fmt.Errorf("session id is required")
	}

	session, err := s.generationSessions.GetByID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %d", sessionID)
	}

	sessionKey := makeSessionKey(sessionID)
	docsBranch := strings.TrimSpace(session.DocsBranch)
	if len(files) == 0 {
		return fmt.Errorf("no documentation files to commit")
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return fmt.Errorf("project not found")
	}

	docRepoPath := strings.TrimSpace(project.DocumentationRepo)
	if docRepoPath == "" {
		return fmt.Errorf("project documentation repository is not configured")
	}
	if !utils.DirectoryExists(docRepoPath) {
		return fmt.Errorf("documentation repository path does not exist: %s", docRepoPath)
	}
	if !utils.HasGitRepo(docRepoPath) {
		return fmt.Errorf("documentation repository is not a git repository: %s", docRepoPath)
	}

	codeRepoRoot := ""
	if codeRepoPath := strings.TrimSpace(project.CodebaseRepo); codeRepoPath != "" {
		if utils.DirectoryExists(codeRepoPath) && utils.HasGitRepo(codeRepoPath) {
			if abs, absErr := filepath.Abs(codeRepoPath); absErr == nil {
				if root, ok := utils.FindGitRepoRoot(abs); ok {
					codeRepoRoot = root
				}
			}
		}
	}
	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
	if err != nil {
		return err
	}

	repo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to open documentation repository: %w", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to load documentation worktree: %w", err)
	}
	refName := plumbing.NewBranchReferenceName(docsBranch)

	// Validate that the branch exists without checking out
	if _, err := repo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("documentation branch '%s' does not exist", docsBranch)
		}
		return fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
	}

	docStatus, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to read documentation repo status: %w", err)
	}
	var normalized []string
	prefix := filepath.ToSlash(filepath.Clean(docCfg.DocsRelative))
	if prefix == "." {
		prefix = ""
	}
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for _, file := range files {
		trimmed := strings.TrimSpace(file)
		if trimmed == "" {
			continue
		}
		rel := filepath.ToSlash(trimmed)
		if prefix != "" && !strings.HasPrefix(rel, prefix) {
			continue
		}
		fsPath := filepath.FromSlash(rel)
		if st, ok := docStatus[fsPath]; ok {
			if st.Worktree == git.Unmodified && st.Staging == git.Unmodified {
				continue
			}
		}
		normalized = append(normalized, fsPath)
	}
	if len(normalized) == 0 {
		return fmt.Errorf("no documentation changes found to commit")
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"CommitDocs: staging %d documentation file(s) for branch '%s'",
		len(normalized), docsBranch,
	))

	if err := s.gitService.StageFiles(repo, normalized); err != nil {
		return fmt.Errorf("failed to stage documentation changes: %w", err)
	}

	message := fmt.Sprintf("Add documentation for %s", docsBranch)
	if _, err := s.gitService.Commit(repo, message); err != nil {
		return fmt.Errorf("failed to commit documentation changes: %w", err)
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"CommitDocs: committed documentation updates to '%s'",
		docsBranch,
	))

	return nil
}

func (s *ClientService) LoadGenerationSession(sessionID uint) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}

	if sessionID == 0 {
		return nil, fmt.Errorf("session id is required")
	}

	session, err := s.generationSessions.GetByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get generation session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found: %d", sessionID)
	}

	projectID := session.ProjectID
	sourceBranch := strings.TrimSpace(session.SourceBranch)
	targetBranch := strings.TrimSpace(session.TargetBranch)
	docsBranch := strings.TrimSpace(session.DocsBranch)
	sessionKey := makeSessionKey(sessionID)

	runtime, err := s.ensureRuntimeFromSession(ctx, session, sessionKey)
	if err != nil {
		return nil, err
	}
	runtime.targetBranch = targetBranch
	if trimmed := strings.TrimSpace(session.ModelKey); trimmed != "" {
		runtime.modelKey = trimmed
	}
	if provider := strings.TrimSpace(session.Provider); provider != "" && strings.TrimSpace(runtime.providerID) == "" {
		runtime.providerID = provider
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"LoadSession: restoring documentation session for project %s (%s → %s)",
		project.ProjectName, targetBranch, sourceBranch,
	))

	codeRepoPath := strings.TrimSpace(project.CodebaseRepo)
	docRepoPath := strings.TrimSpace(project.DocumentationRepo)
	if codeRepoPath == "" || docRepoPath == "" {
		return nil, fmt.Errorf("project repositories are not configured")
	}
	if !utils.DirectoryExists(codeRepoPath) {
		return nil, fmt.Errorf("codebase repository path does not exist: %s", codeRepoPath)
	}
	if !utils.DirectoryExists(docRepoPath) {
		return nil, fmt.Errorf("documentation repository path does not exist: %s", docRepoPath)
	}

	codeRootAbs, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return nil, err
	}
	codeRepoRoot, ok := utils.FindGitRepoRoot(codeRootAbs)
	if !ok {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}

	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
	if err != nil {
		return nil, err
	}

	docRepo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open documentation repository: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(docsBranch)
	if _, err := docRepo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, fmt.Errorf("documentation branch '%s' does not exist - session may be stale", docsBranch)
		}
		return nil, fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
	}

	var baseBranch string
	if docCfg.SharedWithCode {
		baseBranch = sourceBranch
	} else {
		_, baseBranch, err = resolveDocumentationBase(project, docRepo)
		if err != nil {
			return nil, err
		}
	}

	docWorktree, err := docRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to load documentation worktree: %w", err)
	}
	docStatus, err := docWorktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation repo status: %w", err)
	}

	files := collectDocChangedFiles(docStatus, docCfg.DocsRelative)

	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	summary := ""
	if runtime.client != nil {
		summary = strings.TrimSpace(runtime.client.LastAssistantMessage())
	}
	if summary == "" {
		summary = "Restored from previous session"
	}

	chatMessages := parseChatMessagesJSON(session.ChatMessagesJSON)

	emitSessionInfo(ctx, sessionKey, "LoadSession: session restored successfully")

	return &models.DocGenerationResult{
		SessionID:      sessionID,
		SessionKey:     sessionKey,
		Branch:         sourceBranch,
		TargetBranch:   targetBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        summary,
		ChatMessages:   chatMessages,
	}, nil
}

func (s *ClientService) StopStream(sessionID uint, sessionKeyOverride string) {
	if s == nil || sessionID == 0 {
		return
	}
	sessionKey := resolveSessionKey(sessionKeyOverride, sessionID)
	runtime, ok := s.getSessionRuntime(sessionKey)
	if !ok || runtime == nil || runtime.client == nil {
		return
	}
	wasRunning := runtime.client.IsRunning()
	runtime.client.StopStream()
	if wasRunning && s.context != nil {
		emitSessionWarn(s.context, sessionKey, "Cancel requested: stopping LLM session")
	}
}

// BindSessionToTab marks a session as bound to a UI tab
func (s *ClientService) BindSessionToTab(sessionID uint) error {
	if sessionID == 0 {
		return fmt.Errorf("session id is required")
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.tabBoundSessions[sessionID] = true
	return nil
}

// UnbindSessionFromTab marks a session as no longer bound to a UI tab (moved to background)
func (s *ClientService) UnbindSessionFromTab(sessionID uint) error {
	if sessionID == 0 {
		return fmt.Errorf("session id is required")
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	delete(s.tabBoundSessions, sessionID)
	return nil
}

// IsSessionInTab checks if a session is currently bound to a UI tab
func (s *ClientService) IsSessionInTab(sessionID uint) bool {
	if sessionID == 0 {
		return false
	}
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.tabBoundSessions[sessionID]
}

// SessionInfo represents information about a generation session
type SessionInfo struct {
	ID           uint   `json:"id"`
	SessionKey   string `json:"sessionKey"`
	ProjectID    uint   `json:"projectId"`
	SourceBranch string `json:"sourceBranch"`
	TargetBranch string `json:"targetBranch"`
	ModelKey     string `json:"modelKey"`
	Provider     string `json:"provider"`
	DocsBranch   string `json:"docsBranch"`
	InTab        bool   `json:"inTab"`
	IsRunning    bool   `json:"isRunning"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// GetAvailableTabSessions returns sessions for a project
func (s *ClientService) GetAvailableTabSessions(projectID uint) ([]SessionInfo, error) {
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}

	sessions, err := s.generationSessions.List(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list generation sessions: %w", err)
	}

	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	availableSessions := make([]SessionInfo, 0)
	for _, session := range sessions {
		sessionKey := makeSessionKey(session.ID)

		// Check if session is bound to a tab
		inTab := s.tabBoundSessions[session.ID]

		// Check if session has a running client
		isRunning := false
		if runtime, ok := s.sessionRuntimes[sessionKey]; ok && runtime != nil && runtime.client != nil {
			isRunning = runtime.client.IsRunning()
		}

		docsBranch := strings.TrimSpace(session.DocsBranch)

		availableSessions = append(availableSessions, SessionInfo{
			ID:           session.ID,
			SessionKey:   sessionKey,
			ProjectID:    projectID,
			SourceBranch: strings.TrimSpace(session.SourceBranch),
			TargetBranch: strings.TrimSpace(session.TargetBranch),
			ModelKey:     strings.TrimSpace(session.ModelKey),
			Provider:     strings.TrimSpace(session.Provider),
			DocsBranch:   docsBranch,
			InTab:        inTab,
			IsRunning:    isRunning,
			CreatedAt:    session.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    session.UpdatedAt.Format(time.RFC3339),
		})
	}

	return availableSessions, nil
}

// ValidateDocsBranch checks if a docsBranch is available for creating a new session
// Returns an error if a session with this docsBranch already exists for this project
func (s *ClientService) ValidateDocsBranch(projectID uint, docsBranch string) error {
	if projectID == 0 {
		return fmt.Errorf("project id is required")
	}
	docsBranch = strings.TrimSpace(docsBranch)
	if docsBranch == "" {
		return fmt.Errorf("docs branch is required")
	}

	// Check if a session with this docsBranch already exists
	existingSession, err := s.generationSessions.GetByDocsBranch(projectID, docsBranch)
	if err != nil {
		return fmt.Errorf("failed to check for existing session: %w", err)
	}

	if existingSession != nil {
		// Session exists - check if it's in a tab or background
		s.sessionMu.RLock()
		inTab := s.tabBoundSessions[existingSession.ID]
		s.sessionMu.RUnlock()

		if inTab {
			return fmt.Errorf("ERR_SESSION_ALREADY_IN_TAB:%d:%s", existingSession.ID, docsBranch)
		}
		// Session exists but not in tab - return the existing session ID so frontend can offer to load it
		return fmt.Errorf("ERR_SESSION_EXISTS:%d:%s", existingSession.ID, docsBranch)
	}

	// No existing session - docsBranch is available
	return nil
}

func resolveBranchHash(repo *git.Repository, branch string) (plumbing.Hash, error) {
	refName := plumbing.NewBranchReferenceName(branch)
	ref, err := repo.Reference(refName, true)
	if err == nil {
		return ref.Hash(), nil
	}
	if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return plumbing.Hash{}, err
	}
	rev, err := repo.ResolveRevision(plumbing.Revision("refs/heads/" + branch))
	if err == nil {
		return *rev, nil
	}
	return plumbing.Hash{}, fmt.Errorf("branch '%s' not found", branch)
}

func resolveDocumentationBase(project *models.RepoLink, repo *git.Repository) (plumbing.Hash, string, error) {
	if project == nil {
		return plumbing.Hash{}, "", fmt.Errorf("project is not configured")
	}
	branch := strings.TrimSpace(project.DocumentationBaseBranch)
	if branch == "" {
		return plumbing.Hash{}, "", fmt.Errorf("documentation base branch is not configured for project '%s'", project.ProjectName)
	}
	hash, err := resolveBranchHash(repo, branch)
	if err != nil {
		return plumbing.Hash{}, "", fmt.Errorf("failed to resolve documentation base branch '%s': %w", branch, err)
	}
	return hash, branch, nil
}

func ensureDocsBranchExists(repo *git.Repository, branch string, baseHash plumbing.Hash) (bool, error) {
	if repo == nil {
		return false, fmt.Errorf("documentation repository is required")
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false, fmt.Errorf("branch name is required")
	}
	refName := plumbing.NewBranchReferenceName(branch)
	if _, err := repo.Reference(refName, true); err == nil {
		return false, nil
	} else if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, fmt.Errorf("failed to resolve documentation branch '%s': %w", branch, err)
	}

	if baseHash == plumbing.ZeroHash {
		return false, fmt.Errorf("cannot create documentation branch '%s': base hash is empty", branch)
	}

	if err := repo.Storer.SetReference(plumbing.NewHashReference(refName, baseHash)); err != nil {
		return false, fmt.Errorf("failed to create documentation branch '%s': %w", branch, err)
	}

	return true, nil
}

func extractPathsFromDiff(diff string) []string {
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+++") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++"))
			if path == "+" {
				continue
			}
			path = strings.TrimPrefix(path, "b/")

			if path == "/dev/null" || path == "" {
				continue
			}
			path = filepath.ToSlash(path)
			seen[path] = struct{}{}
		}
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func newDocRepoConfig(docPath, codeRepoRoot string) (*docRepoConfig, error) {
	absDoc, err := filepath.Abs(docPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve documentation path: %w", err)
	}
	root, ok := utils.FindGitRepoRoot(absDoc)
	if !ok {
		return nil, fmt.Errorf("documentation repository is not a git repository: %s", docPath)
	}
	rel, err := filepath.Rel(root, absDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve documentation path relative to repository root: %w", err)
	}
	if strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("documentation path escapes its repository root")
	}
	rel = filepath.Clean(rel)
	if rel == "" {
		rel = "."
	}
	shared := codeRepoRoot != "" && utils.SamePath(root, codeRepoRoot)
	return &docRepoConfig{
		RepoRoot:       root,
		DocsPath:       absDoc,
		DocsRelative:   rel,
		SharedWithCode: shared,
	}, nil
}

func documentationBranchName(sourceBranch string) string {
	trimmed := strings.TrimSpace(sourceBranch)
	if trimmed == "" {
		return "docs"
	}
	cleaned := strings.ReplaceAll(trimmed, " ", "-")
	return fmt.Sprintf("docs/%s", cleaned)
}

func parseChatMessagesJSON(raw string) []models.ChatMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var msgs []models.ChatMessage
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		return nil
	}
	clean := make([]models.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		role := strings.TrimSpace(strings.ToLower(m.Role))
		content := strings.TrimSpace(m.Content)
		if role != "user" && role != "assistant" {
			continue
		}
		if content == "" {
			continue
		}
		clean = append(clean, models.ChatMessage{
			Role:      role,
			Content:   content,
			CreatedAt: strings.TrimSpace(m.CreatedAt),
		})
	}
	return clean
}

func marshalChatMessages(msgs []models.ChatMessage) string {
	if len(msgs) == 0 {
		return "[]"
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func appendChatMessages(existing []models.ChatMessage, userText, assistantText string) []models.ChatMessage {
	user := strings.TrimSpace(userText)
	assistant := strings.TrimSpace(assistantText)
	if user == "" && assistant == "" {
		return existing
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	updated := make([]models.ChatMessage, 0, len(existing)+2)
	updated = append(updated, existing...)

	if user != "" {
		updated = append(updated, models.ChatMessage{
			Role:      "user",
			Content:   user,
			CreatedAt: now,
		})
	}

	if assistant != "" {
		updated = append(updated, models.ChatMessage{
			Role:      "assistant",
			Content:   assistant,
			CreatedAt: now,
		})
	}

	return updated
}

// loadStoredChatMessagesFromSession extracts chat messages from a session
func (s *ClientService) loadStoredChatMessagesFromSession(session *models.GenerationSession) []models.ChatMessage {
	if session == nil {
		return nil
	}
	return parseChatMessagesJSON(session.ChatMessagesJSON)
}

func newTempRepoDir(ctx context.Context, sessionKey string) (string, func()) {
	tempID := generateUniqueID()
	repoPath := filepath.Join(os.TempDir(), fmt.Sprintf("narrabyte-docs-%s", tempID))
	cleanup := func() {
		if err := os.RemoveAll(repoPath); err != nil {
			emitSessionWarn(ctx, sessionKey, fmt.Sprintf("Failed to cleanup temp directory %s: %v", repoPath, err))
		}
	}
	return repoPath, cleanup
}

func hasDocsChanges(status git.Status, docsRelative string) bool {
	if docsRelative == "." {
		return !status.IsClean()
	}
	base := filepath.ToSlash(filepath.Clean(docsRelative))
	if base == "." {
		return !status.IsClean()
	}
	prefix := base
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for path, st := range status {
		if st == nil {
			continue
		}
		if st.Staging == git.Unmodified && st.Worktree == git.Unmodified {
			continue
		}
		rel := filepath.ToSlash(path)
		if rel == base || strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func addDocsChanges(wt *git.Worktree, docsRelative string) error {
	if docsRelative == "." {
		if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
			return fmt.Errorf("failed to add documentation changes: %w", err)
		}
		return nil
	}
	path := filepath.Clean(docsRelative)
	if err := wt.AddWithOptions(&git.AddOptions{Path: path}); err != nil {
		return fmt.Errorf("failed to add documentation changes: %w", err)
	}
	return nil
}

func collectDocChangedFiles(status git.Status, docsRelative string) []models.DocChangedFile {
	files := make([]models.DocChangedFile, 0)
	base := filepath.ToSlash(filepath.Clean(docsRelative))
	if base == "." {
		base = ""
	}
	prefix := base
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for path, st := range status {
		if st == nil {
			continue
		}
		fmt.Printf("DEBUG: File %s, Worktree: %v, Staging: %v\n", path, st.Worktree, st.Staging)
		if st.Staging == git.Unmodified && st.Worktree == git.Unmodified {
			continue
		}
		rel := filepath.ToSlash(path)
		switch {
		case base == "":
			// Docs root is the repository root; include everything
		case rel == base:
			// Exact directory match (e.g., docs folder itself)
		default:
			if !strings.HasPrefix(rel, prefix) {
				continue
			}
		}
		files = append(files, models.DocChangedFile{
			Path:   rel,
			Status: describeStatus(*st),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

// generateUniqueID creates a unique identifier for temporary directories
func generateUniqueID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// createTempDocRepo creates a temporary clone of the documentation repository
// checked out to the specified branch. Returns the temp workspace (repo root
// and docs path) alongside a cleanup function.
func createTempDocRepo(ctx context.Context, sessionKey string, cfg *docRepoConfig, branch string, baseBranch string, baseHash plumbing.Hash) (workspace tempDocWorkspace, cleanup func(), err error) {
	return createTempDocWorkspace(ctx, sessionKey, cfg, branch, baseBranch, baseHash, false)
}

func copyNarrabyteDir(ctx context.Context, sessionKey string, sourceDocsPath, destDocsPath string) error {
	sourceDir := filepath.Join(sourceDocsPath, ".narrabyte")
	info, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read .narrabyte instructions: %w", err)
	}
	if !info.IsDir() {
		emitSessionWarn(ctx, sessionKey, ".narrabyte exists but is not a directory; skipping copy")
		return nil
	}

	destDir := filepath.Join(destDocsPath, ".narrabyte")
	if err := os.RemoveAll(destDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to reset temp instructions directory: %w", err)
	}

	emitSessionInfo(ctx, sessionKey, "Copying .narrabyte instructions into temporary docs workspace")

	if err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			mode := entryInfo.Mode().Perm()
			if rel == "." {
				return os.MkdirAll(destDir, mode)
			}
			return os.MkdirAll(target, mode)
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			sourceFile.Close()
			return err
		}
		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			targetFile.Close()
			sourceFile.Close()
			return err
		}
		if err := targetFile.Close(); err != nil {
			sourceFile.Close()
			return err
		}
		if err := sourceFile.Close(); err != nil {
			return err
		}
		return os.Chmod(target, entryInfo.Mode())
	}); err != nil {
		return fmt.Errorf("failed to copy .narrabyte instructions: %w", err)
	}

	return nil
}

// removeNarrabyteDir removes the temporary .narrabyte directory if it exists
func removeNarrabyteDir(ctx context.Context, sessionKey string, docsPath string) error {
	dir := filepath.Join(docsPath, ".narrabyte")
	if !utils.DirectoryExists(dir) {
		return nil
	}

	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temp instructions directory: %w", err)
	}
	if utils.DirectoryExists(dir) {
		return fmt.Errorf("failed to remove temp instructions directory: %s", dir)
	}
	emitSessionInfo(ctx, sessionKey, "Removed temporary .narrabyte instructions")
	return nil
}

// propagateDocChanges commits documentation changes in the temp repository and updates
// the branch reference in the main repository to point to the new commit.
// Returns the list of files that were changed (added/modified/etc).
func propagateDocChanges(ctx context.Context, sessionKey string, workspace tempDocWorkspace, mainRepo *git.Repository, branch string, docsRelative string) ([]models.DocChangedFile, error) {
	emitSessionInfo(ctx, sessionKey, "Propagating documentation changes back to main repository")

	// Open temporary repository
	tempRepo, err := git.PlainOpen(workspace.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp repository: %w", err)
	}

	// Get temp repository worktree
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get temp repository worktree: %w", err)
	}

	if err := removeNarrabyteDir(ctx, sessionKey, workspace.docsPath); err != nil {
		return nil, err
	}

	// Check if there are any changes to commit
	status, err := tempWT.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get temp repository status: %w", err)
	}

	changedFiles := collectDocChangedFiles(status, docsRelative)

	if !hasDocsChanges(status, docsRelative) {
		emitSessionInfo(ctx, sessionKey, "No documentation changes to propagate")
		return nil, nil
	}

	// Add all changes
	if err := addDocsChanges(tempWT, docsRelative); err != nil {
		return nil, err
	}

	// Create commit with generated documentation
	commitHash, err := tempWT.Commit("Generated documentation updates", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Narrabyte Documentation Generator",
			Email: "docs@narrabyte.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to commit changes in temp repository: %w", err)
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Created documentation commit: %s", commitHash.String()[:8]))

	// Transfer git objects from temp repository to main repository
	if err := transferGitObjects(ctx, sessionKey, tempRepo, mainRepo, commitHash); err != nil {
		return nil, fmt.Errorf("failed to transfer git objects to main repository: %w", err)
	}

	// Update the branch reference in main repository to point to new commit
	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, commitHash)
	if err := mainRepo.Storer.SetReference(ref); err != nil {
		return nil, fmt.Errorf("failed to update branch '%s' in main repository: %w", branch, err)
	}

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Updated branch '%s' to commit %s", branch, commitHash.String()[:8]))
	return changedFiles, nil
}

// transferGitObjects transfers all git objects (commit, tree, blobs) from source to target repository
// This ensures the target repository has all objects needed to checkout the commit
func transferGitObjects(ctx context.Context, sessionKey string, sourceRepo, targetRepo *git.Repository, commitHash plumbing.Hash) error {
	emitSessionInfo(ctx, sessionKey, "Transferring git objects to main repository")

	// Get the commit object from source repository
	commit, err := sourceRepo.CommitObject(commitHash)
	if err != nil {
		return fmt.Errorf("failed to get commit object: %w", err)
	}

	// Transfer the commit object
	if err := transferObject(sourceRepo, targetRepo, commitHash, plumbing.CommitObject); err != nil {
		return fmt.Errorf("failed to transfer commit object: %w", err)
	}

	// Get and transfer the tree object
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree from commit: %w", err)
	}

	if err := transferTreeRecursively(sourceRepo, targetRepo, tree.Hash); err != nil {
		return fmt.Errorf("failed to transfer tree objects: %w", err)
	}

	// Transfer parent commits if they don't exist in target
	for _, parentHash := range commit.ParentHashes {
		if exists, _ := objectExists(targetRepo, parentHash); !exists {
			if err := transferGitObjects(ctx, sessionKey, sourceRepo, targetRepo, parentHash); err != nil {
				// Log warning but continue - parent might be from a different branch
				emitSessionWarn(ctx, sessionKey, fmt.Sprintf("Could not transfer parent commit %s: %v", parentHash.String()[:8], err))
			}
		}
	}

	emitSessionInfo(ctx, sessionKey, "Git objects transfer completed")
	return nil
}

// transferObject transfers a single git object from source to target repository
func transferObject(sourceRepo, targetRepo *git.Repository, hash plumbing.Hash, objType plumbing.ObjectType) error {
	// Check if object already exists in target
	if exists, _ := objectExists(targetRepo, hash); exists {
		return nil
	}

	// Get encoded object from source
	encodedObj, err := sourceRepo.Storer.EncodedObject(objType, hash)
	if err != nil {
		return fmt.Errorf("failed to get encoded object %s: %w", hash.String(), err)
	}

	// Store encoded object in target
	_, err = targetRepo.Storer.SetEncodedObject(encodedObj)
	return err
}

// transferTreeRecursively transfers a tree and all its contents (blobs and subtrees)
func transferTreeRecursively(sourceRepo, targetRepo *git.Repository, treeHash plumbing.Hash) error {
	// Check if tree already exists
	if exists, _ := objectExists(targetRepo, treeHash); exists {
		return nil
	}

	// Transfer the tree object itself
	if err := transferObject(sourceRepo, targetRepo, treeHash, plumbing.TreeObject); err != nil {
		return err
	}

	// Get tree to iterate through entries
	tree, err := sourceRepo.TreeObject(treeHash)
	if err != nil {
		return fmt.Errorf("failed to get tree object: %w", err)
	}

	// Transfer all entries (blobs and subtrees)
	for _, entry := range tree.Entries {
		switch entry.Mode {
		case filemode.Regular, filemode.Executable, filemode.Symlink:
			// Transfer blob
			if err := transferObject(sourceRepo, targetRepo, entry.Hash, plumbing.BlobObject); err != nil {
				return fmt.Errorf("failed to transfer blob %s: %w", entry.Hash.String(), err)
			}
		case filemode.Dir:
			// Recursively transfer subtree
			if err := transferTreeRecursively(sourceRepo, targetRepo, entry.Hash); err != nil {
				return fmt.Errorf("failed to transfer subtree %s: %w", entry.Hash.String(), err)
			}
		}
	}

	return nil
}

// objectExists checks if an object exists in the repository
func objectExists(repo *git.Repository, hash plumbing.Hash) (bool, error) {
	_, err := repo.Object(plumbing.AnyObject, hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func describeStatus(st git.FileStatus) string {
	code := st.Worktree
	if code == git.Unmodified {
		code = st.Staging
	}
	switch code {
	case git.Added:
		return "added"
	case git.Untracked:
		return "untracked"
	case git.Modified:
		return "modified"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Copied:
		return "copied"
	default:
		return "changed"
	}
}

func (s *ClientService) GenerateDocsFromBranch(projectID uint, branch string, modelKey string, userInstructions string, docsBranchOverride string, sessionKeyOverride string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	branch = strings.TrimSpace(branch)
	modelKey = strings.TrimSpace(modelKey)
	docsBranchOverride = strings.TrimSpace(docsBranchOverride)
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	if modelKey == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Determine docs branch name early
	docsBranch := documentationBranchName(branch)
	if docsBranchOverride != "" {
		docsBranch = docsBranchOverride
	}

	// Check if a session with this docsBranch already exists
	existingSession, err := s.generationSessions.GetByDocsBranch(projectID, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing session: %w", err)
	}
	if existingSession != nil {
		return nil, fmt.Errorf("ERR_SESSION_EXISTS:a session with docsBranch '%s' already exists (ID: %d)", docsBranch, existingSession.ID)
	}

	runtime, modelInfo, err := s.newSessionRuntime(modelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	project, codeRoot, docCfg, err := s.prepareProjectRepos(projectID)
	if err != nil {
		return nil, err
	}

	// Open documentation repo
	docRepo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open documentation repository: %w", err)
	}

	var (
		baseHash   plumbing.Hash
		baseBranch string
	)
	if docCfg.SharedWithCode {
		baseBranch = branch
		baseHash, err = resolveBranchHash(docRepo, branch)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve branch '%s': %w", branch, err)
		}
	} else {
		baseHash, baseBranch, err = resolveDocumentationBase(project, docRepo)
		if err != nil {
			return nil, err
		}
	}

	providerID := strings.TrimSpace(runtime.providerID)
	if providerID == "" && modelInfo != nil {
		providerID = strings.TrimSpace(modelInfo.ProviderID)
	}

	// Create session record to get ID
	session := &models.GenerationSession{
		ProjectID:    projectID,
		SourceBranch: branch,
		TargetBranch: baseBranch,
		Provider:     providerID,
		ModelKey:     runtime.modelKey,
		DocsBranch:   docsBranch,
	}
	session, err = s.generationSessions.Create(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sessionKey := resolveSessionKey(sessionKeyOverride, session.ID)
	s.setSessionRuntime(sessionKey, runtime)
	runtime.targetBranch = baseBranch

	if err := s.ensureDocsBranchAvailable(docRepo, docsBranch); err != nil {
		_ = s.generationSessions.DeleteByID(session.ID)
		return nil, err
	}

	// Mark this docs branch as in-progress to prevent concurrent generations
	if err := s.markDocsBranchInProgress(docsBranch); err != nil {
		return nil, err
	}
	defer s.unmarkDocsBranchInProgress(docsBranch)

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"GenerateDocsFromBranch: starting for project %s on branch %s using %s via %s into %s",
		project.ProjectName, branch, runtime.modelDisplay, runtime.providerLabel, docsBranch,
	))

	refName := plumbing.NewBranchReferenceName(docsBranch)
	if _, err := docRepo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			// Create docs branch pointing to base commit
			if err := docRepo.Storer.SetReference(plumbing.NewHashReference(refName, baseHash)); err != nil {
				return nil, fmt.Errorf("failed to create documentation branch '%s': %w", docsBranch, err)
			}
			emitSessionInfo(ctx, sessionKey, fmt.Sprintf("Created docs branch '%s' from '%s'", docsBranch, baseBranch))
		} else {
			return nil, fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
		}
	}

	tempWorkspace, cleanup, err := createTempDocRepoAtBranchHead(ctx, sessionKey, docCfg, docsBranch, baseBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup()

	emitSessionInfo(ctx, sessionKey, fmt.Sprintf(
		"GenerateDocsFromBranch: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	))

	// New session, no existing chat
	var existingChat []models.ChatMessage

	streamCtx := runtime.client.StartStream(ctx, sessionKey)
	defer runtime.client.StopStream()

	llmResult, err := runtime.client.DocRefine(streamCtx, &client.DocRefineRequest{
		ProjectName:          project.ProjectName,
		CodebasePath:         codeRoot,
		DocumentationPath:    tempWorkspace.docsPath,
		DocumentationRelPath: docCfg.DocsRelative,
		SourceBranch:         branch,
		Instruction:          userInstructions,
	})
	if err != nil {
		return nil, err
	}

	assistantSummary := ""
	if llmResult != nil {
		assistantSummary = llmResult.Summary
	}
	chatMessages := appendChatMessages(existingChat, userInstructions, assistantSummary)
	chatMessagesJSON := marshalChatMessages(chatMessages)

	// Save conversation history to session
	if runtime.client != nil {
		if jsonStr, err := runtime.client.ConversationHistoryJSON(); err == nil {
			_ = s.generationSessions.UpdateByID(session.ID, map[string]interface{}{
				"messages_json":      jsonStr,
				"chat_messages_json": chatMessagesJSON,
			})
		}
	}

	// Propagate changes from temporary repository back to main repository
	files, err := propagateDocChanges(ctx, sessionKey, tempWorkspace, docRepo, docsBranch, docCfg.DocsRelative)
	if err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	emitSessionInfo(ctx, sessionKey, "GenerateDocsFromBranch: completed")

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
		SessionID:      session.ID,
		SessionKey:     sessionKey,
		Branch:         branch,
		TargetBranch:   baseBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        summary,
		ChatMessages:   chatMessages,
	}, nil
}
