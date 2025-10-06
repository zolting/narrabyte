package services

import (
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestDocumentationBranchName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "docs"},
		{"trimmed", " feature/docs ", "docs/feature/docs"},
		{"spaces", "update docs", "docs/update-docs"},
		{"path", "feature/foo", "docs/feature/foo"},
	}

	for _, tc := range cases {
		if got := documentationBranchName(tc.input); got != tc.expected {
			t.Fatalf("%s: expected %s, got %s", tc.name, tc.expected, got)
		}
	}
}

func TestHasDocsChanges(t *testing.T) {
	status := git.Status{
		"docs/index.md": {
			Staging:  git.Modified,
			Worktree: git.Modified,
		},
		"README.md": {
			Staging:  git.Unmodified,
			Worktree: git.Unmodified,
		},
	}

	if !hasDocsChanges(status, "docs") {
		t.Fatal("expected docs changes to be detected")
	}
	if hasDocsChanges(status, "guides") {
		t.Fatal("did not expect detection for unrelated directory")
	}
}

func TestCollectDocChangedFilesFiltersOutsideDocs(t *testing.T) {
	status := git.Status{
		"docs/index.md": {
			Staging:  git.Modified,
			Worktree: git.Modified,
		},
		"docs/api/intro.mdx": {
			Staging:  git.Added,
			Worktree: git.Added,
		},
		"src/main.go": {
			Staging:  git.Modified,
			Worktree: git.Modified,
		},
	}

	files := collectDocChangedFiles(status, "docs")
	if len(files) != 2 {
		t.Fatalf("expected 2 doc files, got %d", len(files))
	}
	for _, file := range files {
		if file.Path != "docs/index.md" && file.Path != "docs/api/intro.mdx" {
			t.Fatalf("unexpected path included: %s", file.Path)
		}
	}
}
