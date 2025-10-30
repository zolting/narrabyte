package services

import (
	"bufio"
	"context"
	"crypto/rand"
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
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// On pourrait lowkey rendre ca plus generique pour n'importe quel client
// Interface pour clients?
type ClientService struct {
	LLMClient          *client.LLMClient
	currentModelKey    string
	context            context.Context
	repoLinks          RepoLinkService
	gitService         *GitService
	keyringService     *KeyringService
	generationSessions GenerationSessionService
	modelConfigs       ModelConfigService
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
		repoLinks:          repoLinks,
		gitService:         gitService,
		keyringService:     keyringService,
		generationSessions: genSessions,
		modelConfigs:       modelConfigs,
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

// InitializeLLMClient initializes the LLM client for the specified model.
func (s *ClientService) InitializeLLMClient(modelKey string) error {
	if s.context == nil {
		return fmt.Errorf("client service not initialized")
	}
	if s.keyringService == nil {
		return fmt.Errorf("keyring service not configured")
	}

	model, err := s.modelConfigs.GetModel(modelKey)
	if err != nil {
		return err
	}
	if model == nil {
		return fmt.Errorf("model %s not found", modelKey)
	}
	if !model.Enabled {
		return fmt.Errorf("model %s is disabled", model.DisplayName)
	}

	providerID := strings.TrimSpace(model.ProviderID)
	if providerID == "" {
		return fmt.Errorf("model %s is missing provider information", model.DisplayName)
	}

	apiKey, err := s.keyringService.GetApiKey(providerID)
	if err != nil {
		return fmt.Errorf("failed to get API key for %s: %w", providerID, err)
	}
	if apiKey == "" {
		return fmt.Errorf("API key for %s is not configured", providerID)
	}

	var llmClient *client.LLMClient
	switch providerID {
	case "anthropic":
		llmClient, err = client.NewClaudeClient(s.context, apiKey, client.ClaudeModelOptions{
			Model:    model.APIName,
			Thinking: model.Thinking != nil && *model.Thinking,
		})
	case "openai":
		llmClient, err = client.NewOpenAIClient(s.context, apiKey, client.OpenAIModelOptions{
			Model:           model.APIName,
			ReasoningEffort: model.ReasoningEffort,
		})
	case "gemini":
		llmClient, err = client.NewGeminiClient(s.context, apiKey, client.GeminiModelOptions{
			Model:    model.APIName,
			Thinking: model.Thinking != nil && *model.Thinking,
		})
	default:
		return fmt.Errorf("unsupported provider: %s", providerID)
	}

	if err != nil {
		return fmt.Errorf("failed to create %s client: %w", providerID, err)
	}

	s.LLMClient = llmClient
	s.currentModelKey = modelKey
	return nil
}

func (s *ClientService) GenerateDocs(projectID uint, sourceBranch string, targetBranch string, modelKey string, userInstructions string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	modelKey = strings.TrimSpace(modelKey)
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

	modelInfo, err := s.modelConfigs.GetModel(modelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model: %w", err)
	}
	if modelInfo == nil {
		return nil, fmt.Errorf("model not found: %s", modelKey)
	}
	if !modelInfo.Enabled {
		return nil, fmt.Errorf("model %s is disabled", modelInfo.DisplayName)
	}
	providerID := strings.TrimSpace(modelInfo.ProviderID)
	providerName := strings.TrimSpace(modelInfo.ProviderName)
	providerLabel := providerName
	if providerLabel == "" {
		providerLabel = providerID
	}

	// Initialize LLM client with the specified model
	if err := s.InitializeLLMClient(modelKey); err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"GenerateDocs: starting for project %s (%s -> %s) using %s via %s",
		project.ProjectName, targetBranch, sourceBranch, modelInfo.DisplayName, providerLabel,
	)))

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
	if !utils.HasGitRepo(codeRepoPath) {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	if !utils.HasGitRepo(docRepoPath) {
		return nil, fmt.Errorf("documentation repository is not a git repository: %s", docRepoPath)
	}

	codeRootAbs, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return nil, err
	}
	codeRepoRoot, ok := utils.FindGitRepoRoot(codeRootAbs)
	if !ok {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	codeRoot := codeRepoRoot

	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
	if err != nil {
		return nil, err
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
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocs: no code changes detected between branches"))
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
		events.Emit(ctx, events.LLMEventTool, events.NewWarn("Documentation repository has uncommitted changes - these will be preserved"))
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

	docsBranch := documentationBranchName(sourceBranch)

	// Create temporary documentation repository (isolated from working directory)
	tempWorkspace, cleanup, err := createTempDocRepo(ctx, docCfg, docsBranch, baseBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup() // Always cleanup temp directory

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"GenerateDocs: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	)))

	streamCtx := s.LLMClient.StartStream(ctx)
	defer s.LLMClient.StopStream()

	// Use temporary documentation root for LLM operations
	llmResult, err := s.LLMClient.GenerateDocs(streamCtx, &client.DocGenerationRequest{
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
	if err := propagateDocChanges(ctx, tempWorkspace, docRepo, docsBranch, docCfg.DocsRelative); err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	if s.LLMClient != nil {
		if jsonStr, err := s.LLMClient.ConversationHistoryJSON(); err == nil {
			_, _ = s.generationSessions.Upsert(projectID, sourceBranch, targetBranch, modelKey, providerID, jsonStr)
		}
	}

	// Get status from temporary repository to report changed files
	tempRepo, err := git.PlainOpen(tempWorkspace.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp repository for status: %w", err)
	}
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get temp repository worktree for status: %w", err)
	}
	docStatus, err := tempWT.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation repo status after generation: %w", err)
	}
	files := collectDocChangedFiles(docStatus, docCfg.DocsRelative)

	// Generate diff between the new docs branch and its base branch
	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocs: completed"))

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
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
// created for a given source branch ("docs/<sourceBranch>"). It reuses the
// same toolset as GenerateDocs but focuses on targeted edits directed by the
// user's request.
func (s *ClientService) RefineDocs(projectID uint, sourceBranch string, instruction string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	sourceBranch = strings.TrimSpace(sourceBranch)
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if sourceBranch == "" {
		return nil, fmt.Errorf("source branch is required")
	}
	if strings.TrimSpace(instruction) == "" {
		return nil, fmt.Errorf("instruction is required")
	}

	docsBranch := documentationBranchName(sourceBranch)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"RefineDocs: starting for project %d (%s)",
		projectID, docsBranch,
	)))

	// Ensure LLM client is initialized - try to initialize from session if needed
	activeModelKey := strings.TrimSpace(s.currentModelKey)
	if s.LLMClient == nil || activeModelKey == "" {
		sessions, err := s.generationSessions.List(projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to load generation sessions: %w", err)
		}
		for _, sess := range sessions {
			if strings.TrimSpace(sess.SourceBranch) != sourceBranch {
				continue
			}
			candidateModelKey := strings.TrimSpace(sess.ModelKey)
			if candidateModelKey == "" && strings.TrimSpace(sess.Provider) != "" {
				if fallback, fbErr := s.findDefaultModelForProvider(strings.TrimSpace(sess.Provider)); fbErr == nil && fallback != nil {
					candidateModelKey = fallback.Key
				}
			}
			if candidateModelKey == "" {
				continue
			}
			if initErr := s.InitializeLLMClient(candidateModelKey); initErr != nil {
				return nil, fmt.Errorf("failed to initialize LLM client from session: %w", initErr)
			}
			modelInfo, modelErr := s.modelConfigs.GetModel(candidateModelKey)
			if modelErr == nil && modelInfo != nil {
				label := modelInfo.ProviderName
				if strings.TrimSpace(label) == "" {
					label = modelInfo.ProviderID
				}
				events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Initialized %s via %s from session", modelInfo.DisplayName, label)))
			}
			activeModelKey = candidateModelKey
			break
		}
		if s.LLMClient == nil || activeModelKey == "" {
			return nil, fmt.Errorf("LLM client not initialized - please run GenerateDocs first or restore a session")
		}
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

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
	if !utils.HasGitRepo(codeRepoPath) {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	if !utils.HasGitRepo(docRepoPath) {
		return nil, fmt.Errorf("documentation repository is not a git repository: %s", docRepoPath)
	}

	codeRootAbs, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return nil, err
	}
	codeRepoRoot, ok := utils.FindGitRepoRoot(codeRootAbs)
	if !ok {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	codeRoot := codeRepoRoot

	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
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
	refName := plumbing.NewBranchReferenceName(docsBranch)
	// Ensure the docs branch exists in the main repo; if not, create it off base
	if _, err := docRepo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			// Create docs branch pointing to base commit
			if err := docRepo.Storer.SetReference(plumbing.NewHashReference(refName, baseHash)); err != nil {
				return nil, fmt.Errorf("failed to create documentation branch '%s': %w", docsBranch, err)
			}
			events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("RefineDocs: created missing docs branch '%s' from '%s'", docsBranch, baseBranch)))
		} else {
			return nil, fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
		}
	}

	// Create a temporary workspace checked out to the current docs branch head
	tempWorkspace, cleanup, err := createTempDocRepoAtBranchHead(ctx, docCfg, docsBranch, baseBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup()

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"RefineDocs: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	)))

	streamCtx := s.LLMClient.StartStream(ctx)
	defer s.LLMClient.StopStream()

	if s.LLMClient != nil && !s.LLMClient.HasConversationHistory() {
		sessions, err := s.generationSessions.List(projectID)
		if err == nil {
			for _, sess := range sessions {
				if strings.TrimSpace(sess.SourceBranch) == sourceBranch {
					_ = s.LLMClient.LoadConversationHistoryJSON(sess.MessagesJSON)
					if activeModelKey == "" && strings.TrimSpace(sess.ModelKey) != "" {
						activeModelKey = strings.TrimSpace(sess.ModelKey)
					}
					break
				}
			}
		}
	}

	// Run the refinement agent focused on applying user edits
	llmResult, err := s.LLMClient.DocRefine(streamCtx, &client.DocRefineRequest{
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

	if s.LLMClient != nil {
		if jsonStr, err := s.LLMClient.ConversationHistoryJSON(); err == nil {
			modelKeyForSession := strings.TrimSpace(activeModelKey)
			providerForSession := ""
			if modelKeyForSession != "" {
				if modelInfo, getErr := s.modelConfigs.GetModel(modelKeyForSession); getErr == nil && modelInfo != nil {
					providerForSession = strings.TrimSpace(modelInfo.ProviderID)
				}
			}
			if modelKeyForSession == "" || providerForSession == "" {
				if sess, getErr := s.generationSessions.Get(projectID, sourceBranch, baseBranch); getErr == nil && sess != nil {
					if modelKeyForSession == "" {
						modelKeyForSession = strings.TrimSpace(sess.ModelKey)
					}
					if providerForSession == "" {
						providerForSession = strings.TrimSpace(sess.Provider)
					}
				}
			}
			if modelKeyForSession != "" && providerForSession == "" {
				if modelInfo, getErr := s.modelConfigs.GetModel(modelKeyForSession); getErr == nil && modelInfo != nil {
					providerForSession = strings.TrimSpace(modelInfo.ProviderID)
				}
			}
			if modelKeyForSession != "" && providerForSession != "" {
				_, _ = s.generationSessions.Upsert(projectID, sourceBranch, baseBranch, modelKeyForSession, providerForSession, jsonStr)
			}
		}
	}

	// Propagate changes back to the main documentation repository
	if err := propagateDocChanges(ctx, tempWorkspace, docRepo, docsBranch, docCfg.DocsRelative); err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	// Compute changed files from the temp repository status (post-run)
	tempRepo, err := git.PlainOpen(tempWorkspace.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp repository for status: %w", err)
	}
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get temp repository worktree for status: %w", err)
	}
	docStatus, err := tempWT.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation repo status after refinement: %w", err)
	}
	files := collectDocChangedFiles(docStatus, docCfg.DocsRelative)

	// Update diff between base branch and docs branch for UI preview
	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("RefineDocs: completed"))

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
		Branch:         sourceBranch,
		TargetBranch:   baseBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        summary,
	}, nil
}

