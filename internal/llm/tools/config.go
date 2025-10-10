package tools

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type baseContext struct {
	root     string
	snapshot *GitSnapshot
}

var currentBaseContext baseContext
var scopedIgnorePatterns []string

// SetListDirectoryBaseRoot sets the base directory that ListDirectory tools
// will treat as the root for resolving paths.
// Example: if set to "/repo", an input of "frontend" resolves to "/repo/frontend".
func SetListDirectoryBaseRoot(root string) {
	// Normalize and store an absolute, cleaned base root
	if strings.TrimSpace(root) == "" {
		currentBaseContext.root = ""
		return
	}
	if abs, err := filepath.Abs(root); err == nil {
		currentBaseContext.root = abs
		return
	}
	// Fallback to raw value if Abs fails (should be rare)
	currentBaseContext.root = root
}

// getListDirectoryBaseRoot returns the configured base directory for list tools.
// Resolution: value set via SetListDirectoryBaseRoot.
func getListDirectoryBaseRoot() (string, error) {
	if currentBaseContext.root != "" {
		return currentBaseContext.root, nil
	}

	return "", errors.New("list directory base root not set")
}

func SetScopedIgnorePatterns(patterns []string) {
	if len(patterns) == 0 {
		scopedIgnorePatterns = nil
		return
	}
	scopedIgnorePatterns = append([]string{}, patterns...)
}

func GetScopedIgnorePatterns() []string {
	if len(scopedIgnorePatterns) == 0 {
		return nil
	}
	out := make([]string, len(scopedIgnorePatterns))
	copy(out, scopedIgnorePatterns)
	return out
}

func SetGitSnapshot(snapshot *GitSnapshot) {
	currentBaseContext.snapshot = snapshot
}

func CurrentGitSnapshot() *GitSnapshot {
	return currentBaseContext.snapshot
}

// safeJoinUnderBase resolves a path under base, returning an absolute path that
// is guaranteed to remain within base. If the resolution escapes base, ok=false.
func safeJoinUnderBase(base, p string) (abs string, ok bool) {
	// Clean inputs
	cleanBase := base
	if cleanBase == "" {
		cleanBase = "."
	}
	// Ensure absolute base
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return "", false
	}
	// Resolve symlinks for consistent comparison
	evalBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		// If symlink evaluation fails, fall back to absolute path
		evalBase = absBase
	}

	// Join and clean the target
	candidate := filepath.Join(evalBase, p)
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	// Resolve symlinks for the candidate path
	evalCandidate, err := filepath.EvalSymlinks(absCandidate)
	if err != nil {
		// If symlink evaluation fails (e.g., file doesn't exist yet), fall back to absolute path
		evalCandidate = absCandidate
	}

	// Ensure evalCandidate is within evalBase
	relToBase, err := filepath.Rel(evalBase, evalCandidate)
	if err != nil {
		return "", false
	}
	if relToBase == "." {
		return absCandidate, true
	}
	// If rel starts with ".." then it escapes
	if len(relToBase) >= 2 && relToBase[:2] == ".." {
		return "", false
	}
	return absCandidate, true
}

func formatSnapshotInfo(snapshot *GitSnapshot) string {
	if snapshot == nil {
		return "no-snapshot"
	}
	branch := snapshot.Branch()
	commit := snapshot.CommitHash().String()
	if len(commit) > 8 {
		commit = commit[:8]
	}
	if branch != "" {
		return fmt.Sprintf("%s@%s", branch, commit)
	}
	return commit
}
