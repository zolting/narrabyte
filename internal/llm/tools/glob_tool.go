// go
package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
	filepathx "github.com/yargevad/filepathx"
	"narrabyte/internal/events"
)

const globResultLimit = 100

type GlobInput struct {
	// Repository specifies which repository the path is relative to.
	Repository Repository `json:"repository" jsonschema:"enum=docs,enum=code,description=Which repository the path is relative to: 'docs' for documentation repository or 'code' for the codebase repository"`
	// Pattern is the glob to match files against (supports **).
	Pattern string `json:"pattern" jsonschema:"description=The glob pattern to match files against"`
	// Path is a relative directory within the repository to search. If omitted, the repository root is used.
	Path string `json:"path,omitempty" jsonschema:"description=Relative directory within the repository to search. Omit or use empty string for repository root. NEVER use absolute paths."`
}

type GlobOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Glob finds files matching a glob pattern under a directory, limited and sorted by mtime desc.
func Glob(ctx context.Context, in *GlobInput) (*GlobOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Glob: starting"))

	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Glob: input is required"))
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

	// Validate repository
	if !in.Repository.IsValid() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Glob: invalid repository '%s'", in.Repository)))
		return &GlobOutput{
			Title:  "",
			Output: fmt.Sprintf("Format error: invalid repository '%s'; must be 'docs' or 'code'", in.Repository),
			Metadata: map[string]string{
				"error":     "format_error",
				"count":     "0",
				"truncated": "false",
			},
		}, nil
	}

	pattern := strings.TrimSpace(in.Pattern)
	if pattern == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Glob: pattern is required"))
		return &GlobOutput{
			Title:  FormatDisplayPath(in.Repository, in.Path),
			Output: "Format error: pattern is required",
			Metadata: map[string]string{
				"error":     "format_error",
				"count":     "0",
				"truncated": "false",
			},
		}, nil
	}

	pathArg := strings.TrimSpace(in.Path)
	if pathArg == "" {
		pathArg = "."
	}

	// Resolve path using the repository-scoped resolver
	searchPath, err := ResolveRepositoryPath(ctx, in.Repository, pathArg)
	if err != nil {
		displayPath := FormatDisplayPath(in.Repository, pathArg)
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Glob: %v", err)))
		return &GlobOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: %v", err),
			Metadata: map[string]string{
				"error":     "format_error",
				"count":     "0",
				"truncated": "false",
			},
		}, nil
	}

	displayPath := FormatDisplayPath(in.Repository, pathArg)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Glob: searching in '%s'", displayPath)))

	ignorePatterns := append([]string{}, DefaultIgnorePatterns...)
	ignorePatterns = append(ignorePatterns, scopedIgnorePatterns(ctx)...)

	type fileInfo struct {
		path  string
		mtime int64
	}
	var (
		files     []fileInfo
		truncated bool
	)

	// Use git snapshot only for code repository when a snapshot is configured
	if in.Repository == RepositoryCode {
		if snapshot := currentGitSnapshot(ctx); snapshot != nil {
			rel, relErr := snapshot.relativeFromAbs(searchPath)
			if relErr != nil {
				if errors.Is(relErr, ErrSnapshotEscapes) {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn("Glob: path escapes git snapshot root"))
					return &GlobOutput{
						Title:  displayPath,
						Output: "Format error: path escapes the configured project root",
						Metadata: map[string]string{
							"error":     "format_error",
							"count":     "0",
							"truncated": "false",
						},
					}, nil
				}
				events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Glob: snapshot rel path error: %v", relErr)))
				return &GlobOutput{
					Title:  displayPath,
					Output: "Format error: failed to resolve path within repository snapshot",
					Metadata: map[string]string{
						"error":     "format_error",
						"count":     "0",
						"truncated": "false",
					},
				}, nil
			}

			absPattern := pattern
			if !filepath.IsAbs(pattern) {
				absPattern = filepath.Join(searchPath, pattern)
			}
			slashPattern := filepath.ToSlash(absPattern)
			rx, rxErr := globToRegexp(slashPattern)
			if rxErr != nil {
				events.Emit(ctx, events.LLMEventTool, events.NewError("Glob: invalid glob pattern"))
				return &GlobOutput{
					Title:  displayPath,
					Output: "Format error: invalid glob pattern",
					Metadata: map[string]string{
						"error":     "format_error",
						"count":     "0",
						"truncated": "false",
					},
				}, nil
			}

			walkErr := snapshot.walkFiles(rel, func(relPath string, entry GitTreeEntry, file *object.File) error {
				if ctx != nil {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
				}
				if matchIgnoredFile(relPath, ignorePatterns) {
					return nil
				}
				absCandidate := filepath.Join(searchPath, filepath.FromSlash(relPath))
				slashCandidate := filepath.ToSlash(absCandidate)
				if rx.MatchString(slashCandidate) {
					files = append(files, fileInfo{path: absCandidate, mtime: 0})
					if len(files) >= globResultLimit {
						truncated = true
						return errListLimitReached
					}
				}
				return nil
			})
			if walkErr != nil {
				if errors.Is(walkErr, errListLimitReached) {
					// limit reached; continue with collected results
				} else if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
					return nil, walkErr
				} else if errors.Is(walkErr, ErrSnapshotNotFound) {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn("Glob: directory not found in snapshot"))
					return &GlobOutput{
						Title:  displayPath,
						Output: "Format error: directory does not exist in the repository snapshot",
						Metadata: map[string]string{
							"error":     "format_error",
							"count":     "0",
							"truncated": "false",
						},
					}, nil
				} else {
					events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Glob: snapshot traversal error: %v", walkErr)))
					return &GlobOutput{
						Title:  displayPath,
						Output: fmt.Sprintf("Format error: failed to evaluate glob in snapshot: %v", walkErr),
						Metadata: map[string]string{
							"error":     "format_error",
							"count":     "0",
							"truncated": "false",
						},
					}, nil
				}
			}

			// Jump to output building for snapshot case
			goto buildOutput
		}
	}

	// Fall through to filesystem search (for docs repository, or code when no snapshot)
	{
		info, statErr := os.Stat(searchPath)
		if statErr != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Glob: path does not exist or is not accessible"))
			return &GlobOutput{
				Title:  displayPath,
				Output: "Format error: path does not exist or is not accessible",
				Metadata: map[string]string{
					"error":     "format_error",
					"count":     "0",
					"truncated": "false",
				},
			}, nil
		}
		if !info.IsDir() {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Glob: not a directory"))
			return &GlobOutput{
				Title:  displayPath,
				Output: "Format error: not a directory",
				Metadata: map[string]string{
					"error":     "format_error",
					"count":     "0",
					"truncated": "false",
				},
			}, nil
		}

		absPattern := pattern
		if !filepath.IsAbs(pattern) {
			absPattern = filepath.Join(searchPath, pattern)
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Glob: using pattern '%s'", filepath.ToSlash(absPattern))))

		matches, matchErr := filepathx.Glob(absPattern)
		if matchErr != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Glob: invalid glob pattern"))
			return &GlobOutput{
				Title:  displayPath,
				Output: "Format error: invalid glob pattern",
				Metadata: map[string]string{
					"error":     "format_error",
					"count":     "0",
					"truncated": "false",
				},
			}, nil
		}

		files = make([]fileInfo, 0, len(matches))
		for _, p := range matches {
			st, stErr := os.Stat(p)
			if stErr != nil {
				continue
			}
			if st.IsDir() {
				continue
			}
			relToSearch, relErr := filepath.Rel(searchPath, p)
			if relErr != nil {
				relToSearch = p
			}
			relToSearch = filepath.ToSlash(relToSearch)
			if matchIgnoredFile(relToSearch, ignorePatterns) {
				continue
			}
			files = append(files, fileInfo{path: p, mtime: st.ModTime().UnixNano()})
			if len(files) >= globResultLimit {
				truncated = true
				break
			}
		}
	}

buildOutput:

	sort.Slice(files, func(i, j int) bool {
		if files[i].mtime == files[j].mtime {
			return files[i].path < files[j].path
		}
		return files[i].mtime > files[j].mtime
	})

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

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Glob: matched %d file(s)%s", len(files), map[bool]string{true: " (truncated)", false: ""}[truncated])))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("Glob: done for '%s'", displayPath), "glob", displayPath))

	out := &GlobOutput{
		Title:  displayPath,
		Output: strings.Join(lines, "\n"),
		Metadata: map[string]string{
			"count":     fmt.Sprintf("%d", len(files)),
			"truncated": fmt.Sprintf("%v", truncated),
		},
	}
	return out, nil
}