// MergeDocsIntoSource fast-forwards the source code branch to include the latest
// documentation commit generated on docs/<sourceBranch>. Only supported when
// documentation lives within the code repository.
func (s *ClientService) MergeDocsIntoSource(projectID uint, sourceBranch string) error {
	ctx := s.context
	if ctx == nil {
		return fmt.Errorf("client service not initialized")
	}
	sourceBranch = strings.TrimSpace(sourceBranch)
	if projectID == 0 {
		return fmt.Errorf("project id is required")
	}
	if sourceBranch == "" {
		return fmt.Errorf("source branch is required")
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return fmt.Errorf("project not found")
	}

	codeRepoPath := strings.TrimSpace(project.CodebaseRepo)
	docRepoPath := strings.TrimSpace(project.DocumentationRepo)
	if codeRepoPath == "" || docRepoPath == "" {
		return fmt.Errorf("project repositories are not configured")
	}
	if !utils.DirectoryExists(codeRepoPath) {
		return fmt.Errorf("codebase repository path does not exist: %s", codeRepoPath)
	}
	if !utils.DirectoryExists(docRepoPath) {
		return fmt.Errorf("documentation repository path does not exist: %s", docRepoPath)
	}
	if !utils.HasGitRepo(codeRepoPath) {
		return fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	if !utils.HasGitRepo(docRepoPath) {
		return fmt.Errorf("documentation repository is not a git repository: %s", docRepoPath)
	}

	codeRootAbs, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return err
	}
	codeRepoRoot, ok := utils.FindGitRepoRoot(codeRootAbs)
	if !ok {
		return fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}

	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
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

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"MergeDocs: fast-forwarded '%s' to include documentation commit %s",
		sourceBranch,
		docRef.Hash().String()[:8],
	)))
	return nil
}

