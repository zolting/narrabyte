// go
// file: internal/llm/tools/edit_file_tool.go
package tools

import (
	"context"
	"fmt"
	"narrabyte/internal/events"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type EditInput struct {
	FilePath   string `json:"file_path" jsonschema:"description=Path to the file to edit (absolute or relative to the configured project root)"`
	OldString  string `json:"old_string" jsonschema:"description=The exact text to replace. Leave empty to overwrite the entire file with new_string."`
	NewString  string `json:"new_string" jsonschema:"description=The replacement text"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"description=Replace all occurrences of old_string instead of a single instance"`
}

type EditOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func Edit(ctx context.Context, in *EditInput) (*EditOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Edit: starting"))

	// Validate input
	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: input is required"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	p := strings.TrimSpace(in.FilePath)
	if p == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: file_path is required"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	// If both strings provided, they must differ
	if strings.TrimSpace(in.OldString) != "" && in.OldString == in.NewString {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: old_string and new_string must be different"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: old_string and new_string must be different",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	// Resolve base root
	base, err := getListDirectoryBaseRoot(ctx)
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: project root not set"))
		return &EditOutput{
			Title:  "",
			Output: "Format error: project root not set",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	// Resolve target absolute path under base, disallowing escape
	events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("Edit: resolving '%s'", p)))
	var abs string
	if filepath.IsAbs(p) {
		absBase, e1 := filepath.Abs(base)
		absReq, e2 := filepath.Abs(p)
		if e1 != nil || e2 != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: resolve error"))
			return nil, fmt.Errorf("resolve error")
		}
		// Resolve symlinks for consistent comparison
		evalBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			evalBase = absBase
		}
		evalReq, err := filepath.EvalSymlinks(absReq)
		if err != nil {
			// If file doesn't exist yet, fall back to absolute path
			evalReq = absReq
		}

		relToBase, e3 := filepath.Rel(evalBase, evalReq)
		if e3 != nil || strings.HasPrefix(relToBase, "..") {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("Edit: path escapes the configured project root"))
			return &EditOutput{
				Title:  filepath.ToSlash(absReq),
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error":       "format_error",
					"replaced":    "false",
					"occurrences": "0",
				},
			}, nil
		}
		abs = absReq
	} else {
		a, ok := safeJoinUnderBase(base, p)
		if !ok {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("Edit: path escapes the configured project root"))
			return &EditOutput{
				Title:  filepath.ToSlash(p),
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error":       "format_error",
					"replaced":    "false",
					"occurrences": "0",
				},
			}, nil
		}
		abs = a
	}
	title := filepath.ToSlash(abs)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: target '%s'", title)))

	// Directory checks
	dir := filepath.Dir(abs)
	if st, derr := os.Stat(dir); derr != nil || !st.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: directory does not exist"))
		return &EditOutput{
			Title:  title,
			Output: "Format error: directory does not exist",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	if st, err := os.Stat(abs); err == nil && st.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: cannot edit directory"))
		return &EditOutput{
			Title:  title,
			Output: "Format error: cannot edit directory",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}

	// If file exists, block editing of binary files
	if st, err := os.Stat(abs); err == nil && !st.IsDir() {
		if bin, berr := isBinaryFile(abs); berr == nil && bin {
			events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: cannot edit binary file"))
			return &EditOutput{
				Title:  title,
				Output: "Format error: cannot edit binary file",
				Metadata: map[string]string{
					"error":       "format_error",
					"replaced":    "false",
					"occurrences": "0",
				},
			}, nil
		}
	}

	// Overwrite entire file if OldString is empty
	if strings.TrimSpace(in.OldString) == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("Edit: overwriting entire file"))
		if err := os.WriteFile(abs, []byte(in.NewString), 0644); err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Edit: write error: %v", err)))
			return nil, err
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: done for '%s'", title)))
		return &EditOutput{
			Title:  title,
			Output: "Edit success: file overwritten",
			Metadata: map[string]string{
				"error":       "",
				"replaced":    "true",
				"occurrences": "1",
			},
		}, nil
	}

	// Read existing file for search/replace
	contentBytes, rerr := os.ReadFile(abs)
	if rerr != nil {
		// For edit-with-old-string, require existing file
		events.Emit(ctx, events.LLMEventTool, events.NewError("Edit: file does not exist"))
		return &EditOutput{
			Title:  title,
			Output: "Format error: file does not exist",
			Metadata: map[string]string{
				"error":       "format_error",
				"replaced":    "false",
				"occurrences": "0",
			},
		}, nil
	}
	content := string(contentBytes)
	events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("Edit: read %d bytes", len(contentBytes))))

	old := in.OldString
	newVal := in.NewString

	// First, try exact substring matching
	occ := strings.Count(content, old)
	if occ == 0 {
		// Try a flexible multi-line, indentation-tolerant match
		replaced, n, ok := flexibleBlockReplace(content, old, newVal, in.ReplaceAll)
		if !ok {
			events.Emit(ctx, events.LLMEventTool, events.NewInfo("Edit: old_string not found"))
			return &EditOutput{
				Title:  title,
				Output: "Edit error: old_string not found in content",
				Metadata: map[string]string{
					"error":       "search_not_found",
					"replaced":    "false",
					"occurrences": "0",
				},
			}, nil
		}
		if !in.ReplaceAll && n > 1 {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("Edit: ambiguous match without replace_all"))
			return &EditOutput{
				Title:  title,
				Output: "Edit error: old_string found multiple times and requires more code context to uniquely identify the intended match",
				Metadata: map[string]string{
					"error":       "ambiguous_match",
					"replaced":    "false",
					"occurrences": fmt.Sprintf("%d", n),
				},
			}, nil
		}
		if err := os.WriteFile(abs, []byte(replaced), 0644); err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Edit: write error: %v", err)))
			return nil, err
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: replaced %d occurrence(s)", map[bool]int{true: n, false: 1}[in.ReplaceAll])))
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: done for '%s'", title)))
		return &EditOutput{
			Title:  title,
			Output: "Edit success",
			Metadata: map[string]string{
				"error":       "",
				"replaced":    "true",
				"occurrences": fmt.Sprintf("%d", map[bool]int{true: n, false: 1}[in.ReplaceAll]),
			},
		}, nil
	}

	// Exact match path
	if occ > 1 && !in.ReplaceAll {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn("Edit: ambiguous match without replace_all"))
		return &EditOutput{
			Title:  title,
			Output: "Edit error: old_string found multiple times and requires more code context to uniquely identify the intended match",
			Metadata: map[string]string{
				"error":       "ambiguous_match",
				"replaced":    "false",
				"occurrences": fmt.Sprintf("%d", occ),
			},
		}, nil
	}

	var updated string
	var replacedCount int
	if in.ReplaceAll {
		updated = strings.ReplaceAll(content, old, newVal)
		replacedCount = occ
	} else {
		updated = strings.Replace(content, old, newVal, 1)
		replacedCount = 1
	}

	if err := os.WriteFile(abs, []byte(updated), 0644); err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("Edit: write error: %v", err)))
		return nil, err
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: replaced %d occurrence(s)", replacedCount)))
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("Edit: done for '%s'", title)))

	return &EditOutput{
		Title:  title,
		Output: "Edit success",
		Metadata: map[string]string{
			"error":       "",
			"replaced":    "true",
			"occurrences": fmt.Sprintf("%d", replacedCount),
		},
	}, nil
}

