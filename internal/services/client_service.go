package services

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"narrabyte/internal/events"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/models"
	"narrabyte/internal/utils"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/cloudwego/eino/schema"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// On pourrait lowkey rendre ca plus generique pour n'importe quel client
// Interface pour clients?
type ClientService struct {
	OpenAIClient *client.OpenAIClient
	context      context.Context
	repoLinks    RepoLinkService
	gitService   *GitService
}

func (s *ClientService) Startup(ctx context.Context) error {
	s.context = ctx
	if s.repoLinks == nil {
		return fmt.Errorf("repo link service not configured")
	}
	if s.gitService == nil {
		return fmt.Errorf("git service not configured")
	}

	err := utils.LoadEnv()
	if err != nil {
		return err
	}
	key := os.Getenv("OPENAI_API_KEY")

	temp, err := client.NewOpenAIClient(ctx, key)

	if err != nil {
		return err
	}

	s.OpenAIClient = temp

	return nil
}

func NewClientService(repoLinks RepoLinkService, gitService *GitService) *ClientService {
	return &ClientService{
		repoLinks:  repoLinks,
		gitService: gitService,
	}
}

func (s *ClientService) ExploreDemo() (string, error) {
	root, err := utils.FindProjectRoot()
	if err != nil {
		return "", err
	}

	ctx := s.OpenAIClient.StartStream(s.context)
	defer s.OpenAIClient.StopStream()

	result, err := s.OpenAIClient.ExploreCodebaseDemo(ctx, root)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (s *ClientService) GenerateDocs(projectID uint, sourceBranch, targetBranch string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}
	if sourceBranch == targetBranch {
		return nil, fmt.Errorf("source and target branches must differ")
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"GenerateDocs: starting for project %d (%s -> %s)",
		projectID, targetBranch, sourceBranch,
	)))

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

	codeRoot, err := filepath.Abs(codeRepoPath)
	if err != nil {
		return nil, err
	}
	docRoot, err := filepath.Abs(docRepoPath)
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

	docRepo, err := s.gitService.Open(docRoot)
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
	if !status.IsClean() {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn("Documentation repository has uncommitted changes - these will be preserved"))
	}

	baseHash, err := ensureBaseBranch(docRepo, targetBranch)
	if err != nil {
		return nil, err
	}

	docsBranch := "docs-" + sourceBranch

	// Create temporary documentation repository (isolated from working directory)
	tempDocRoot, cleanup, err := createTempDocRepo(ctx, docRoot, docsBranch, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary documentation workspace: %w", err)
	}
	defer cleanup() // Always cleanup temp directory

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"GenerateDocs: temporary documentation workspace ready for branch '%s'",
		docsBranch,
	)))

	streamCtx := s.OpenAIClient.StartStream(ctx)
	defer s.OpenAIClient.StopStream()

	// Use temporary documentation root for LLM operations
	llmResult, err := s.OpenAIClient.GenerateDocs(streamCtx, &client.DocGenerationRequest{
		ProjectName:       project.ProjectName,
		CodebasePath:      codeRoot,
		DocumentationPath: tempDocRoot, // Use temporary workspace
		SourceBranch:      sourceBranch,
		TargetBranch:      targetBranch,
		SourceCommit:      sourceHash.String(),
		Diff:              diffText,
		ChangedFiles:      changedFiles,
	})
	if err != nil {
		return nil, err
	}

	// Propagate changes from temporary repository back to main repository
	if err := propagateDocChanges(ctx, tempDocRoot, docRepo, docsBranch); err != nil {
		return nil, fmt.Errorf("failed to propagate documentation changes: %w", err)
	}

	// Get status from temporary repository to report changed files
	tempRepo, err := git.PlainOpen(tempDocRoot)
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
	var files []models.DocChangedFile
	for path, st := range docStatus {
		if st == nil {
			continue
		}
		if st.Staging == git.Unmodified && st.Worktree == git.Unmodified {
			continue
		}
		files = append(files, models.DocChangedFile{
			Path:   filepath.ToSlash(path),
			Status: describeStatus(*st),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	// Generate diff between the new docs branch and its base branch
	docDiff, err := s.gitService.DiffBetweenBranches(docRepo, targetBranch, docsBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation diff: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocs: completed"))

	var (
		summary      string
		conversation []models.DocConversationMessage
	)
	if llmResult != nil {
		summary = llmResult.Summary
		conversation = ConvertConversation(llmResult.Messages)
	}
	return &models.DocGenerationResult{
		Branch:       sourceBranch,
		Files:        files,
		Diff:         docDiff,
		Summary:      summary,
		Conversation: conversation,
	}, nil
}

func (s *ClientService) RequestDocChanges(projectID uint, feedback string) (*models.DocGenerationResult, error) {
	ctx := s.context
	if ctx == nil {
		return nil, fmt.Errorf("client service not initialized")
	}
	if projectID == 0 {
		return nil, fmt.Errorf("project id is required")
	}
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		return nil, fmt.Errorf("feedback is required")
	}

	sessionReq := s.OpenAIClient.DocSessionRequest()
	if sessionReq == nil {
		return nil, fmt.Errorf("no documentation generation session available")
	}
	if sessionReq.ProjectID != projectID {
		return nil, fmt.Errorf("documentation session does not match the requested project")
	}

	docRoot := strings.TrimSpace(sessionReq.DocumentationPath)
	if docRoot == "" {
		return nil, fmt.Errorf("documentation repository path is not available")
	}
	if !utils.DirectoryExists(docRoot) {
		return nil, fmt.Errorf("documentation repository path does not exist: %s", docRoot)
	}
	if !utils.HasGitRepo(docRoot) {
		return nil, fmt.Errorf("documentation repository is not a git repository: %s", docRoot)
	}

	repo, err := s.gitService.Open(docRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open documentation repository: %w", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to load documentation worktree: %w", err)
	}

	sourceBranch := strings.TrimSpace(sessionReq.SourceBranch)
	if sourceBranch == "" {
		return nil, fmt.Errorf("documentation session source branch is not available")
	}
	docsBranch := "docs-" + sourceBranch
	refName := plumbing.NewBranchReferenceName(docsBranch)
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation HEAD: %w", err)
	}
	if headRef.Name() != refName {
		if err := worktree.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
			return nil, fmt.Errorf("failed to checkout documentation branch '%s': %w", docsBranch, err)
		}
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("RequestDocChanges: starting refinement"))

	streamCtx := s.OpenAIClient.StartStream(ctx)
	defer s.OpenAIClient.StopStream()

	llmResult, err := s.OpenAIClient.ApplyDocFeedback(streamCtx, feedback)
	if err != nil {
		return nil, err
	}

	docStatus, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation repo status: %w", err)
	}

	var files []models.DocChangedFile
	for path, st := range docStatus {
		if st == nil {
			continue
		}
		if st.Staging == git.Unmodified && st.Worktree == git.Unmodified {
			continue
		}
		files = append(files, models.DocChangedFile{
			Path:   filepath.ToSlash(path),
			Status: describeStatus(*st),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	docDiff, err := runGitDiff(docRoot)
	if err != nil {
		return nil, err
	}

	var (
		summary      string
		conversation []models.DocConversationMessage
	)
	if llmResult != nil {
		summary = llmResult.Summary
		conversation = ConvertConversation(llmResult.Messages)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("RequestDocChanges: completed"))

	return &models.DocGenerationResult{
		Branch:       sourceBranch,
		Files:        files,
		Diff:         docDiff,
		Summary:      summary,
		Conversation: conversation,
	}, nil
}

func (s *ClientService) CommitDocs(projectID uint, branch string, files []string) error {
	ctx := s.context
	if ctx == nil {
		return fmt.Errorf("client service not initialized")
	}
	if projectID == 0 {
		return fmt.Errorf("project id is required")
	}
	branch = "docs-" + strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("branch is required")
	}
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

	docRoot, err := filepath.Abs(docRepoPath)
	if err != nil {
		return err
	}

	repo, err := s.gitService.Open(docRoot)
	if err != nil {
		return fmt.Errorf("failed to open documentation repository: %w", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to load documentation worktree: %w", err)
	}
	refName := plumbing.NewBranchReferenceName(branch)

	// Validate that the branch exists without checking out
	if _, err := repo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("documentation branch '%s' does not exist", branch)
		}
		return fmt.Errorf("failed to resolve documentation branch '%s': %w", branch, err)
	}

	docStatus, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to read documentation repo status: %w", err)
	}
	var normalized []string
	for _, file := range files {
		trimmed := strings.TrimSpace(file)
		if trimmed == "" {
			continue
		}
		fsPath := filepath.FromSlash(trimmed)
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
		len(normalized), branch,
	)))

	if err := s.gitService.StageFiles(repo, normalized); err != nil {
		return fmt.Errorf("failed to stage documentation changes: %w", err)
	}

	message := fmt.Sprintf("Add documentation for %s", branch)
	if _, err := s.gitService.Commit(repo, message); err != nil {
		return fmt.Errorf("failed to commit documentation changes: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"CommitDocs: committed documentation updates to '%s'",
		branch,
	)))

	return nil
}