// createTempDocRepoAtBranchHead clones the documentation repository into a temp directory
// and checks out the specified branch at its current HEAD.
func createTempDocRepoAtBranchHead(ctx context.Context, cfg *docRepoConfig, branch string, baseBranch string, baseHash plumbing.Hash) (workspace tempDocWorkspace, cleanup func(), err error) {
	if cfg == nil {
		return tempDocWorkspace{}, nil, fmt.Errorf("documentation repository configuration is required")
	}
	repoPath, cleanup := newTempRepoDir(ctx)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Creating temporary docs workspace at %s", repoPath)))
	cloneOpts := &git.CloneOptions{
		URL:           cfg.RepoRoot,
		Depth:         1,
		Progress:      nil,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	tempRepo, err := git.PlainClone(repoPath, false, cloneOpts)
	if err != nil {
		cleanup()
		repoPath, cleanup = newTempRepoDir(ctx)
		events.Emit(ctx, events.LLMEventTool, events.NewWarn("Shallow clone failed; retrying with full clone"))
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
				events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Creating docs branch '%s' from base '%s'", branch, baseBranch)))
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

	tempDocsPath := repoPath
	if cfg.DocsRelative != "." {
		tempDocsPath = filepath.Join(repoPath, cfg.DocsRelative)
	}

	if err := copyNarrabyteDir(ctx, cfg.DocsPath, tempDocsPath); err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, err
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Temporary docs workspace ready: branch '%s' at %s", branch, repoPath)))
	return tempDocWorkspace{repoPath: repoPath, docsPath: tempDocsPath}, cleanup, nil
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

func (s *ClientService) CommitDocs(projectID uint, branch string, files []string) error {
	ctx := s.context
	if ctx == nil {
		return fmt.Errorf("client service not initialized")
	}
	if projectID == 0 {
		return fmt.Errorf("project id is required")
	}
	sourceBranch := strings.TrimSpace(branch)
	if sourceBranch == "" {
		return fmt.Errorf("branch is required")
	}
	docsBranch := documentationBranchName(sourceBranch)
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

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"CommitDocs: staging %d documentation file(s) for branch '%s'",
		len(normalized), docsBranch,
	)))

	if err := s.gitService.StageFiles(repo, normalized); err != nil {
		return fmt.Errorf("failed to stage documentation changes: %w", err)
	}

	message := fmt.Sprintf("Add documentation for %s", docsBranch)
	if _, err := s.gitService.Commit(repo, message); err != nil {
		return fmt.Errorf("failed to commit documentation changes: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"CommitDocs: committed documentation updates to '%s'",
		docsBranch,
	)))

	return nil
}

