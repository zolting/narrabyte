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
		return nil, fmt.Errorf("documentation repository has uncommitted changes; please commit or stash them before generating docs")
	}

	baseHash, err := ensureBaseBranch(docRepo, docWorktree, targetBranch)
	if err != nil {
		return nil, err
	}
	if err := prepareDocumentationBranch(docRepo, docWorktree, sourceBranch, baseHash); err != nil {
		return nil, err
	}
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"GenerateDocs: documentation branch '%s' ready",
		sourceBranch,
	)))

	streamCtx := s.OpenAIClient.StartStream(ctx)
	defer s.OpenAIClient.StopStream()

	llmResult, err := s.OpenAIClient.GenerateDocs(streamCtx, &client.DocGenerationRequest{
		ProjectName:       project.ProjectName,
		CodebasePath:      codeRoot,
		DocumentationPath: docRoot,
		SourceBranch:      sourceBranch,
		TargetBranch:      targetBranch,
		SourceCommit:      sourceHash.String(),
		Diff:              diffText,
		ChangedFiles:      changedFiles,
	})
	if err != nil {
		return nil, err
	}

	docStatus, err := docWorktree.Status()
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

	docDiff, err := runGitDiff(docRoot)
	if err != nil {
		return nil, err
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocs: completed"))

	summary := ""
	if llmResult != nil {
		summary = llmResult.Summary
	}
	return &models.DocGenerationResult{
		Branch:  sourceBranch,
		Files:   files,
		Diff:    docDiff,
		Summary: summary,
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
	branch = strings.TrimSpace(branch)
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
	headRef, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to read documentation HEAD: %w", err)
	}
	currentRefName := headRef.Name()
	if currentRefName != refName {
		if err := worktree.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
			if errors.Is(err, plumbing.ErrReferenceNotFound) || errors.Is(err, git.ErrBranchNotFound) {
				return fmt.Errorf("documentation branch '%s' does not exist", branch)
			}
			return fmt.Errorf("failed to checkout documentation branch '%s': %w", branch, err)
		}
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

	args := append([]string{"add", "--"}, normalized...)
	if err := runGitCommand(docRoot, args...); err != nil {
		return fmt.Errorf("failed to stage documentation changes: %w", err)
	}

	message := fmt.Sprintf("Add documentation for %s", branch)
	if err := runGitCommand(docRoot, "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit documentation changes: %w", err)
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
		"CommitDocs: committed documentation updates to '%s'",
		branch,
	)))

	return nil
}

func (s *ClientService) StopStream() {
	s.OpenAIClient.StopStream()
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

func ensureBaseBranch(repo *git.Repository, wt *git.Worktree, targetBranch string) (plumbing.Hash, error) {
	head, err := repo.Head()
	if err != nil {
		return plumbing.Hash{}, fmt.Errorf("failed to read documentation HEAD: %w", err)
	}
	baseHash := head.Hash()
	if targetBranch == "" {
		return baseHash, nil
	}
	refName := plumbing.NewBranchReferenceName(targetBranch)
	if err := wt.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) || errors.Is(err, git.ErrBranchNotFound) {
			return baseHash, nil
		}
		return plumbing.Hash{}, fmt.Errorf("failed to checkout documentation branch '%s': %w", targetBranch, err)
	}
	head, err = repo.Head()
	if err != nil {
		return plumbing.Hash{}, fmt.Errorf("failed to read documentation HEAD: %w", err)
	}
	return head.Hash(), nil
}

func prepareDocumentationBranch(repo *git.Repository, wt *git.Worktree, branch string, baseHash plumbing.Hash) error {
	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, baseHash)
	if err := repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to update documentation branch '%s': %w", branch, err)
	}
	if err := wt.Checkout(&git.CheckoutOptions{Branch: refName, Force: true}); err != nil {
		return fmt.Errorf("failed to checkout documentation branch '%s': %w", branch, err)
	}
	return nil
}

func runGitDiff(repoPath string) (string, error) {
	var diffOutput strings.Builder

	// Get diff for tracked files (staged and unstaged changes)
	cmd := exec.Command("git", "diff", "--no-color", "HEAD")
	cmd.Dir = repoPath
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("git diff error: %s", errMsg)
		}
		return "", fmt.Errorf("git diff error: %w", err)
	}
	diffOutput.WriteString(stdout.String())

	// Get diff for untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = repoPath
	stdout.Reset()
	stderr.Reset()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("git ls-files error: %s", errMsg)
		}
		return "", fmt.Errorf("git ls-files error: %w", err)
	}

	// Create diff entries for untracked files
	untrackedFiles := strings.Fields(stdout.String())
	for _, file := range untrackedFiles {
		if file == "" {
			continue
		}
		diffOutput.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))
		diffOutput.WriteString("new file mode 100644\n")
		diffOutput.WriteString("index 0000000..0000000\n")
		diffOutput.WriteString("--- /dev/null\n")
		diffOutput.WriteString(fmt.Sprintf("+++ b/%s\n", file))

		// Read file content to include in diff
		filePath := filepath.Join(repoPath, file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			// If we can't read the file, just mark it as added without content
			diffOutput.WriteString("@@ -0,0 +1,1 @@\n")
			diffOutput.WriteString("+[Binary file or unreadable content]\n")
		} else {
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1] // Remove empty last line
			}
			diffOutput.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
			for _, line := range lines {
				diffOutput.WriteString(fmt.Sprintf("+%s\n", line))
			}
		}
	}

	return diffOutput.String(), nil
}

func runGitCommand(repoPath string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		if errMsg != "" {
			return errors.New(errMsg)
		}
		return err
	}
	return nil
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
