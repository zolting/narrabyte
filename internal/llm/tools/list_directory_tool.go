package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"narrabyte/internal/utils"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// ---------- Public API ----------

type TreeOptions struct {
	MaxDepth           int      // default 2; 1 = items directly under root
	MaxEntries         int      // default 800 (hard cap on total nodes)
	ExtraExcludeGlobs  []string // appended to .gitignore patterns
	CollapseDirEntries int      // default 100; mark dirs "collapsed" if >= this many children
}

type ListDirectoryInput struct {
	DirectoryRelativePath string `json:"directory_relative_path" jsonschema:"description=The directory path relative to the project root"`
}

type TreeNode struct {
	ID            int       `json:"id"`
	Path          string    `json:"path"`                     // repo-relative (unix-style)
	Type          string    `json:"type"`                     // "dir" | "file" | "symlink"
	Depth         int       `json:"depth"`                    // 1 = entries under repo root
	ChildrenCount int       `json:"children_count,omitempty"` // for directories
	Collapsed     bool      `json:"collapsed,omitempty"`      // depth limit / too many children
	Ext           string    `json:"ext,omitempty"`            // lowercase (files)
	Size          int64     `json:"size,omitempty"`           // bytes (files)
	MTime         time.Time `json:"mtime,omitempty"`          // (files)
}

type TreeLangCount struct {
	Lang  string `json:"lang"`
	Files int    `json:"files"`
}

type TreeDirCount struct {
	Path  string `json:"path"`
	Files int    `json:"files"`
}

type TreeResponse struct {
	Version string `json:"version"`
	Root    string `json:"root"`
	Limits  struct {
		MaxDepth   int `json:"max_depth"`
		MaxEntries int `json:"max_entries"`
	} `json:"limits"`
	Truncated struct {
		ByEntries bool `json:"by_entries"`
		ByDepth   bool `json:"by_depth"`
	} `json:"truncated"`
	Ignore struct {
		Gitignore    bool     `json:"gitignore"`
		ExcludeGlobs []string `json:"exclude_globs"`
	} `json:"ignore"`
	Summary struct {
		Dirs        int             `json:"dirs"`
		Files       int             `json:"files"`
		ByLang      []TreeLangCount `json:"by_lang"`
		LargestDirs []TreeDirCount  `json:"largest_dirs"`
	} `json:"summary"`
	Nodes []TreeNode `json:"nodes"`
}

