// go
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

	"github.com/go-git/go-git/v5/plumbing/object"
	"narrabyte/internal/events"
)

const grepResultLimit = 100

type GrepInput struct {
	// Repository specifies which repository the path is relative to.
	Repository Repository `json:"repository" jsonschema:"enum=docs,enum=code,description=Which repository the path is relative to: 'docs' for documentation repository or 'code' for the codebase repository"`
	// Pattern is the regex to search for in file contents.
	Pattern string `json:"pattern" jsonschema:"description=The regex pattern to search for in file contents"`
	// Path is a relative directory within the repository to search. If omitted, the repository root is used.
	Path string `json:"path,omitempty" jsonschema:"description=Relative directory within the repository to search. Omit or use empty string for repository root. NEVER use absolute paths."`
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
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Grep: starting"))

	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Grep: input is required"))
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

	// Validate repository
	if !in.Repository.IsValid() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Grep: invalid repository '%s'", in.Repository)))
		return &GrepOutput{
			Title:  "",
			Output: fmt.Sprintf("Format error: invalid repository '%s'; must be 'docs' or 'code'", in.Repository),
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}

	pattern := strings.TrimSpace(in.Pattern)
	if pattern == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Grep: pattern is required"))
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
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Grep: pattern '%s', include '%s'", pattern, strings.TrimSpace(in.Include))))

	pathArg := strings.TrimSpace(in.Path)
	if pathArg == "" {
		pathArg = "."
	}

	// Resolve path using the repository-scoped resolver
	searchPath, err := ResolveRepositoryPath(ctx, in.Repository, pathArg)
	if err != nil {
		displayPath := FormatDisplayPath(in.Repository, pathArg)
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Grep: %v", err)))
		return &GrepOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: %v", err),
			Metadata: map[string]string{
				"error":     "format_error",
				"matches":   "0",
				"truncated": "false",
			},
		}, nil
	}

	displayPath := FormatDisplayPath(in.Repository, pathArg)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Grep: searching in '%s'", displayPath)))

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
	if include != "" {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Grep: include filter '%s'", include)))
	}

	ignorePatterns := append([]string{}, DefaultIgnorePatterns...)
	ignorePatterns = append(ignorePatterns, scopedIgnorePatterns(ctx)...)

	// Check for context cancellation early
	if ctx != nil {
		select {
		case <-ctx.Done():
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("Grep: canceled"))
			return nil, ctx.Err()
		default:
		}
	}

	// Compile content regex
	rx, err := regexp.Compile(pattern)
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Grep: invalid regex pattern"))
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

	// Use git snapshot only for code repository when a snapshot is configured
	if in.Repository == RepositoryCode {
		if snapshot := currentGitSnapshot(ctx); snapshot != nil {
			rel, relErr := snapshot.relativeFromAbs(searchPath)
			if relErr != nil {
				if errors.Is(relErr, ErrSnapshotEscapes) {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn("Grep: path escapes git snapshot root"))
					return &GrepOutput{
						Title:  displayPath,
						Output: "Format error: path escapes the configured project root",
						Metadata: map[string]string{
							"error":     "format_error",
							"matches":   "0",
							"truncated": "false",
						},
					}, nil
				}
				events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Grep: snapshot rel path error: %v", relErr)))
				return &GrepOutput{
					Title:  displayPath,
					Output: "Format error: failed to resolve path within repository snapshot",
					Metadata: map[string]string{
						"error":     "format_error",
						"matches":   "0",
						"truncated": "false",
					},
				}, nil
			}

			if _, treeErr := snapshot.treeFor(rel); treeErr != nil {
				if errors.Is(treeErr, ErrSnapshotNotFound) {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn("Grep: not a directory in snapshot"))
					return &GrepOutput{
						Title:  displayPath,
						Output: "Format error: path does not refer to a directory in the repository snapshot",
						Metadata: map[string]string{
							"error":     "format_error",
							"matches":   "0",
							"truncated": "false",
						},
					}, nil
				}
				return nil, treeErr
			}

		var walk func(commitPath, displayPath string) error
		walk = func(commitPath, displayPath string) error {
			if ctx != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
			entries, listErr := snapshot.list(commitPath)
			if listErr != nil {
				return listErr
			}
			for _, entry := range entries {
				relPath := joinCommitPath(displayPath, entry.Name)
				if entry.IsDir() {
					if matchIgnoredDir(relPath, ignorePatterns) {
						continue
					}
					if err := walk(entry.Path, relPath); err != nil {
						return err
					}
					continue
				}
				if !entry.IsFile() {
					continue
				}
				if matchIgnoredFile(relPath, ignorePatterns) {
					continue
				}
				if len(includeMatchers) > 0 {
					baseName := path.Base(relPath)
					matched := false
					for _, m := range includeMatchers {
						if m.MatchString(relPath) || m.MatchString(baseName) {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
				}

				file, fileErr := snapshot.commit.File(entry.Path)
				if fileErr != nil {
					if !errors.Is(fileErr, object.ErrFileNotFound) {
						events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Grep: unable to read '%s' from snapshot: %v", entry.Path, fileErr)))
					}
					continue
				}
				isBinary, binErr := file.IsBinary()
				if binErr != nil {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Grep: binary check failed for '%s': %v", entry.Path, binErr)))
					continue
				}
				if isBinary {
					continue
				}

				reader, rdrErr := file.Reader()
				if rdrErr != nil {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Grep: reader error for '%s': %v", entry.Path, rdrErr)))
					continue
				}

				scanner := bufio.NewScanner(reader)
				buf := make([]byte, 0, 64*1024)
				scanner.Buffer(buf, 2*1024*1024)
				lineNum := 0
				absCandidate := filepath.Join(searchPath, filepath.FromSlash(relPath))
				for scanner.Scan() {
					if ctx != nil {
						select {
						case <-ctx.Done():
							reader.Close()
							return ctx.Err()
						default:
						}
					}
					lineNum++
					lineText := scanner.Text()
					if rx.MatchString(lineText) {
						matches = append(matches, match{
							path:    absCandidate,
							lineNum: lineNum,
							line:    lineText,
							mtime:   0,
						})
					}
				}
				reader.Close()
				if scanErr := scanner.Err(); scanErr != nil {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Grep: scanner error for '%s': %v", absCandidate, scanErr)))
				}
			}
			return nil
		}

		if walkErr := walk(rel, ""); walkErr != nil {
			if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
				return nil, walkErr
			}
			if errors.Is(walkErr, ErrSnapshotNotFound) {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("Grep: directory not found in snapshot"))
				return &GrepOutput{
					Title:  displayPath,
					Output: "Format error: path does not exist in the repository snapshot",
					Metadata: map[string]string{
						"error":     "format_error",
						"matches":   "0",
						"truncated": "false",
					},
				}, nil
			}
			return nil, walkErr
		}

			// Jump to output building for snapshot case
			goto buildOutput
		}
	}

	// Fall through to filesystem search (for docs repository, or code when no snapshot)
	{
		// Ensure directory exists
		info, statErr := os.Stat(searchPath)
		if statErr != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Grep: path does not exist or is not accessible"))
			return &GrepOutput{
				Title:  displayPath,
				Output: "Format error: path does not exist or is not accessible",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		if !info.IsDir() {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Grep: not a directory"))
			return &GrepOutput{
				Title:  displayPath,
				Output: "Format error: not a directory",
				Metadata: map[string]string{
					"error":     "format_error",
					"matches":   "0",
					"truncated": "false",
				},
			}, nil
		}
		walkErr := filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Grep: skipping unreadable dir '%s'", filepath.ToSlash(p))))
					return fs.SkipDir
				}
				events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("Grep: unreadable entry '%s'", filepath.ToSlash(p))))
				return nil
			}

			rel, _ := filepath.Rel(searchPath, p)
			rel = filepath.ToSlash(rel)

			if d.IsDir() {
				if rel == "." || rel == "" {
					return nil
				}
				if matchIgnoredDir(rel, ignorePatterns) {
					return fs.SkipDir
				}
				return nil
			}

			if matchIgnoredFile(rel, ignorePatterns) {
				return nil
			}

			if len(includeMatchers) > 0 {
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

			if bin, berr := isBinaryFile(p); berr == nil && bin {
				return nil
			}

			st, err := os.Stat(p)
			if err != nil {
				return nil
			}

			f, err := os.Open(p)
			if err != nil {
				return nil
			}

			scanner := bufio.NewScanner(f)
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
			f.Close()
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, context.Canceled) {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Grep: traversal error: %v", walkErr)))
			return nil, walkErr
		}
	}

buildOutput:

	if len(matches) == 0 {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("Grep: no matches"))
		events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("Grep: done for '%s'", displayPath), "grep", displayPath))
		return &GrepOutput{
			Title:  displayPath,
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

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Grep: matched %d item(s)%s", len(matches), map[bool]string{true: " (truncated)", false: ""}[truncated])))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("Grep: done for '%s'", displayPath), "grep", displayPath))

	return &GrepOutput{
		Title:  displayPath,
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
