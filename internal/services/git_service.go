package services

import (
	"context"
	"fmt"

	"bytes"
	"sort"

	"narrabyte/internal/models"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitService struct {
	context context.Context
}

func (g *GitService) Startup(ctx context.Context) {
	g.context = ctx
}

func NewGitService() *GitService {
	return &GitService{}
}

// PlainInit initializes a new git repo at given path
func (g *GitService) Init(path string) (*git.Repository, error) {
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Open an existing repo
func (g *GitService) Open(path string) (*git.Repository, error) {

	repo, err := git.PlainOpen(path)

	if err != nil {
		return nil, err
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
	err = patch.Encode(&buf)

	if err != nil {
		return "", fmt.Errorf("failed to encode patch: %w", err)
	}
	return buf.String(), nil
}

// LatestCommit returns the latest commit hash for the given repository path
func (g *GitService) LatestCommit(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path cannot be empty")
	}

	// Validate that the path is a git repository
	if err := g.ValidateRepository(repoPath); err != nil {
		return "", fmt.Errorf("invalid repository: %w", err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository at %s: %w", repoPath, err)
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

	repo, err := git.PlainOpen(repoPath)
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

	// Keep alphabetical order by branch name; frontend can sort by recency
	sort.Slice(branches, func(i, j int) bool { return branches[i].Name < branches[j].Name })
	return branches, nil
}

// ListBranchesByPath opens the repo at repoPath and returns all local branches.
func (g *GitService) ListBranchesByPath(repoPath string) ([]models.BranchInfo, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository at %s: %w", repoPath, err)
	}
	return g.ListBranches(repo)
}
