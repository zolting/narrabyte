package tools

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    filepathx "github.com/yargevad/filepathx"
)

const globResultLimit = 100

type GlobInput struct {
    // Pattern is the glob to match files against (supports **).
    Pattern string `json:"pattern" jsonschema:"description=The glob pattern to match files against"`
    // Path is an absolute directory to search. If omitted, the configured base root is used.
    Path string `json:"path,omitempty" jsonschema:"description=Absolute directory to search. Omit to use the configured project root."`
}

type GlobOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Glob finds files matching a glob pattern under a directory, limited and sorted by mtime desc.
func Glob(_ context.Context, in *GlobInput) (*GlobOutput, error) {
    println("Glob input: ", in.Pattern, in.Path)
    if in == nil {
        return &GlobOutput{
            Title:  "",
            Output: "Format error: input is required",
            Metadata: map[string]string{
                "error":     "format_error",
                "count":     "0",
                "truncated": "false",
            },
        }, nil
    }
    pattern := strings.TrimSpace(in.Pattern)
    if pattern == "" {
        return &GlobOutput{
            Title:  strings.TrimSpace(in.Path),
            Output: "Format error: pattern is required",
            Metadata: map[string]string{
                "error":     "format_error",
                "count":     "0",
                "truncated": "false",
            },
        }, nil
    }

    base, err := getListDirectoryBaseRoot()
    if err != nil {
        return &GlobOutput{
            Title:  strings.TrimSpace(in.Path),
            Output: "Format error: project root not set",
            Metadata: map[string]string{
                "error":     "format_error",
                "count":     "0",
                "truncated": "false",
            },
        }, nil
    }

	// Resolve search directory under base
	search := strings.TrimSpace(in.Path)
	var searchPath string
    if search == "" {
        searchPath = base
    } else if filepath.IsAbs(search) {
        absBase, err := filepath.Abs(base)
        if err != nil {
            return &GlobOutput{
                Title:  search,
                Output: "Format error: invalid project root",
                Metadata: map[string]string{
                    "error":     "format_error",
                    "count":     "0",
                    "truncated": "false",
                },
            }, nil
        }
        absReq, err := filepath.Abs(search)
        if err != nil {
            return &GlobOutput{
                Title:  search,
                Output: "Format error: invalid search path",
                Metadata: map[string]string{
                    "error":     "format_error",
                    "count":     "0",
                    "truncated": "false",
                },
            }, nil
        }
        relToBase, err := filepath.Rel(absBase, absReq)
        if err != nil {
            return &GlobOutput{
                Title:  search,
                Output: "Format error: invalid search path",
                Metadata: map[string]string{
                    "error":     "format_error",
                    "count":     "0",
                    "truncated": "false",
                },
            }, nil
        }
        if strings.HasPrefix(relToBase, "..") {
            return &GlobOutput{
                Title:  search,
                Output: "Format error: path escapes the configured project root",
                Metadata: map[string]string{
                    "error":     "format_error",
                    "count":     "0",
                    "truncated": "false",
                },
            }, nil
        }
        searchPath = absReq
    } else {
        abs, ok := safeJoinUnderBase(base, search)
        if !ok {
            return &GlobOutput{
                Title:  search,
                Output: "Format error: path escapes the configured project root",
                Metadata: map[string]string{
                    "error":     "format_error",
                    "count":     "0",
                    "truncated": "false",
                },
            }, nil
        }
        searchPath = abs
    }

	// Ensure directory exists
    info, err := os.Stat(searchPath)
    if err != nil {
        return &GlobOutput{
            Title:  filepath.ToSlash(searchPath),
            Output: "Format error: path does not exist or is not accessible",
            Metadata: map[string]string{
                "error":     "format_error",
                "count":     "0",
                "truncated": "false",
            },
        }, nil
    }
    if !info.IsDir() {
        return &GlobOutput{
            Title:  filepath.ToSlash(searchPath),
            Output: "Format error: not a directory",
            Metadata: map[string]string{
                "error":     "format_error",
                "count":     "0",
                "truncated": "false",
            },
        }, nil
    }

	// Build absolute pattern rooted at searchPath
	absPattern := pattern
	if !filepath.IsAbs(pattern) {
		absPattern = filepath.Join(searchPath, pattern)
	}

	// Expand glob
    matches, err := filepathx.Glob(absPattern)
    if err != nil {
        return &GlobOutput{
            Title:  filepath.ToSlash(strings.TrimPrefix(searchPath, base+string(os.PathSeparator))),
            Output: "Format error: invalid glob pattern",
            Metadata: map[string]string{
                "error":     "format_error",
                "count":     "0",
                "truncated": "false",
            },
        }, nil
    }

	type fileInfo struct {
		path  string
		mtime int64
	}
	files := make([]fileInfo, 0, len(matches))
	truncated := false
	for _, p := range matches {
		st, err := os.Stat(p)
		if err != nil {
			// skip unreadable entries
			continue
		}
		if st.IsDir() {
			continue
		}
		files = append(files, fileInfo{path: p, mtime: st.ModTime().UnixNano()})
		if len(files) >= globResultLimit {
			truncated = true
			break
		}
	}

	sort.Slice(files, func(i, j int) bool { return files[i].mtime > files[j].mtime })

	// Build output
	var lines []string
	if len(files) == 0 {
		lines = append(lines, "No files found")
	} else {
		for _, f := range files {
			lines = append(lines, filepath.Clean(f.path))
		}
		if truncated {
			lines = append(lines, "")
			lines = append(lines, "(Results are truncated. Consider using a more specific path or pattern.)")
		}
	}

    // Title: absolute search path
    relTitle := filepath.ToSlash(searchPath)

	println("Glob output: ", strings.Join(lines, "\n"))

    out := &GlobOutput{
        Title:  relTitle,
        Output: strings.Join(lines, "\n"),
        Metadata: map[string]string{
            "count":     fmt.Sprintf("%d", len(files)),
            "truncated": fmt.Sprintf("%v", truncated),
        },
    }
    return out, nil
}
