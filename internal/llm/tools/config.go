package tools

import (
    "errors"
    "path/filepath"
    "strings"
)

// listDirBaseRoot holds an optional base directory for the list directory tools.
// If unset, we fall back to environment or project root discovery.
var listDirBaseRoot string

// SetListDirectoryBaseRoot sets the base directory that ListDirectory tools
// will treat as the root for resolving paths.
// Example: if set to "/repo", an input of "frontend" resolves to "/repo/frontend".
func SetListDirectoryBaseRoot(root string) {
    // Normalize and store an absolute, cleaned base root
    if strings.TrimSpace(root) == "" {
        listDirBaseRoot = ""
        return
    }
    if abs, err := filepath.Abs(root); err == nil {
        listDirBaseRoot = abs
        return
    }
    // Fallback to raw value if Abs fails (should be rare)
    listDirBaseRoot = root
}

// getListDirectoryBaseRoot returns the configured base directory for list tools.
// Resolution: value set via SetListDirectoryBaseRoot.
func getListDirectoryBaseRoot() (string, error) {
    if listDirBaseRoot != "" {
        return listDirBaseRoot, nil
    }

    return "", errors.New("list directory base root not set")
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
	// Join and clean the target
    candidate := filepath.Join(absBase, p)
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	// Ensure absCandidate is within absBase
	relToBase, err := filepath.Rel(absBase, absCandidate)
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