// LoadGenerationSession restores an existing generation session for a specific branch pair and computes the current
// documentation state (diff, changed files, etc.) for display in the UI.
func (s *ClientService) LoadGenerationSession(projectID uint, sourceBranch, targetBranch string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}

	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	if sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}

	session, err := s.generationSessions.Get(projectID, sourceBranch, targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get generation session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("no active generation session found for project %d (%s -> %s)", projectID, sourceBranch, targetBranch)
	}

	// Initialize LLM client with the model stored in the session (falls back to provider if necessary)
	modelKey := strings.TrimSpace(session.ModelKey)
	providerID := strings.TrimSpace(session.Provider)
	if modelKey == "" && providerID != "" {
		if fallback, fbErr := s.findDefaultModelForProvider(providerID); fbErr == nil && fallback != nil {
			modelKey = fallback.Key
		}
	}
	if modelKey == "" {
		return nil, fmt.Errorf("session is missing model information")
	}
	if err := s.InitializeLLMClient(modelKey); err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	if modelInfo, getErr := s.modelConfigs.GetModel(modelKey); getErr == nil && modelInfo != nil {
		label := modelInfo.ProviderName
		if strings.TrimSpace(label) == "" {
			label = modelInfo.ProviderID
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Initialized %s via %s from session", modelInfo.DisplayName, label)))
	}
	if session.ModelKey == "" {
		if jsonStr, err := s.LLMClient.ConversationHistoryJSON(); err == nil {
			if modelInfo, getErr := s.modelConfigs.GetModel(modelKey); getErr == nil && modelInfo != nil {
				_, _ = s.generationSessions.Upsert(projectID, sourceBranch, targetBranch, modelKey, modelInfo.ProviderID, jsonStr)
			}
		}
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"LoadSession: restoring documentation session for project %s (%s → %s)",
		project.ProjectName, targetBranch, sourceBranch,
	)))

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

	docsBranch := documentationBranchName(sourceBranch)
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

	if session.MessagesJSON != "" && s.LLMClient != nil {
		if err := s.LLMClient.LoadConversationHistoryJSON(session.MessagesJSON); err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf(
				"Failed to restore conversation history: %v", err,
			)))
		} else {
			events.Emit(ctx, events.LLMEventTool, events.NewInfo("Restored LLM conversation history"))
		}
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("LoadSession: session restored successfully"))

	return &models.DocGenerationResult{
		Branch:         sourceBranch,
		TargetBranch:   targetBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        "Restored from previous session",
	}, nil
}

