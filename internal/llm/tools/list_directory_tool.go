package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ---------- LS-style listing tool ----------

// Default ignore patterns similar to the TS tool.
var DefaultIgnorePatterns = []string{
	"node_modules/",
	"__pycache__/",
	".git/",
	"dist/",
	"build/",
	"target/",
	"vendor/",
	"bin/",
	"obj/",
	".idea/",
	".vscode/",
	".zig-cache/",
	"zig-out",
	".coverage",
	"coverage/",
	"vendor/",
	"tmp/",
	"temp/",
	".cache/",
	"cache/",
	"logs/",
	".venv/",
	"venv/",
	"env/",
}

const listLimit = 100

type ListLSInput struct {
	// Absolute or relative to project root; relative is resolved under the configured base root.
	Path   string   `json:"path,omitempty" jsonschema:"description=The absolute path to the directory to list (or relative to project root)"`
	Ignore []string `json:"ignore,omitempty" jsonschema:"description=List of glob-like patterns to ignore"`
}

// ListDirectory produces a simple textual tree listing similar to the TS tool.
func ListDirectory(_ context.Context, in *ListLSInput) (string, error) {
	base := getListDirectoryBaseRoot()
	req := "."
	if in != nil && strings.TrimSpace(in.Path) != "" {
		req = strings.TrimSpace(in.Path)
	}

	// Resolve search path under base (absolute allowed if it resides under base)
	var searchPath string
	if filepath.IsAbs(req) {
		absBase, err := filepath.Abs(base)
		if err != nil {
			return "", err
		}
		absReq, err := filepath.Abs(req)
		if err != nil {
			return "", err
		}
		relToBase, err := filepath.Rel(absBase, absReq)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(relToBase, "..") {
			return "", fmt.Errorf("path escapes the configured base root")
		}
		searchPath = absReq
	} else {
		abs, ok := safeJoinUnderBase(base, req)
		if !ok {
			return "", fmt.Errorf("path escapes the configured base root")
		}
		searchPath = abs
	}

	// Ensure directory exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", searchPath)
	}

	// Compose ignore patterns
	patterns := append([]string{}, DefaultIgnorePatterns...)
	if in != nil && len(in.Ignore) > 0 {
		patterns = append(patterns, in.Ignore...)
	}

	// Collect files up to limit, honoring ignore patterns
	var files []string // slash-separated paths relative to searchPath
	err = filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// skip unreadable entries
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if p == searchPath {
			return nil
		}
		rel, _ := filepath.Rel(searchPath, p)
		rel = filepath.ToSlash(rel)

		// If directory and ignored, skip subtree
		if d.IsDir() {
			if matchIgnoredDir(rel, patterns) {
				return fs.SkipDir
			}
			return nil
		}
		// If file is ignored, skip
		if matchIgnoredFile(rel, patterns) {
			return nil
		}

		files = append(files, rel)
		if len(files) >= listLimit {
			// Stop traversal once we've reached the limit
			return errors.New("__LIST_LIMIT_REACHED__")
		}
		return nil
	})
	if err != nil {
		if err.Error() != "__LIST_LIMIT_REACHED__" {
			return "", err
		}
	}

	// Build directory structure
	dirs := map[string]struct{}{}
	filesByDir := map[string][]string{}

	for _, f := range files {
		dir := path.Dir(f)
		if dir == "." {
			// no additional parents
		}
		parts := []string{}
		if dir != "." {
			parts = strings.Split(dir, "/")
		}
		// add all parent directories, including "."
		dirs["."] = struct{}{}
		for i := 0; i <= len(parts); i++ {
			if i == 0 {
				continue // already added "."
			}
			dp := strings.Join(parts[:i], "/")
			if dp != "" {
				dirs[dp] = struct{}{}
			}
		}

		base := path.Base(f)
		if _, ok := filesByDir[dir]; !ok {
			filesByDir[dir] = []string{}
		}
		filesByDir[dir] = append(filesByDir[dir], base)
	}

	// Renderer
	// Collect all dirs into slice for iteration
	allDirs := make([]string, 0, len(dirs))
	for d := range dirs {
		allDirs = append(allDirs, d)
	}

	// Helper to compute children of a directory
	childrenOf := func(parent string) []string {
		var children []string
		for _, d := range allDirs {
			if d == parent {
				continue
			}
			if path.Dir(d) == parent {
				children = append(children, d)
			}
		}
		sort.Strings(children)
		return children
	}

	var renderDir func(dirPath string, depth int) string
	renderDir = func(dirPath string, depth int) string {
		indent := strings.Repeat("  ", depth)
		var out strings.Builder
		if depth > 0 {
			out.WriteString(indent)
			out.WriteString(path.Base(dirPath))
			out.WriteString("/\n")
		}

		childIndent := strings.Repeat("  ", depth+1)
		for _, child := range childrenOf(dirPath) {
			out.WriteString(renderDir(child, depth+1))
		}
		// files for this dir
		fs := filesByDir[dirPath]
		sort.Strings(fs)
		for _, f := range fs {
			out.WriteString(childIndent)
			out.WriteString(f)
			out.WriteByte('\n')
		}
		return out.String()
	}

	// Top header and tree
	absHeader := filepath.Clean(searchPath) + string(os.PathSeparator)
	var b strings.Builder
	b.WriteString(absHeader)
	b.WriteByte('\n')
	b.WriteString(renderDir(".", 0))
	println("ListDirectory result: ", b.String())
	return b.String(), nil
}

// matchIgnoredDir returns true if the directory (relative, slash-separated) should be ignored.
func matchIgnoredDir(relDir string, patterns []string) bool {
	segs := strings.Split(relDir, "/")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Treat entries ending with '/' (or '/**') as dir names to skip anywhere in the path
		dirPat := strings.TrimSuffix(p, "/**")
		dirPat = strings.TrimSuffix(dirPat, "/")
		if dirPat != p { // indicates it was a dir-style pattern
			for _, s := range segs {
				if s == dirPat {
					return true
				}
			}
			continue
		}
		// Also handle bare names like ".coverage" as a directory match
		if !strings.ContainsAny(p, "*?[") {
			for _, s := range segs {
				if s == p {
					return true
				}
			}
		}
	}
	return false
}

// matchIgnoredFile returns true if the file path (relative, slash-separated) matches a simple ignore.
func matchIgnoredFile(relFile string, patterns []string) bool {
	// For simplicity, reuse directory logic for parent components and add basename checks
	dir := path.Dir(relFile)
	if dir != "." && matchIgnoredDir(dir, patterns) {
		return true
	}
	base := path.Base(relFile)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Exact basename match when pattern has no wildcard and no trailing '/'
		if !strings.ContainsAny(p, "*?[") && !strings.HasSuffix(p, "/") {
			if base == p {
				return true
			}
		}
	}
	return false
}
