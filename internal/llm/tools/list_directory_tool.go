// go
package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"narrabyte/internal/events"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

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
	// Absolute path to the directory to list.
	Path   string   `json:"path,omitempty" jsonschema:"description=The absolute path to the directory to list"`
	Ignore []string `json:"ignore,omitempty" jsonschema:"description=List of glob-like patterns to ignore"`
}

type ListDirectoryOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ListDirectory produces a simple textual tree listing similar to the TS tool.
func ListDirectory(ctx context.Context, in *ListLSInput) (*ListDirectoryOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("ListDirectory: starting"))

	base, err := getListDirectoryBaseRoot()
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: base root error: %v", err)))
		return &ListDirectoryOutput{
			Title:  "",
			Output: "Format error: project root not set",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	req := "."
	if in != nil && strings.TrimSpace(in.Path) != "" {
		req = strings.TrimSpace(in.Path)
	}
	events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("ListDirectory: resolving '%s' under base '%s'", req, base)))

	// Resolve search path under base (absolute allowed if it resides under base)
	var searchPath string
	if filepath.IsAbs(req) {
		absBase, err := filepath.Abs(base)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: abs base resolve error: %v", err)))
			return &ListDirectoryOutput{
				Title:  filepath.ToSlash(req),
				Output: "Format error: failed to resolve base directory",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		absReq, err := filepath.Abs(req)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: abs req resolve error: %v", err)))
			return &ListDirectoryOutput{
				Title:  filepath.ToSlash(req),
				Output: "Format error: failed to resolve directory path",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		relToBase, err := filepath.Rel(absBase, absReq)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: rel error: %v", err)))
			return &ListDirectoryOutput{
				Title:  filepath.ToSlash(absReq),
				Output: "Format error: failed to resolve relative path",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		if strings.HasPrefix(relToBase, "..") {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("ListDirectory: path escapes configured base root"))
			return &ListDirectoryOutput{
				Title:  filepath.ToSlash(absReq),
				Output: "Format error: path escapes the configured base root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		searchPath = absReq
	} else {
		abs, ok := safeJoinUnderBase(base, req)
		if !ok {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("ListDirectory: path escapes configured base root"))
			return &ListDirectoryOutput{
				Title:  filepath.ToSlash(filepath.Join(base, req)),
				Output: "Format error: path escapes the configured base root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		searchPath = abs
	}
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ListDirectory: listing '%s'", filepath.ToSlash(searchPath))))

	// Ensure directory exists
	info, err := os.Stat(searchPath)
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: stat error: %v", err)))
		return &ListDirectoryOutput{
			Title:  filepath.ToSlash(searchPath),
			Output: fmt.Sprintf("Format error: directory does not exist or is not accessible: %s", filepath.ToSlash(searchPath)),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}
	if !info.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: not a directory: %s", filepath.ToSlash(searchPath))))
		return &ListDirectoryOutput{
			Title:  filepath.ToSlash(searchPath),
			Output: fmt.Sprintf("Format error: path is not a directory: %s", filepath.ToSlash(searchPath)),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Compose ignore patterns
	patterns := append([]string{}, DefaultIgnorePatterns...)
	if in != nil && len(in.Ignore) > 0 {
		patterns = append(patterns, in.Ignore...)
	}

	// Collect files up to limit, honoring ignore patterns
	var files []string // slash-separated paths under searchPath
	err = filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// skip unreadable entries
			if d != nil && d.IsDir() {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("ListDirectory: skipping unreadable dir '%s'", filepath.ToSlash(p))))
				return fs.SkipDir
			}
			events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("ListDirectory: unreadable entry '%s'", filepath.ToSlash(p))))
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
				events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("ListDirectory: ignoring dir '%s'", rel)))
				return fs.SkipDir
			}
			return nil
		}
		// If file is ignored, skip
		if matchIgnoredFile(rel, patterns) {
			events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("ListDirectory: ignoring file '%s'", rel)))
			return nil
		}

		files = append(files, rel)
		if len(files) >= listLimit {
			events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ListDirectory: limit reached at %d files", len(files))))
			// Stop traversal once we've reached the limit
			return errors.New("__LIST_LIMIT_REACHED__")
		}
		return nil
	})
	if err != nil {
		if err.Error() != "__LIST_LIMIT_REACHED__" {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory: traversal error: %v", err)))
			return &ListDirectoryOutput{
				Title:  filepath.ToSlash(searchPath),
				Output: fmt.Sprintf("Format error: failed to traverse directory: %v", err),
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
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

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ListDirectory: done, %d files listed for '%s'", len(files), filepath.ToSlash(searchPath))))
	return &ListDirectoryOutput{
		Title:  filepath.ToSlash(searchPath),
		Output: b.String(),
		Metadata: map[string]string{
			"files_count": fmt.Sprintf("%d", len(files)),
			"limited":     fmt.Sprintf("%v", len(files) >= listLimit),
		},
	}, nil
}

// matchIgnoredDir returns true if the directory (slash-separated under the root) should be ignored.
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

// matchIgnoredFile returns true if the file path (slash-separated under the root) matches a simple ignore.
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