// ListDirectoryJSON returns the tree response
func ListDirectoryJSON(_ context.Context, input *ListDirectoryInput) (*TreeResponse, error) {
	// Resolve the input path relative to the configured base root.
	base := getListDirectoryBaseRoot()
	rel := "."
	if input != nil {
		rel = strings.TrimSpace(input.DirectoryRelativePath)
		if rel == "" {
			rel = "."
		}
	}

	var target string
	if rel == "." {
		// Use base directly
		absBase, err := filepath.Abs(base)
		if err != nil {
			return nil, err
		}
		target = absBase
	} else {
		abs, ok := safeJoinUnderBase(base, rel)
		if !ok {
			return nil, fmt.Errorf("path escapes the configured base root")
		}
		target = abs
	}

	resp, err := buildTree(target, TreeOptions{
		MaxDepth:           2,
		MaxEntries:         800,
		CollapseDirEntries: 100,
		ExtraExcludeGlobs:  []string{},
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ---------- Implementation ----------

func buildTree(repoPath string, opts TreeOptions) (TreeResponse, error) {
	var out TreeResponse

	if repoPath == "" {
		return out, errors.New("repoPath is required")
	}
	absRoot, err := filepath.Abs(repoPath)
	if err != nil {
		return out, err
	}
	if fi, err := os.Stat(absRoot); err != nil || !fi.IsDir() {
		return out, fmt.Errorf("invalid repoPath: %s", repoPath)
	}

	// Defaults
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 2
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 800
	}
	if opts.CollapseDirEntries <= 0 {
		opts.CollapseDirEntries = 100
	}

	// Build a gitignore matcher from the repo, plus extra excludes.
	fs := osfs.New(absRoot)
	patterns, _ := gitignore.ReadPatterns(fs, nil) // reads .gitignore recursively, ordered by priority. :contentReference[oaicite:1]{index=1}

	// Load default excludes from assets file
	defaultExcludes := []string{}
	if root, err := utils.FindProjectRoot(); err == nil {
		ignorePath := filepath.Join(root, "internal", "assets", "treeJsonIgnore.txt")
		if lines, err := utils.ReadNonEmptyLines(ignorePath); err == nil {
			defaultExcludes = lines
		}
	}
	// Fallback to a sensible default if file missing/unreadable
	if len(defaultExcludes) == 0 {
		defaultExcludes = []string{
			"**/.git/**", "**/node_modules/**", "**/dist/**", "**/build/**",
			"**/.next/**", "**/.turbo/**", "**/coverage/**", "**/vendor/**",
		}
	}
	for _, gl := range append(defaultExcludes, opts.ExtraExcludeGlobs...) {
		patterns = append(patterns, gitignore.ParsePattern(gl, nil))
	}
	matcher := gitignore.NewMatcher(patterns)

	// Helper: should we ignore this path?
	ignored := func(rel string, isDir bool) bool {
		rel = filepath.ToSlash(rel)
		segs := strings.Split(rel, "/")
		return matcher.Match(segs, isDir)
	}

	// Wire up response header
	out.Version = "1"
	out.Root = "."
	out.Limits.MaxDepth = opts.MaxDepth
	out.Limits.MaxEntries = opts.MaxEntries
	out.Ignore.Gitignore = true
	out.Ignore.ExcludeGlobs = append([]string(nil), append(defaultExcludes, opts.ExtraExcludeGlobs...)...)

	type counters struct{ dirs, files int }
	var c counters

	id := 0
	truncByEntries := false
	truncByDepth := false

	langMap := map[string]int{}
	dirFileCounts := map[string]int{}

	addFileRollup := func(rel string) {
		parts := strings.Split(rel, "/")
		for i := 0; i < len(parts)-1; i++ {
			d := strings.Join(parts[:i+1], "/")
			if d != "" {
				dirFileCounts[d]++
			}
		}
	}

	// Walk root (depth=1 for entries under repo root)
	var nodes []TreeNode

	var walk func(dir string, depth int) error
	walk = func(dir string, depth int) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil // unreadable -> skip
		}

		// Filter and sort: dirs first, then alpha
		type entry struct {
			os.DirEntry
			abs string
			rel string
		}
		var list []entry
		for _, e := range entries {
			abs := filepath.Join(dir, e.Name())
			rel, _ := filepath.Rel(absRoot, abs)
			rel = filepath.ToSlash(rel)

			// We must pass correct isDir to matcher
			isDir := e.IsDir()
			if ignored(rel, isDir) {
				// if ignored by patterns, skip entirely
				if isDir {
					// do not descend
				}
				continue
			}
			list = append(list, entry{DirEntry: e, abs: abs, rel: rel})
		}

		sort.SliceStable(list, func(i, j int) bool {
			di, dj := list[i].IsDir(), list[j].IsDir()
			if di != dj {
				return di
			}
			return strings.ToLower(list[i].Name()) < strings.ToLower(list[j].Name())
		})

		for _, e := range list {
			if id >= opts.MaxEntries {
				truncByEntries = true
				return nil
			}

			mode := e.Type()
			isLink := (mode & os.ModeSymlink) != 0
			isDir := e.IsDir()

			id++
			n := TreeNode{
				ID:    id,
				Path:  e.rel,
				Depth: depth,
			}

			if isDir {
				n.Type = "dir"

				// Count visible children (post-filter) for this dir
				childEntries, _ := os.ReadDir(e.abs)
				visible := 0
				for _, ce := range childEntries {
					abs := filepath.Join(e.abs, ce.Name())
					rel, _ := filepath.Rel(absRoot, abs)
					if ignored(filepath.ToSlash(rel), ce.IsDir()) {
						continue
					}
					visible++
				}
				n.ChildrenCount = visible

				// Collapsed if depth limit or too many children
				if depth >= opts.MaxDepth {
					n.Collapsed = true
					if visible > 0 {
						truncByDepth = true
					}
				} else if visible >= opts.CollapseDirEntries {
					n.Collapsed = true
				}

				nodes = append(nodes, n)
				c.dirs++

				// Recurse if allowed
				if depth < opts.MaxDepth {
					if err := walk(e.abs, depth+1); err != nil {
						return err
					}
					if truncByEntries {
						return nil
					}
				}
			} else {
				if isLink {
					n.Type = "symlink"
				} else {
					n.Type = "file"
				}
				if info, err := e.Info(); err == nil && !info.IsDir() {
					n.Size = info.Size()
					n.MTime = info.ModTime()
				}
				ext := strings.TrimPrefix(filepath.Ext(e.Name()), ".")
				if ext != "" {
					n.Ext = strings.ToLower(ext)
					langMap[n.Ext]++
				}
				nodes = append(nodes, n)
				c.files++
				addFileRollup(e.rel)
			}
		}
		return nil
	}

	if err := walk(absRoot, 1); err != nil {
		return out, err
	}

	// Summaries
	out.Summary.Dirs = c.dirs
	out.Summary.Files = c.files

	// By language (sorted desc by count, then name)
	type kv struct {
		k string
		v int
	}
	var langs []kv
	for k, v := range langMap {
		langs = append(langs, kv{k, v})
	}
	sort.Slice(langs, func(i, j int) bool {
		if langs[i].v != langs[j].v {
			return langs[i].v > langs[j].v
		}
		return langs[i].k < langs[j].k
	})
	for _, p := range langs {
		out.Summary.ByLang = append(out.Summary.ByLang, TreeLangCount{Lang: p.k, Files: p.v})
	}

	// Largest dirs (top 8 by recursive file count)
	const topN = 8
	var dirs []kv
	for k, v := range dirFileCounts {
		if k != "" && k != "." {
			dirs = append(dirs, kv{k, v})
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i].v != dirs[j].v {
			return dirs[i].v > dirs[j].v
		}
		return dirs[i].k < dirs[j].k
	})
	for i := 0; i < len(dirs) && i < topN; i++ {
		out.Summary.LargestDirs = append(out.Summary.LargestDirs, TreeDirCount{Path: dirs[i].k, Files: dirs[i].v})
	}

	out.Truncated.ByEntries = truncByEntries
	out.Truncated.ByDepth = truncByDepth
	out.Nodes = nodes
	return out, nil
}
