package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"narrabyte/internal/models"
	"narrabyte/internal/utils"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var excludedPatterns = []string{
	// Lock and dependency files
	"package-lock.json",
	"npm-shrinkwrap.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"bun.lockb",
	"composer.lock",
	"Gemfile.lock",
	"poetry.lock",
	"Pipfile.lock",
	"Cargo.lock",
	"go.sum",
	"gradle.lockfile",
	"packages.lock.json",
	"project.assets.json",
	"pubspec.lock",

	// Generated code
	"*.pb.go",
	"*.pb.ts",
	"*.pb.swift",
	"*.pb.cc",
	"*_grpc.*",
	"**/generated/**",
	"generated/**",
	"src/api/**",
	"**snapshots/**",
	"testdata/**/*.golden",
	"**/*.expected",
	"**/*.out",

	// Build artifacts
	"dist/**",
	"build/**",
	"public/**",
	"coverage/**",
	"sonarqube-report/**",
	".next/**",
	"docs/_build/**",
	"site/**",
	"generated-docs/**",

	// Frontend / bundles
	"*.min.js",
	"*.min.css",
	"*.bundle.js",
	"*.map",

	// Localization files
	"locales/**/*.json",
	"i18n/**/*.json",
	"translations/**/*.json",
	"config/locales/**/*.yml",

	// Migrations / seeds / fixtures
	"prisma/migrations/**/*",
	"**/migrations/000*.py",
	"db/migrate/**/*.rb",
	"fixtures/**/*.json",
	"seed/**/*.json",
	"seed/**/*.sql",
	"data/**/*.csv",
	"data/**/*.json",

	// Assets
	"*.png",
	"*.jpg",
	"*.jpeg",
	"*.gif",
	"*.webp",
	"*.svg",
	"*.woff",
	"*.woff2",
	"*.ttf",
	"*.mp3",
	"*.mp4",
	"*.webm",

	// Editor / workspace
	".vscode/**",
	".idea/**",
	"*.iml",
	"*.pbxproj",
	"*.xcworkspace/**",
	"*.xcodeproj/**",
	"workspace.json",
	"project.json",
	"nx.json",
	"lerna.json",

	// Miscellaneous build artifacts
	"tsconfig.tsbuildinfo",
	".eslintcache",
}

type GitService struct {
	context context.Context
}

func (g *GitService) Startup(ctx context.Context) {
	g.context = ctx
}

func NewGitService() *GitService {
	return &GitService{}
}

// Init initializes a new git repo at given path
func (g *GitService) Init(path string) (*git.Repository, error) {
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Open an existing repo
func (g *GitService) Open(path string) (*git.Repository, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	abs, err := filepath.Abs(clean)
	if err != nil {
		abs = filepath.Clean(clean)
	}
	// Resolve to the actual Git repository root to support being passed a nested folder (e.g., docs/)
	if root, ok := utils.FindGitRepoRoot(abs); ok {
		abs = root
	}

	repo, err := git.PlainOpen(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository at %s: %w", abs, err)
	}

	return repo, nil
}

// Clone clones a repository from a remote URL into the given local path
func (g *GitService) Clone(url, path string) (*git.Repository, error) {
	if url == "" {
		return nil, fmt.Errorf("clone url cannot be empty")
	}
	if path == "" {
		return nil, fmt.Errorf("clone path cannot be empty")
	}

	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Push local commits to remote
func (g *GitService) Push(repo *git.Repository) error {
	return repo.Push(&git.PushOptions{RemoteName: "origin"}) //Other options can be added
}

// Pull changes from remote
func (g *GitService) Pull(repo *git.Repository) error {
	w, err := repo.Worktree()

	if err != nil {
		return err
	}

	return w.Pull(&git.PullOptions{RemoteName: "origin"}) //Other options can be added
}

// Checkout
func (g *GitService) Checkout(repo *git.Repository, branch string) error {

	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	return w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
}

// DiffBetweenCommits returns the patch (diff) between two commits by their hashes.
func (g *GitService) DiffBetweenCommits(repo *git.Repository, hash1, hash2 string) (string, error) {
	commit1, err := repo.CommitObject(plumbing.NewHash(hash1))
	if err != nil {
		return "", fmt.Errorf("failed to get commit1: %w", err)
	}
	commit2, err := repo.CommitObject(plumbing.NewHash(hash2))
	if err != nil {
		return "", fmt.Errorf("failed to get commit2: %w", err)
	}

	tree1, err := commit1.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get tree1: %w", err)
	}
	tree2, err := commit2.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get tree2: %w", err)
	}

	patch, err := tree1.Patch(tree2)
	if err != nil {
		return "", fmt.Errorf("failed to get patch: %w", err)
	}

	var buf bytes.Buffer
	if err := patch.Encode(&buf); err != nil {
		return "", fmt.Errorf("failed to encode patch: %w", err)
	}

	filtered := filterUnifiedDiff(buf.String())
	return filtered, nil
}