// flexibleBlockReplace attempts a multi-line, indentation-tolerant replace.
// Returns (newContent, matches, ok).
func flexibleBlockReplace(content, old, newVal string, replaceAll bool) (string, int, bool) {
	// Try a regex that permits variable indentation at the start of each line,
	// and exact literal content otherwise.
	lines := strings.Split(old, "\n")
	if len(lines) == 0 {
		return "", 0, false
	}

	var b strings.Builder
	// Enable multi-line mode
	b.WriteString("(?m)")
	for i, ln := range lines {
		b.WriteString(`\s*`)
		b.WriteString(regexp.QuoteMeta(ln))
		if i < len(lines)-1 {
			b.WriteString(`\s*\n`)
		}
	}
	re, err := regexp.Compile(b.String())
	if err != nil {
		return "", 0, false
	}

	idx := re.FindAllStringIndex(content, -1)
	if len(idx) == 0 {
		return "", 0, false
	}

	if replaceAll {
		return re.ReplaceAllString(content, newVal), len(idx), true
	}

	// Replace only the first match
	first := idx[0]
	var out strings.Builder
	out.Grow(len(content) - (first[1] - first[0]) + len(newVal))
	out.WriteString(content[:first[0]])
	out.WriteString(newVal)
	out.WriteString(content[first[1]:])
	return out.String(), len(idx), true
}