func (s *ClientService) StopStream() {
	if s == nil || s.OpenAIClient == nil {
		return
	}
	wasRunning := s.OpenAIClient.IsRunning()
	s.OpenAIClient.StopStream()
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

func ensureBaseBranch(repo *git.Repository, targetBranch string) (plumbing.Hash, error) {
	head, err := repo.Head()
	if err != nil {
		return plumbing.Hash{}, fmt.Errorf("failed to read documentation HEAD: %w", err)
	}
	baseHash := head.Hash()
	if targetBranch == "" {
		return baseHash, nil
	}

	// Resolve target branch reference without checking out
	refName := plumbing.NewBranchReferenceName(targetBranch)
	ref, err := repo.Reference(refName, true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return baseHash, nil // Target branch doesn't exist, use current HEAD
		}
		return plumbing.Hash{}, fmt.Errorf("failed to resolve documentation branch '%s': %w", targetBranch, err)
	}
	return ref.Hash(), nil
}

// generateUniqueID creates a unique identifier for temporary directories
func generateUniqueID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// createTempDocRepo creates a temporary clone of the documentation repository
// checked out to the specified branch. Returns the temp path and cleanup function.
func createTempDocRepo(ctx context.Context, sourceRepoPath, branch string, baseHash plumbing.Hash) (tempPath string, cleanup func(), err error) {
	// Create unique temporary directory
	tempID := generateUniqueID()
	tempPath = filepath.Join(os.TempDir(), fmt.Sprintf("narrabyte-docs-%s", tempID))

	// Cleanup function that always removes temp directory
	cleanup = func() {
		if err := os.RemoveAll(tempPath); err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Failed to cleanup temp directory %s: %v", tempPath, err)))
		}
	}

	// Clone source repository to temporary location
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Creating temporary docs workspace at %s", tempPath)))

	tempRepo, err := git.PlainClone(tempPath, false, &git.CloneOptions{
		URL:      sourceRepoPath,
		Progress: nil, // Suppress clone progress output
	})
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to clone repository to temp location: %w", err)
	}

	// Get worktree for checkout operations
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to get temp repository worktree: %w", err)
	}

	// Create and checkout the documentation branch at the specified commit
	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, baseHash)
	if err := tempRepo.Storer.SetReference(ref); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to create branch '%s' in temp repo: %w", branch, err)
	}

	if err := tempWT.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to checkout branch '%s' in temp repo: %w", branch, err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Temporary docs workspace ready: branch '%s' at %s", branch, tempPath)))
	return tempPath, cleanup, nil
}

