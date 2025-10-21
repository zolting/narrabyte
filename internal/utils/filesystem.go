package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}

// HasGitRepo returns true when the provided directory either contains a .git
// folder or is nested within a Git worktree. The caller is responsible for
// ensuring the input path exists.
func HasGitRepo(path string) bool {
	_, ok := FindGitRepoRoot(path)
	return ok
}

// FindGitRepoRoot traverses upward from the given path until it finds a
// directory containing a .git folder. The second return value reports whether a
// repository root was discovered.
func FindGitRepoRoot(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	current := abs
	for {
		info, statErr := os.Stat(current)
		if statErr != nil || !info.IsDir() {
			return "", false
		}
		gitDir := filepath.Join(current, ".git")
		if gitInfo, gitErr := os.Stat(gitDir); gitErr == nil && gitInfo.IsDir() {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

// SamePath returns true when two filesystem paths resolve to the same absolute
// location.
func SamePath(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	if errA != nil {
		absA = filepath.Clean(a)
	}
	absB, errB := filepath.Abs(b)
	if errB != nil {
		absB = filepath.Clean(b)
	}
	return absA == absB
}