// filterUnifiedDiff removes segments whose file path matches shouldExclude; preserves unified diff markers.
func filterUnifiedDiff(diffText string) string {
	if diffText == "" {
		return ""
	}
	lines := strings.Split(diffText, "\n")
	var out strings.Builder
	segment := make([]string, 0, 128)
	fileA := ""
	fileB := ""
	flush := func() {
		candidateA := filepath.Base(fileA)
		candidateB := filepath.Base(fileB)
		if (candidateA != "" && shouldExclude(candidateA)) || (candidateB != "" && shouldExclude(candidateB)) {
			segment = segment[:0]
			fileA, fileB = "", ""
			return
		}
		for _, l := range segment {
			out.WriteString(l)
			out.WriteByte('\n')
		}
		segment = segment[:0]
		fileA, fileB = "", ""
	}
	for _, l := range lines {
		if strings.HasPrefix(l, "diff --git ") {
			if len(segment) > 0 {
				flush()
			}
			segment = append(segment, l)
			parts := strings.Split(l, " ")
			if len(parts) >= 5 {
				if strings.HasPrefix(parts[2], "a/") {
					fileA = normalizePathSlashes(strings.TrimPrefix(parts[2], "a/"))
				}
				if strings.HasPrefix(parts[3], "b/") {
					fileB = normalizePathSlashes(strings.TrimPrefix(parts[3], "b/"))
				}
			}
			continue
		}
		if strings.HasPrefix(l, "--- a/") {
			fileA = normalizePathSlashes(strings.TrimPrefix(l, "--- a/"))
		}
		if strings.HasPrefix(l, "+++ b/") {
			fileB = normalizePathSlashes(strings.TrimPrefix(l, "+++ b/"))
		}
		segment = append(segment, l)
	}
	if len(segment) > 0 {
		flush()
	}
	return out.String()
}

// normalizePathSlashes ensures forward slashes and strips leading ./
func normalizePathSlashes(p string) string {
	clean := strings.TrimSpace(p)
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.ReplaceAll(clean, "\\", "/")
	return clean
}

func shouldExclude(path string) bool {
	base := filepath.Base(path)
	for _, excl := range excludedPatterns {
		if base == excl {
			return true
		}
	}
	return false
}

// DiffBetweenBranches returns the patch (diff) between two branches by name.
func (g *GitService) DiffBetweenBranches(repo *git.Repository, baseBranch, compareBranch string) (string, error) {
	if repo == nil {
		return "", fmt.Errorf("repo cannot be nil")
	}
	if baseBranch == "" || compareBranch == "" {
		return "", fmt.Errorf("branch names are required")
	}

	baseRef, err := repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", fmt.Errorf("branch '%s' not found", baseBranch)
		}
		return "", fmt.Errorf("failed to resolve branch '%s': %w", baseBranch, err)
	}

	compareRef, err := repo.Reference(plumbing.NewBranchReferenceName(compareBranch), true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", fmt.Errorf("branch '%s' not found", compareBranch)
		}
		return "", fmt.Errorf("failed to resolve branch '%s': %w", compareBranch, err)
	}

	return g.DiffBetweenCommits(repo, baseRef.Hash().String(), compareRef.Hash().String())
}

// LatestCommit returns the latest commit hash for the given repository path
func (g *GitService) LatestCommit(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path cannot be empty")
	}

	// Resolve to repo root and validate that the path is a git repository
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = filepath.Clean(repoPath)
	}
	if root, ok := utils.FindGitRepoRoot(abs); ok {
		abs = root
	}
	if err := g.ValidateRepository(abs); err != nil {
		return "", fmt.Errorf("invalid repository: %w", err)
	}

	repo, err := git.PlainOpen(abs)
	if err != nil {
		return "", fmt.Errorf("failed to open repository at %s: %w", abs, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return ref.Hash().String(), nil
}

// ValidateRepository checks if the given path is a valid git repository
func (g *GitService) ValidateRepository(repoPath string) error {
	if repoPath == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = filepath.Clean(repoPath)
	}
	if root, ok := utils.FindGitRepoRoot(abs); ok {
		abs = root
	}

	repo, err := git.PlainOpen(abs)
	if err != nil {
		return fmt.Errorf("not a valid git repository: %w", err)
	}

	// Try to get HEAD to ensure repository is in a valid state
	_, err = repo.Head()
	if err != nil {
		return fmt.Errorf("repository is in an invalid state: %w", err)
	}

	return nil
}

