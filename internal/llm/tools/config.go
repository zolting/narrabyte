package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type baseContext struct {
	root     string
	docsRoot string
	codeRoot string
	snapshot *GitSnapshot
	ignores  []string
}

var (
	contextMu       sync.RWMutex
	defaultContext  = &baseContext{}
	sessionContexts = make(map[string]*baseContext)
)

type contextKey string

const sessionIDKey contextKey = "narrabyte/tools/session"

// ContextWithSession annotates ctx with a logical session identifier so tools can
// keep per-session state (e.g., project root, snapshot, ignore patterns) without
// interfering with parallel sessions.
func ContextWithSession(ctx context.Context, sessionID string) context.Context {
	if strings.TrimSpace(sessionID) == "" {
		return ctx
	}
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// SessionIDFromContext extracts the logical session identifier associated with ctx.
func SessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(sessionIDKey).(string); ok {
		return v
	}
	return ""
}

func normalizeRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return ""
	}
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func ensureSessionContext(sessionID string) *baseContext {
	if strings.TrimSpace(sessionID) == "" {
		return defaultContext
	}
	contextMu.Lock()
	defer contextMu.Unlock()
	if ctx, ok := sessionContexts[sessionID]; ok {
		return ctx
	}
	ctx := &baseContext{}
	sessionContexts[sessionID] = ctx
	return ctx
}

func lookupSessionContext(sessionID string) *baseContext {
	if strings.TrimSpace(sessionID) == "" {
		return defaultContext
	}
	contextMu.RLock()
	defer contextMu.RUnlock()
	return sessionContexts[sessionID]
}

// SetListDirectoryBaseRoot sets the default base directory used by list-style tools.
func SetListDirectoryBaseRoot(root string) {
	defaultContext.root = normalizeRoot(root)
}

// SetListDirectoryBaseRootForSession sets the base directory for a specific logical session.
func SetListDirectoryBaseRootForSession(sessionID, root string) {
	ctx := ensureSessionContext(sessionID)
	ctx.root = normalizeRoot(root)
}

// ListDirectoryBaseRootForSession returns the configured base directory for a session.
func ListDirectoryBaseRootForSession(sessionID string) string {
	if ctx := lookupSessionContext(sessionID); ctx != nil {
		return ctx.root
	}
	return ""
}

// SetScopedIgnorePatterns configures default ignore patterns for list-style tools.
func SetScopedIgnorePatterns(patterns []string) {
	if len(patterns) == 0 {
		defaultContext.ignores = nil
		return
	}
	defaultContext.ignores = append([]string{}, patterns...)
}

// SetScopedIgnorePatternsForSession configures ignore patterns for a specific session.
func SetScopedIgnorePatternsForSession(sessionID string, patterns []string) {
	ctx := ensureSessionContext(sessionID)
	if len(patterns) == 0 {
		ctx.ignores = nil
		return
	}
	ctx.ignores = append([]string{}, patterns...)
}

// GetScopedIgnorePatterns returns a copy of the default ignore patterns.
func GetScopedIgnorePatterns() []string {
	if len(defaultContext.ignores) == 0 {
		return nil
	}
	out := make([]string, len(defaultContext.ignores))
	copy(out, defaultContext.ignores)
	return out
}

// GetScopedIgnorePatternsForSession returns the ignore patterns configured for a session.
func GetScopedIgnorePatternsForSession(sessionID string) []string {
	if ctx := lookupSessionContext(sessionID); ctx != nil && len(ctx.ignores) > 0 {
		out := make([]string, len(ctx.ignores))
		copy(out, ctx.ignores)
		return out
	}
	return nil
}

// scopedIgnorePatterns resolves the effective ignore pattern list for ctx.
func scopedIgnorePatterns(ctx context.Context) []string {
	sessionID := SessionIDFromContext(ctx)
	if ignores := GetScopedIgnorePatternsForSession(sessionID); len(ignores) > 0 {
		return ignores
	}
	return GetScopedIgnorePatterns()
}

// SetGitSnapshot binds the default Git snapshot for read operations.
func SetGitSnapshot(snapshot *GitSnapshot) {
	defaultContext.snapshot = snapshot
}

