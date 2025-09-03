package main

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitService struct{}

//Logic of this whole service will need to be verified. Unsure of how to use a repo once it's been open
//Right now, git.PlainOpen(path) is being called everytime. Would be better to open it once and then use
//the right instance, but unsure of how to proceed
//Using go-git, which implements most of the git commands.

// Open an existing repo
func (g *GitService) Open(path string) (*git.Repository, error) {

	repo, err := git.PlainOpen(path)

	if err != nil {
		return nil, err
	}

	return repo, nil
}

// Push local commits to remote
func (g *GitService) Push(repoPath string) error {
	repo, err := g.Open(repoPath)
	if err != nil {
		return err
	}
	return repo.Push(&git.PushOptions{RemoteName: "origin"})
}

// Checkout
func (g *GitService) Checkout(repoPath, branch string) error {
	repo, err := g.Open(repoPath)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	return w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
}

// Fonction exemple pour go-git. Ouvre le repo et retourne le hash du current head
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
