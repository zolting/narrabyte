package Intergration_tests

import (
	"narrabyte/internal/services"
	"os"
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

// tests the difference between two commits
func TestDiffBetweenCommits(t *testing.T) {
	// Create a temp dir for the repo
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	// Init repo
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Create first file and commit
	file1 := dir + "/test.txt"
	err = os.WriteFile(file1, []byte("hello world\n"), 0644)
	assert.NoError(t, err)

	w, err := repo.Worktree()
	assert.NoError(t, err)

	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	commit1, err := w.Commit("first commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	// Modify file and commit again
	err = os.WriteFile(file1, []byte("hello world!\nnew line\n"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	commit2, err := w.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	diff, err := gs.DiffBetweenCommits(repo, commit1.String(), commit2.String())
	assert.NoError(t, err)
	t.Log(diff)
	assert.Contains(t, diff, "+new line")
	assert.Contains(t, diff, "-hello world")
	assert.Contains(t, diff, "+hello world!")
}

// tests the difference between head commits of two branches
func TestDiffBetweenBranches(t *testing.T) {
	// Create a temp dir for the repo
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	// Init repo
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Create initial file and commit on main
	file1 := dir + "/shared.txt"
	err = os.WriteFile(file1, []byte("shared content\n"), 0644)
	assert.NoError(t, err)

	w, err := repo.Worktree()
	assert.NoError(t, err)

	_, err = w.Add("shared.txt")
	assert.NoError(t, err)

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	// Create and checkout new branch
	branchName := "feature-branch"
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: true,
	})
	assert.NoError(t, err)

	// Modify file on feature branch
	err = os.WriteFile(file1, []byte("shared content\nfeature change\n"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("shared.txt")
	assert.NoError(t, err)

	_, err = w.Commit("feature commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	// Get head commit of feature branch
	featureHead, err := repo.Head()
	assert.NoError(t, err)
	featureCommitHash := featureHead.Hash().String()

	// Checkout back to main
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("master"),
	})
	assert.NoError(t, err)

	// Modify file on main branch
	err = os.WriteFile(file1, []byte("shared content\nmain change\n"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("shared.txt")
	assert.NoError(t, err)

	_, err = w.Commit("main commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	// Get head commit of main branch
	mainHead, err := repo.Head()
	assert.NoError(t, err)
	mainCommitHash := mainHead.Hash().String()

	// Diff between the two branch heads
	diff, err := gs.DiffBetweenCommits(repo, mainCommitHash, featureCommitHash)
	assert.NoError(t, err)
	t.Log(diff)

	// Assert the differences
	assert.Contains(t, diff, "+feature change")
	assert.Contains(t, diff, "-main change")
}

// Test that Init properly creates a git repository
func TestInitRepo(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	repo, err := gs.Init(dir)
	assert.NoError(t, err)
	assert.NotNil(t, repo)

	// Verify .git directory was created
	gitDir := dir + "/.git"
	_, err = os.Stat(gitDir)
	assert.NoError(t, err)

	// Verify HEAD file exists
	headFile := gitDir + "/HEAD"
	_, err = os.Stat(headFile)
	assert.NoError(t, err)
}

// Test the Checkout method
func TestCheckoutBranch(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Create initial commit
	w, err := repo.Worktree()
	assert.NoError(t, err)

	file1 := dir + "/test.txt"
	err = os.WriteFile(file1, []byte("content"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	// Create and checkout new branch
	branchName := "test-branch"

	// Create the branch first
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: true,
	})
	assert.NoError(t, err)

	// Then use our service method to checkout
	err = gs.Checkout(repo, branchName)
	assert.NoError(t, err)

	// Verify we're on the new branch
	head, err := repo.Head()
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/"+branchName, head.Name().String())
}

// Test LatestCommit method
func TestLatestCommit(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Create a commit
	w, err := repo.Worktree()
	assert.NoError(t, err)

	file1 := dir + "/test.txt"
	err = os.WriteFile(file1, []byte("content"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	commit, err := w.Commit("test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	// Test LatestCommit with the repository path
	hash, err := gs.LatestCommit(dir)
	assert.NoError(t, err)
	assert.Equal(t, commit.String(), hash)
}

// Test ValidateRepository method
func TestValidateRepository(t *testing.T) {
	// Test with empty path
	gs := &services.GitService{}
	err := gs.ValidateRepository("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Test with non-existent directory
	err = gs.ValidateRepository("/non/existent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid git repository")

	// Test with valid repository (with initial commit)
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	repo, err := gs.Init(dir)
	assert.NoError(t, err)
	assert.NotNil(t, repo)

	// Create initial commit so repository has a HEAD
	w, err := repo.Worktree()
	assert.NoError(t, err)

	file1 := dir + "/test.txt"
	err = os.WriteFile(file1, []byte("initial content"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	assert.NoError(t, err)

	err = gs.ValidateRepository(dir)
	assert.NoError(t, err)
}
