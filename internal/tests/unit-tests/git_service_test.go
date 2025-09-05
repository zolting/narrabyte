package unit_tests

import (
	"narrabyte/internal/services"
	"os"
	"testing"

	git "github.com/go-git/go-git/v5"
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