func (s *ClientService) StopStream() {
	if s == nil || s.LLMClient == nil {
		return
	}
	wasRunning := s.LLMClient.IsRunning()
	s.LLMClient.StopStream()
	if wasRunning && s.context != nil {
		events.Emit(s.context, events.LLMEventTool, events.NewWarn("Cancel requested: stopping LLM session"))
	}
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

func newTempRepoDir(ctx context.Context) (string, func()) {
	tempID := generateUniqueID()
	repoPath := filepath.Join(os.TempDir(), fmt.Sprintf("narrabyte-docs-%s", tempID))
	cleanup := func() {
		if err := os.RemoveAll(repoPath); err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Failed to cleanup temp directory %s: %v", repoPath, err)))
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
func createTempDocRepo(ctx context.Context, cfg *docRepoConfig, branch string, baseBranch string, baseHash plumbing.Hash) (workspace tempDocWorkspace, cleanup func(), err error) {
	if cfg == nil {
		return tempDocWorkspace{}, nil, fmt.Errorf("documentation repository configuration is required")
	}
	repoPath, cleanup := newTempRepoDir(ctx)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Creating temporary docs workspace at %s", repoPath)))

	cloneOpts := &git.CloneOptions{
		URL:          cfg.RepoRoot,
		Depth:        1,
		Progress:     nil,
		SingleBranch: true,
	}
	if strings.TrimSpace(baseBranch) != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(baseBranch)
	}
	tempRepo, err := git.PlainClone(repoPath, false, cloneOpts)
	if err != nil {
		cleanup()
		repoPath, cleanup = newTempRepoDir(ctx)
		events.Emit(ctx, events.LLMEventTool, events.NewWarn("Shallow clone failed; retrying with full clone"))
		tempRepo, err = git.PlainClone(repoPath, false, &git.CloneOptions{URL: cfg.RepoRoot, Progress: nil})
		if err != nil {
			cleanup()
			return tempDocWorkspace{}, nil, fmt.Errorf("failed to clone repository to temp location: %w", err)
		}
	}

	tempWT, err := tempRepo.Worktree()
	if err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, fmt.Errorf("failed to get temp repository worktree: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, baseHash)
	if err := tempRepo.Storer.SetReference(ref); err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, fmt.Errorf("failed to create branch '%s' in temp repo: %w", branch, err)
	}

	if err := tempWT.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, err)
	}

	tempDocsPath := repoPath
	if cfg.DocsRelative != "." {
		tempDocsPath = filepath.Join(repoPath, cfg.DocsRelative)
	}

	if err := copyNarrabyteDir(ctx, cfg.DocsPath, tempDocsPath); err != nil {
		cleanup()
		return tempDocWorkspace{}, nil, err
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Temporary docs workspace ready: branch '%s' at %s", branch, repoPath)))
	return tempDocWorkspace{repoPath: repoPath, docsPath: tempDocsPath}, cleanup, nil
}

func copyNarrabyteDir(ctx context.Context, sourceDocsPath, destDocsPath string) error {
	sourceDir := filepath.Join(sourceDocsPath, ".narrabyte")
	info, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read .narrabyte instructions: %w", err)
	}
	if !info.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn(".narrabyte exists but is not a directory; skipping copy"))
		return nil
	}

	destDir := filepath.Join(destDocsPath, ".narrabyte")
	if err := os.RemoveAll(destDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to reset temp instructions directory: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Copying .narrabyte instructions into temporary docs workspace"))

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

func removeNarrabyteDir(ctx context.Context, docsPath string) error {
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
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Removed temporary .narrabyte instructions"))
	return nil
}

// propagateDocChanges commits documentation changes in the temp repository and updates
// the branch reference in the main repository to point to the new commit.
func propagateDocChanges(ctx context.Context, workspace tempDocWorkspace, mainRepo *git.Repository, branch string, docsRelative string) error {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Propagating documentation changes back to main repository"))

	// Open temporary repository
	tempRepo, err := git.PlainOpen(workspace.repoPath)
	if err != nil {
		return fmt.Errorf("failed to open temp repository: %w", err)
	}

	// Get temp repository worktree
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get temp repository worktree: %w", err)
	}

	if err := removeNarrabyteDir(ctx, workspace.docsPath); err != nil {
		return err
	}

	// Check if there are any changes to commit
	status, err := tempWT.Status()
	if err != nil {
		return fmt.Errorf("failed to get temp repository status: %w", err)
	}

	if !hasDocsChanges(status, docsRelative) {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("No documentation changes to propagate"))
		return nil
	}

	// Add all changes
	if err := addDocsChanges(tempWT, docsRelative); err != nil {
		return err
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
		return fmt.Errorf("failed to commit changes in temp repository: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Created documentation commit: %s", commitHash.String()[:8])))

	// Transfer git objects from temp repository to main repository
	if err := transferGitObjects(ctx, tempRepo, mainRepo, commitHash); err != nil {
		return fmt.Errorf("failed to transfer git objects to main repository: %w", err)
	}

	// Update the branch reference in main repository to point to new commit
	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, commitHash)
	if err := mainRepo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to update branch '%s' in main repository: %w", branch, err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Updated branch '%s' to commit %s", branch, commitHash.String()[:8])))
	return nil
}

