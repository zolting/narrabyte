package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const grepResultLimit = 100

type GrepInput struct {
	// Pattern is the regex to search for in file contents.
	Pattern string `json:"pattern" jsonschema:"description=The regex pattern to search for in file contents"`
	// Path is an absolute directory to search. If omitted, the configured base root is used.
	Path string `json:"path,omitempty" jsonschema:"description=Absolute directory to search. If omitted, uses the configured project root."`
	// Include is an optional file glob to include (e.g. "*.js", "*.{ts,tsx}").
	Include string `json:"include,omitempty" jsonschema:"description=Optional file pattern to include in the search (e.g. \"*.js\", \"*.{ts,tsx}\")"`
}

type GrepOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Grep scans files under a directory and searches for a regex pattern.
// It limits results to grepResultLimit, sorted by file mtime desc, and groups by file.
func Grep(ctx context.Context, in *GrepInput) (*GrepOutput, error) {
	if in == nil {
		return &GrepOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}
	pattern := strings.TrimSpace(in.Pattern)
	if pattern == "" {
		return &GrepOutput{
			Title:  "",
			Output: "Format error: pattern is required",
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}

	base, err := getListDirectoryBaseRoot()
	if err != nil {
		return &GrepOutput{
			Title:  pattern,
			Output: "Format error: project root not set",
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
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
			return &GrepOutput{
				Title:  pattern,
				Output: "Format error: invalid project root",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		absReq, err := filepath.Abs(search)
		if err != nil {
			return &GrepOutput{
				Title:  pattern,
				Output: "Format error: invalid search path",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		relToBase, err := filepath.Rel(absBase, absReq)
		if err != nil {
			return &GrepOutput{
				Title:  pattern,
				Output: "Format error: invalid search path",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		if strings.HasPrefix(relToBase, "..") {
			return &GrepOutput{
				Title:  pattern,
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		searchPath = absReq
	} else {
		abs, ok := safeJoinUnderBase(base, search)
		if !ok {
			return &GrepOutput{
				Title:  pattern,
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		searchPath = abs
	}

	// Ensure directory exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return &GrepOutput{
			Title:  pattern,
			Output: "Format error: path does not exist or is not accessible",
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}
	if !info.IsDir() {
		return &GrepOutput{
			Title:  pattern,
			Output: "Format error: not a directory",
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}

	// Prepare include matcher
	include := strings.TrimSpace(in.Include)
	var includeMatchers []*regexp.Regexp
	if include != "" {
		for _, p := range expandBraces(include) {
			rx, err := globToRegexp(p)
			if err == nil {
				includeMatchers = append(includeMatchers, rx)
			}
		}
	}

	// Check for context cancellation early
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	// Compile content regex
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return &GrepOutput{
			Title:  pattern,
			Output: "Format error: invalid regex pattern",
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}

	type match struct {
		path    string
		lineNum int
		line    string
		mtime   int64
	}
	var matches []match

	// Walk filesystem
	err = filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Compute path under searchPath using forward slashes
		rel, _ := filepath.Rel(searchPath, p)
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." || rel == "" {
				return nil
			}
			// Skip ignored directories using default patterns
			if matchIgnoredDir(rel, DefaultIgnorePatterns) {
				return fs.SkipDir
			}
			return nil
		}

		// Optionally filter by include pattern(s)
		if len(includeMatchers) > 0 {
			// Try rel path and base name
			baseName := path.Base(rel)
			ok := false
			for _, m := range includeMatchers {
				if m.MatchString(rel) || m.MatchString(baseName) {
					ok = true
					break
				}
			}
			if !ok {
				return nil
			}
		}

		// Skip binary files
		if bin, berr := isBinaryFile(p); berr == nil && bin {
			return nil
		}

		// Stats for mtime
		st, err := os.Stat(p)
		if err != nil {
			return nil
		}

		// Scan file lines
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		// Raise the scanner buffer limit for long lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 2*1024*1024)
		lineNum := 0
		for scanner.Scan() {
			if ctx != nil {
				select {
				case <-ctx.Done():
					f.Close()
					return ctx.Err()
				default:
				}
			}
			lineNum++
			lineText := scanner.Text()
			if rx.MatchString(lineText) {
				matches = append(matches, match{
					path:    p,
					lineNum: lineNum,
					line:    lineText,
					mtime:   st.ModTime().UnixNano(),
				})
			}
		}
		// ignore scanner errors for now; continue
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		// Operational error, not input format; return as an error
		return nil, err
	}

	if len(matches) == 0 {
		return &GrepOutput{
			Title:  pattern,
			Output: "No files found",
			Metadata: map[string]string{
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].mtime == matches[j].mtime {
			return matches[i].path < matches[j].path
		}
		return matches[i].mtime > matches[j].mtime
	})

	truncated := false
	if len(matches) > grepResultLimit {
		matches = matches[:grepResultLimit]
		truncated = true
	}

	var outLines []string
	outLines = append(outLines, fmt.Sprintf("Found %d matches", len(matches)))
	current := ""
	for _, m := range matches {
		if m.path != current {
			if current != "" {
				outLines = append(outLines, "")
			}
			current = m.path
			outLines = append(outLines, fmt.Sprintf("%s:", filepath.Clean(m.path)))
		}
		outLines = append(outLines, fmt.Sprintf("  Line %d: %s", m.lineNum, m.line))
	}
	if truncated {
		outLines = append(outLines, "")
		outLines = append(outLines, "(Results are truncated. Consider using a more specific path or pattern.)")
	}

	return &GrepOutput{
		Title:  pattern,
		Output: strings.Join(outLines, "\n"),
		Metadata: map[string]string{
			"matches":   fmt.Sprintf("%d", len(matches)),
			"truncated": fmt.Sprintf("%v", truncated),
		},
	}, nil
}

// expandBraces expands simple comma-separated brace sections like *.{ts,tsx}.
// Nested braces are supported recursively; unmatched braces are left as-is.
func expandBraces(p string) []string {
	// find first '{'
	i := strings.IndexByte(p, '{')
	if i < 0 {
		return []string{p}
	}
	// find matching '}' after i
	j := strings.IndexByte(p[i+1:], '}')
	if j < 0 {
		return []string{p}
	}
	j = i + 1 + j
	head := p[:i]
	body := p[i+1 : j]
	tail := p[j+1:]
	parts := strings.Split(body, ",")
	var out []string
	for _, part := range parts {
		out = append(out, expandBraces(head+part+tail)...)
	}
	return out
}

// globToRegexp converts a glob pattern with **, *, ? into a regexp matching slash-separated paths.
func globToRegexp(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch c {
		case '*':
			// check for **
			if i+1 < len(glob) && glob[i+1] == '*' {
				// consume the second *
				i++
				b.WriteString(".*")
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '^', '$', '|', '[', ']', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