// SetGitSnapshotForSession binds a Git snapshot for a specific session.
func SetGitSnapshotForSession(sessionID string, snapshot *GitSnapshot) {
	ctx := ensureSessionContext(sessionID)
	ctx.snapshot = snapshot
}

// GitSnapshotForSession returns the Git snapshot associated with a session.
func GitSnapshotForSession(sessionID string) *GitSnapshot {
	if ctx := lookupSessionContext(sessionID); ctx != nil {
		return ctx.snapshot
	}
	return nil
}

// currentGitSnapshot resolves the snapshot to use for ctx.
func currentGitSnapshot(ctx context.Context) *GitSnapshot {
	if snap := GitSnapshotForSession(SessionIDFromContext(ctx)); snap != nil {
		return snap
	}
	return defaultContext.snapshot
}

// ClearSession releases per-session state.
func ClearSession(sessionID string) {
	if strings.TrimSpace(sessionID) == "" {
		return
	}
	contextMu.Lock()
	delete(sessionContexts, sessionID)
	contextMu.Unlock()
}

// SetDocsRootForSession sets the documentation repository root for a specific session.
func SetDocsRootForSession(sessionID, root string) {
	ctx := ensureSessionContext(sessionID)
	ctx.docsRoot = normalizeRoot(root)
}

// SetCodeRootForSession sets the codebase repository root for a specific session.
func SetCodeRootForSession(sessionID, root string) {
	ctx := ensureSessionContext(sessionID)
	ctx.codeRoot = normalizeRoot(root)
}

// DocsRootForSession returns the documentation repository root for a session.
func DocsRootForSession(sessionID string) string {
	if ctx := lookupSessionContext(sessionID); ctx != nil {
		return ctx.docsRoot
	}
	return ""
}

// CodeRootForSession returns the codebase repository root for a session.
func CodeRootForSession(sessionID string) string {
	if ctx := lookupSessionContext(sessionID); ctx != nil {
		return ctx.codeRoot
	}
	return ""
}

// getDocsRoot resolves the documentation root for ctx.
func getDocsRoot(ctx context.Context) string {
	return DocsRootForSession(SessionIDFromContext(ctx))
}

// getCodeRoot resolves the codebase root for ctx.
func getCodeRoot(ctx context.Context) string {
	return CodeRootForSession(SessionIDFromContext(ctx))
}

// ResolveRepositoryPath resolves a relative path against a repository root.
// Returns absolute path or error if repository is invalid, path is absolute, or path escapes the repository.
func ResolveRepositoryPath(ctx context.Context, repo Repository, relPath string) (string, error) {
	rel := strings.TrimSpace(relPath)
	if rel == "" {
		rel = "."
	}

	// Reject absolute paths
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute paths are not allowed; use a relative path within the repository (got: %s)", rel)
	}

	var root string
	switch repo {
	case RepositoryDocs:
		root = getDocsRoot(ctx)
	case RepositoryCode:
		root = getCodeRoot(ctx)
	default:
		return "", fmt.Errorf("invalid repository '%s'; must be 'docs' or 'code'", repo)
	}

	if root == "" {
		return "", fmt.Errorf("repository '%s' is not configured for this session", repo)
	}

	// Safe join with escape check
	abs, ok := safeJoinUnderBase(root, rel)
	if !ok {
		return "", fmt.Errorf("path '%s' escapes the %s repository", rel, repo)
	}

	return abs, nil
}

// GetRepositoryRoot returns the root path for a repository in the given context.
func GetRepositoryRoot(ctx context.Context, repo Repository) (string, error) {
	switch repo {
	case RepositoryDocs:
		root := getDocsRoot(ctx)
		if root == "" {
			return "", fmt.Errorf("docs repository is not configured")
		}
		return root, nil
	case RepositoryCode:
		root := getCodeRoot(ctx)
		if root == "" {
			return "", fmt.Errorf("code repository is not configured")
		}
		return root, nil
	default:
		return "", fmt.Errorf("invalid repository '%s'", repo)
	}
}

// FormatDisplayPath creates a display path with repository prefix (e.g., "docs:path/to/file").
func FormatDisplayPath(repo Repository, relPath string) string {
	rel := strings.TrimSpace(relPath)
	if rel == "" || rel == "." {
		return string(repo) + ":/"
	}
	return string(repo) + ":" + filepath.ToSlash(rel)
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