// transferGitObjects transfers all git objects (commit, tree, blobs) from source to target repository
// This ensures the target repository has all objects needed to checkout the commit
func transferGitObjects(ctx context.Context, sourceRepo, targetRepo *git.Repository, commitHash plumbing.Hash) error {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Transferring git objects to main repository"))

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
			if err := transferGitObjects(ctx, sourceRepo, targetRepo, parentHash); err != nil {
				// Log warning but continue - parent might be from a different branch
				events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Could not transfer parent commit %s: %v", parentHash.String()[:8], err)))
			}
		}
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Git objects transfer completed"))
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

func (s *ClientService) GenerateDocsFromBranch(projectID uint, branch string, modelKey string, userInstructions string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	branch = strings.TrimSpace(branch)
	modelKey = strings.TrimSpace(modelKey)
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	if modelKey == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Initialize LLM client with the specified model
	if err := s.InitializeLLMClient(modelKey); err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	project, err := s.repoLinks.Get(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

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
	if !utils.HasGitRepo(codeRepoPath) {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	if !utils.HasGitRepo(docRepoPath) {
		return nil, fmt.Errorf("documentation repository is not a git repository: %s", docRepoPath)
	}

	codeRootAbs, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return nil, err
	}
	codeRepoRoot, ok := utils.FindGitRepoRoot(codeRootAbs)
	if !ok {
		return nil, fmt.Errorf("codebase repository is not a git repository: %s", codeRepoPath)
	}
	codeRoot := codeRepoRoot

	docCfg, err := newDocRepoConfig(docRepoPath, codeRepoRoot)
	if err != nil {
		return nil, err
	}

	docRepo, err := s.gitService.Open(docCfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open documentation repository: %w", err)
	}

	var (
		baseHash   plumbing.Hash
		baseBranch string
	)
	if docCfg.SharedWithCode {
		// When docs live with code, base is the same branch
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

	docsBranch := documentationBranchName(branch)
	refName := plumbing.NewBranchReferenceName(docsBranch)
	// Ensure docs branch exists; create off base if missing
	if _, err := docRepo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			if err := docRepo.Storer.SetReference(plumbing.NewHashReference(refName, baseHash)); err != nil {
				return nil, fmt.Errorf("failed to create documentation branch '%s': %w", docsBranch, err)
			}
			events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Created docs branch '%s' from '%s'", docsBranch, baseBranch)))
		} else {
			return nil, fmt.Errorf("failed to resolve documentation branch '%s': %w", docsBranch, err)
		}
	}

	// Create temporary documentation workspace checked out at current docs branch head
	tempWorkspace, cleanup, err := createTempDocRepoAtBranchHead(ctx, docCfg, docsBranch, baseBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup()

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"GenerateDocsFromBranch: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	)))

	streamCtx := s.LLMClient.StartStream(ctx)
	defer s.LLMClient.StopStream()

	// Invoke refinement agent with user-provided instruction
	llmResult, err := s.LLMClient.DocRefine(streamCtx, &client.DocRefineRequest{
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

	// Persist conversation for future refinements
	if s.LLMClient != nil {
		if jsonStr, err := s.LLMClient.ConversationHistoryJSON(); err == nil {
			modelInfo, _ := s.modelConfigs.GetModel(modelKey)
			provider := ""
			if modelInfo != nil {
				provider = strings.TrimSpace(modelInfo.ProviderID)
			}
			if provider != "" {
				_, _ = s.generationSessions.Upsert(projectID, branch, baseBranch, modelKey, provider, jsonStr)
			}
		}
	}

	// Propagate changes back to main documentation repository
	if err := propagateDocChanges(ctx, tempWorkspace, docRepo, docsBranch, docCfg.DocsRelative); err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	// Collect changed files and diff for UI
	tempRepo, err := git.PlainOpen(tempWorkspace.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp repository for status: %w", err)
	}
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get temp repository worktree for status: %w", err)
	}
	docStatus, err := tempWT.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation repo status after generation: %w", err)
	}
	files := collectDocChangedFiles(docStatus, docCfg.DocsRelative)

	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, baseBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocsFromBranch: completed"))

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
		Branch:         branch,
		TargetBranch:   baseBranch,
		DocsBranch:     docsBranch,
		DocsInCodeRepo: docCfg.SharedWithCode,
		Files:          files,
		Diff:           docDiff,
		Summary:        summary,
	}, nil
}
