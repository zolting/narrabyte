package integration_tests

import (
	"narrabyte/internal/services"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
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

	// Diff between the two branch heads using branch names
	diff, err := gs.DiffBetweenBranches(repo, "master", branchName)
	assert.NoError(t, err)
	assert.Contains(t, diff, "+feature change")
	assert.Contains(t, diff, "-main change")
}

func TestDiffBetweenBranches_ReturnsErrorForMissingBranch(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := services.NewGitService()
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Seed the repository with an initial commit so the base branch exists.
	w, err := repo.Worktree()
	assert.NoError(t, err)

	file1 := filepath.Join(dir, "seed.txt")
	assert.NoError(t, os.WriteFile(file1, []byte("seed"), 0644))
	_, err = w.Add("seed.txt")
	assert.NoError(t, err)
	_, err = w.Commit("seed", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	})
	assert.NoError(t, err)

	_, err = gs.DiffBetweenBranches(repo, "master", "missing-branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch 'missing-branch' not found")
}

func TestStageFilesAndCommit_UsesDefaultSignature(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	restoreAuthor := unsetEnvForTest("GIT_AUTHOR_NAME")
	defer restoreAuthor()
	restoreEmail := unsetEnvForTest("GIT_AUTHOR_EMAIL")
	defer restoreEmail()
	restoreCommitter := unsetEnvForTest("GIT_COMMITTER_NAME")
	defer restoreCommitter()
	restoreCommitterEmail := unsetEnvForTest("GIT_COMMITTER_EMAIL")
	defer restoreCommitterEmail()

	gs := services.NewGitService()
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Create the file to stage.
	filePath := filepath.Join(dir, "docs", "guide.md")
	assert.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
	assert.NoError(t, os.WriteFile(filePath, []byte("hello docs\n"), 0644))

	winStylePath := strings.ReplaceAll(filepath.Join("docs", "guide.md"), string(filepath.Separator), "\\")
	err = gs.StageFiles(repo, []string{"   ", winStylePath})
	assert.NoError(t, err)

	commitHash, err := gs.Commit(repo, "Add documentation guide")
	assert.NoError(t, err)

	commit, err := repo.CommitObject(commitHash)
	assert.NoError(t, err)
	assert.Equal(t, "Add documentation guide", commit.Message)
	assert.Equal(t, "Narrabyte Documentation Generator", commit.Author.Name)
	assert.Equal(t, "docs@narrabyte.ai", commit.Author.Email)

	tree, err := commit.Tree()
	assert.NoError(t, err)
	entry, err := tree.FindEntry("docs/guide.md")
	assert.NoError(t, err)
	assert.Equal(t, "guide.md", entry.Name)
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

func TestListBranches_ReturnsSortedShortNames(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := services.NewGitService()
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Initial commit to materialize the default branch `master`
	w, err := repo.Worktree()
	assert.NoError(t, err)

	f := dir + "/a.txt"
	assert.NoError(t, os.WriteFile(f, []byte("a"), 0644))
	_, err = w.Add("a.txt")
	assert.NoError(t, err)
	_, err = w.Commit("init", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	})
	assert.NoError(t, err)

	// Create branches in unsorted order
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("zzz"),
		Create: true,
	})
	assert.NoError(t, err)
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("aaa"),
		Create: true,
	})
	assert.NoError(t, err)

	branches, err := gs.ListBranches(repo)
	assert.NoError(t, err)

	// Compare only names, preserving the expected sorted order
	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	expected := []string{"aaa", "master", "zzz"}
	assert.Equal(t, expected, names)
}

func TestListBranchesByPath_ReturnsSortedShortNames(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := services.NewGitService()
	repo, err := gs.Init(dir)
	assert.NoError(t, err)

	// Initial commit to materialize the default branch `master`
	w, err := repo.Worktree()
	assert.NoError(t, err)

	f := dir + "/b.txt"
	assert.NoError(t, os.WriteFile(f, []byte("b"), 0644))
	_, err = w.Add("b.txt")
	assert.NoError(t, err)
	_, err = w.Commit("init", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	})
	assert.NoError(t, err)

	// Create another branch
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature/x"),
		Create: true,
	})
	assert.NoError(t, err)

	branches, err := gs.ListBranchesByPath(dir)
	assert.NoError(t, err)

	// Compare only names, preserving the expected sorted order
	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	expected := []string{"feature/x", "master"}
	assert.Equal(t, expected, names)
}

func unsetEnvForTest(key string) func() {
	value, ok := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	return func() {
		if !ok {
			_ = os.Unsetenv(key)
			return
		}
		_ = os.Setenv(key, value)
	}
}

