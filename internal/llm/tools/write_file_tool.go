package tools

import (
	"context"
	"fmt"
	"narrabyte/internal/events"
	"os"
	"path/filepath"
	"strings"
)

type WriteFileInput struct {
	FilePath string `json:"file_path" jsonschema:"description=The absolute path to the file to write"`
	Content  string `json:"content" jsonschema:"description=The content to write to the file"`
}

type WriteFileOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func WriteFile(ctx context.Context, in *WriteFileInput) (*WriteFileOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("WriteFile: starting"))

	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile: input is required"))
		return &WriteFileOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	base, err := getListDirectoryBaseRoot(ctx)
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile: project root not set"))
		return &WriteFileOutput{
			Title:  "",
			Output: "Format error: project root not set",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	p := strings.TrimSpace(in.FilePath)
	if p == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile: file_path is required"))
		return &WriteFileOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	var absPath string
	if filepath.IsAbs(p) {
		absBase, err := filepath.Abs(base)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: base resolve error: %v", err)))
			return nil, err
		}
		// Resolve symlinks for consistent comparison
		evalBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			evalBase = absBase
		}

		absCandidate, err := filepath.Abs(p)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: abs path error: %v", err)))
			return nil, err
		}
		// Resolve symlinks for the candidate path
		evalCandidate, err := filepath.EvalSymlinks(absCandidate)
		if err != nil {
			// If file doesn't exist yet, fall back to absolute path
			evalCandidate = absCandidate
		}

		relToBase, err := filepath.Rel(evalBase, evalCandidate)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: rel error: %v", err)))
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("WriteFile: path escapes the configured project root"))
			return &WriteFileOutput{
				Title:  filepath.ToSlash(absCandidate),
				Output: "Format error: file is not in the configured project root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		absPath = absCandidate
	} else {
		abs, ok := safeJoinUnderBase(base, p)
		if !ok {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("WriteFile: path escapes the configured project root"))
			return &WriteFileOutput{
				Title:  filepath.ToSlash(filepath.Join(base, p)),
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		absPath = abs
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("WriteFile: writing '%s'", filepath.ToSlash(absPath))))

	dir := filepath.Dir(absPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: directory does not exist: %s", filepath.ToSlash(dir))))
			return &WriteFileOutput{
				Title:  filepath.ToSlash(absPath),
				Output: fmt.Sprintf("Format error: directory does not exist: %s", dir),
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: stat error: %v", err)))
		return nil, err
	}
	if !info.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: not a directory: %s", filepath.ToSlash(dir))))
		return &WriteFileOutput{
			Title:  filepath.ToSlash(absPath),
			Output: fmt.Sprintf("Format error: not a directory: %s", dir),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	existed := false
	if st, err := os.Stat(absPath); err == nil && !st.IsDir() {
		existed = true
	}

	if err := os.WriteFile(absPath, []byte(in.Content), 0o644); err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: write error: %v", err)))
		return nil, err
	}

	outputMsg := ""
	if existed {
		outputMsg = fmt.Sprintf("Overwrote file: %s", filepath.ToSlash(absPath))
	} else {
		outputMsg = fmt.Sprintf("Created file: %s", filepath.ToSlash(absPath))
	}
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(outputMsg))
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("WriteFile: done for '%s'", filepath.ToSlash(absPath))))

	return &WriteFileOutput{
		Title:  filepath.ToSlash(absPath),
		Output: outputMsg,
		Metadata: map[string]string{
			"filepath": filepath.ToSlash(absPath),
			"exists":   fmt.Sprintf("%v", existed),
		},
	}, nil
}