// propagateDocChanges commits all changes in the temp repository and updates
// the branch reference in the main repository to point to the new commit
func propagateDocChanges(ctx context.Context, tempRepoPath string, mainRepo *git.Repository, branch string) error {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Propagating documentation changes back to main repository"))

	// Open temporary repository
	tempRepo, err := git.PlainOpen(tempRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open temp repository: %w", err)
	}

	// Get temp repository worktree
	tempWT, err := tempRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get temp repository worktree: %w", err)
	}

	// Check if there are any changes to commit
	status, err := tempWT.Status()
	if err != nil {
		return fmt.Errorf("failed to get temp repository status: %w", err)
	}

	if status.IsClean() {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("No documentation changes to propagate"))
		return nil
	}

	// Add all changes
	if _, err := tempWT.Add("."); err != nil {
		return fmt.Errorf("failed to add changes in temp repository: %w", err)
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
}

func ConvertConversation(messages []*schema.Message) []models.DocConversationMessage {
	if len(messages) == 0 {
		return nil
	}
	result := make([]models.DocConversationMessage, 0, len(messages))
	for idx, msg := range messages {
		if msg == nil {
			continue
		}
		if msg.Role == schema.Tool {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		role := string(msg.Role)
		if idx == 0 && msg.Role == schema.User {
			role = "context"
		}
		result = append(result, models.DocConversationMessage{
			Role:    role,
			Content: content,
		})
	}
	return result
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