func TestListBranchesByPath_EmptyPath_ReturnsError(t *testing.T) {
	gs := services.NewGitService()
	branches, err := gs.ListBranchesByPath("")
	assert.Error(t, err)
	assert.Nil(t, branches)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestListBranchesByPath_InvalidPath_ReturnsError(t *testing.T) {
	gs := services.NewGitService()
	branches, err := gs.ListBranchesByPath("/non/existent/path")
	assert.Error(t, err)
	assert.Nil(t, branches)
	assert.Contains(t, err.Error(), "failed to open repository")
}

func TestGitService_StageAllInitialCommit(t *testing.T) {
	gs := services.NewGitService()
	repoDir := t.TempDir()

	// Create some files
	assert.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Docs"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(repoDir, "index.md"), []byte("Hello"), 0o644))

	repo, err := gs.Init(repoDir)
	assert.NoError(t, err)
	assert.NotNil(t, repo)

	// Stage all files
	// Use the worktree directly since integration tests exercise lower-level operations
	wt, err := repo.Worktree()
	assert.NoError(t, err)
	_, err = wt.Add(".")
	assert.NoError(t, err)

	// Commit
	hash, err := gs.Commit(repo, "Initial commit")
	assert.NoError(t, err)
	assert.NotEqual(t, hash.String(), "")

	// Detect current branch (go-git default initial branch name is 'master')
	head, err := repo.Head()
	assert.NoError(t, err)
	assert.Equal(t, plumbing.NewBranchReferenceName("master"), head.Name())
}

func TestDiffBetweenCommits_ExcludesManyLockFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	repo, err := gs.Init(dir)
	assert.NoError(t, err)
	wt, err := repo.Worktree()
	assert.NoError(t, err)

	lockFiles := []string{
		"package-lock.json",
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
	}

	// Initial commit: create all lock files and a meaningful file
	for _, f := range lockFiles {
		p := filepath.Join(dir, f)
		assert.NoError(t, os.WriteFile(p, []byte("initial\n"), 0o644))
		_, err = wt.Add(f)
		assert.NoError(t, err)
	}
	mainPath := filepath.Join(dir, "main.go")
	assert.NoError(t, os.WriteFile(mainPath, []byte("package main\nfunc main(){}\n"), 0o644))
	_, err = wt.Add("main.go")
	assert.NoError(t, err)
	c1, err := wt.Commit("init", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com"}})
	assert.NoError(t, err)

	// Second commit: modify all lock files and the meaningful file
	for _, f := range lockFiles {
		p := filepath.Join(dir, f)
		assert.NoError(t, os.WriteFile(p, []byte("changed\n"), 0o644))
		_, err = wt.Add(f)
		assert.NoError(t, err)
	}
	assert.NoError(t, os.WriteFile(mainPath, []byte("package main\nfunc main(){ /* changed */ }\n"), 0o644))
	_, err = wt.Add("main.go")
	assert.NoError(t, err)
	c2, err := wt.Commit("update", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com"}})
	assert.NoError(t, err)

	diff, err := gs.DiffBetweenCommits(repo, c1.String(), c2.String())
	assert.NoError(t, err)
	if diff == "" {
		t.Fatalf("expected non-empty diff output")
	}

	// Assert all lock files are excluded from diff output
	for _, f := range lockFiles {
		assert.NotContains(t, diff, f, "diff should exclude %s", f)
	}
	// But main.go should be present
	assert.Contains(t, diff, "main.go")
}

func TestDiffBetweenCommits_ExcludesDistLocalesPbGoAndGenerated(t *testing.T) {
	dir, err := os.MkdirTemp("", "gitservicetest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	gs := &services.GitService{}
	repo, err := gs.Init(dir)
	assert.NoError(t, err)
	wt, err := repo.Worktree()
	assert.NoError(t, err)

	// Create initial files across various patterns
	files := []string{
		"dist/main.js",           // should be excluded by dist/**
		"dist/assets/app.css",    // should be excluded by dist/**
		"locales/en/common.json", // should be excluded by locales/**/*.json
		"api/service.pb.go",      // should be excluded by *.pb.go
		"src/generated/foo.go",   // should be excluded by **/generated/**
		"src/app/main.go",        // should NOT be excluded
		"docs/_build/index.html", // should be excluded by docs/_build/**
		"public/app.min.js",      // should be excluded by *.min.js
	}

	for _, f := range files {
		p := filepath.Join(dir, f)
		assert.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		assert.NoError(t, os.WriteFile(p, []byte("v1\n"), 0o644))
		_, err = wt.Add(f)
		assert.NoError(t, err)
	}
	c1, err := wt.Commit("init", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com"}})
	assert.NoError(t, err)

	// Modify all files
	for _, f := range files {
		p := filepath.Join(dir, f)
		assert.NoError(t, os.WriteFile(p, []byte("v2\n"), 0o644))
		_, err = wt.Add(f)
		assert.NoError(t, err)
	}
	c2, err := wt.Commit("update", &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com"}})
	assert.NoError(t, err)

	diff, err := gs.DiffBetweenCommits(repo, c1.String(), c2.String())
	assert.NoError(t, err)
	if diff == "" {
		t.Fatalf("expected non-empty diff output")
	}

	// Excluded files should not be present in diff output
	excluded := []string{
		"dist/main.js",
		"dist/assets/app.css",
		"locales/en/common.json",
		"api/service.pb.go",
		"src/generated/foo.go",
		"docs/_build/index.html",
		"public/app.min.js",
	}
	for _, name := range excluded {
		assert.NotContains(t, diff, name, "diff should exclude %s", name)
	}

	// Included file should be present
	assert.Contains(t, diff, "src/app/main.go")
}