// ListBranches returns all local branches and their last commit date for an opened repository.
func (g *GitService) ListBranches(repo *git.Repository) ([]models.BranchInfo, error) {
	if repo == nil {
		return nil, fmt.Errorf("repo cannot be nil")
	}

	iter, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var branches []models.BranchInfo
	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		// Get the commit at the tip of this branch to extract the commit date
		commit, cErr := repo.CommitObject(ref.Hash())
		if cErr != nil {
			return cErr
		}
		branches = append(branches, models.BranchInfo{
			Name:           name,
			LastCommitDate: commit.Author.When,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	return branches, nil
}

// ListBranchesByPath opens the repo at repoPath and returns all local branches.
func (g *GitService) ListBranchesByPath(repoPath string) ([]models.BranchInfo, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = filepath.Clean(repoPath)
	}
	if root, ok := utils.FindGitRepoRoot(abs); ok {
		abs = root
	}
	repo, err := git.PlainOpen(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository at %s: %w", abs, err)
	}
	return g.ListBranches(repo)
}

// StageFiles adds the provided file paths to the index of the repository.
func (g *GitService) StageFiles(repo *git.Repository, paths []string) error {
	if repo == nil {
		return fmt.Errorf("repo cannot be nil")
	}
	if len(paths) == 0 {
		return nil
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	for _, path := range paths {
		clean := strings.TrimSpace(path)
		if clean == "" {
			continue
		}
		normalized := filepath.ToSlash(strings.ReplaceAll(clean, "\\", "/"))
		if _, err := wt.Add(normalized); err != nil {
			return fmt.Errorf("failed to stage '%s': %w", path, err)
		}
	}

	return nil
}

// Commit creates a commit with the provided message using the staged changes.
func (g *GitService) Commit(repo *git.Repository, message string) (plumbing.Hash, error) {
	if repo == nil {
		return plumbing.ZeroHash, fmt.Errorf("repo cannot be nil")
	}
	if strings.TrimSpace(message) == "" {
		return plumbing.ZeroHash, fmt.Errorf("commit message cannot be empty")
	}

	wt, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get worktree: %w", err)
	}

	sig := signatureFromEnv()
	hash, err := wt.Commit(message, &git.CommitOptions{
		Author:    sig,
		Committer: sig,
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to create commit: %w", err)
	}

	return hash, nil
}

func signatureFromEnv() *object.Signature {
	name := os.Getenv("GIT_AUTHOR_NAME")
	if name == "" {
		name = os.Getenv("GIT_COMMITTER_NAME")
	}
	if name == "" {
		name = "Narrabyte Documentation Generator"
	}

	email := os.Getenv("GIT_AUTHOR_EMAIL")
	if email == "" {
		email = os.Getenv("GIT_COMMITTER_EMAIL")
	}
	if email == "" {
		email = "docs@narrabyte.ai"
	}

	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

// GetCurrentBranch returns the name of the current branch for the repository at the given path.
func (g *GitService) GetCurrentBranch(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path cannot be empty")
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = filepath.Clean(repoPath)
	}
	if root, ok := utils.FindGitRepoRoot(abs); ok {
		abs = root
	}

	repo, err := git.PlainOpen(abs)
	if err != nil {
		return "", fmt.Errorf("failed to open repository at %s: %w", abs, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return ref.Name().Short(), nil
}

// HasUncommittedChanges checks if the repository has any uncommitted changes (staged or unstaged).
func (g *GitService) HasUncommittedChanges(repoPath string) (bool, error) {
	if repoPath == "" {
		return false, fmt.Errorf("repository path cannot be empty")
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = filepath.Clean(repoPath)
	}
	if root, ok := utils.FindGitRepoRoot(abs); ok {
		abs = root
	}

	repo, err := git.PlainOpen(abs)
	if err != nil {
		return false, fmt.Errorf("failed to open repository at %s: %w", abs, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	// Check if there are any changes (staged or unstaged)
	return !status.IsClean(), nil
}

// BranchExists checks if a local branch exists in the given repository.
func (g *GitService) BranchExists(repo *git.Repository, branch string) (bool, error) {
	if repo == nil {
		return false, fmt.Errorf("repo cannot be nil")
	}
	name := strings.TrimSpace(branch)
	if name == "" {
		return false, fmt.Errorf("branch name is required")
	}
	_, err := repo.Reference(plumbing.NewBranchReferenceName(name), true)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check branch '%s': %w", name, err)
}

// DeleteBranch deletes a local branch reference from the repository.
// It does not modify the working tree and will fail if the reference cannot be removed.
func (g *GitService) DeleteBranch(repo *git.Repository, branch string) error {
	if repo == nil {
		return fmt.Errorf("repo cannot be nil")
	}
	name := strings.TrimSpace(branch)
	if name == "" {
		return fmt.Errorf("branch name is required")
	}
	refName := plumbing.NewBranchReferenceName(name)
	return repo.Storer.RemoveReference(refName)
}

// DeleteBranchByPath opens the repository at the given path (resolving nested directories)
// and deletes the specified local branch reference.
func (g *GitService) DeleteBranchByPath(repoPath string, branch string) error {
	clean := strings.TrimSpace(repoPath)
	if clean == "" {
		return fmt.Errorf("repository path cannot be empty")
	}
	repo, err := g.Open(clean)
	if err != nil {
		return err
	}
	return g.DeleteBranch(repo, branch)
}
