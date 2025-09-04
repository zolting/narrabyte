package main

import (
	"fmt"

	"bytes"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitService struct{}

// Open an existing repo
func (g *GitService) Open(path string) (*git.Repository, error) {

	repo, err := git.PlainOpen(path)

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

// Return latest commit hash
func (g *GitService) LatestCommit() (string, error) {
	// Ouvre notre repo
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", fmt.Errorf("failed to open repo: %w", err)
	}

	// Va chercher le head
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Va chercher le hash du head
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.String(), nil
}
