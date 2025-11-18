package tools

import (
	"context"
	"fmt"
	"narrabyte/internal/events"
	"os"
	"path/filepath"
	"strings"
)

type DeleteFileInput struct {
	FilePath string `json:"file_path" jsonschema:"description=The absolute path to the file to delete"`
}

type DeleteFileOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func DeleteFile(ctx context.Context, in *DeleteFileInput) (*DeleteFileOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("DeleteFile: starting"))

	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile: input is required"))
		return &DeleteFileOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	base, err := getListDirectoryBaseRoot(ctx)
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile: project root not set"))
		return &DeleteFileOutput{
			Title:  "",
			Output: "Format error: project root not set",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	p := strings.TrimSpace(in.FilePath)
	if p == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile: file_path is required"))
		return &DeleteFileOutput{
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
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: base resolve error: %v", err)))
			return nil, err
		}
		// Resolve symlinks for consistent comparison
		evalBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			evalBase = absBase
		}

		absCandidate, err := filepath.Abs(p)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: abs path error: %v", err)))
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
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: rel error: %v", err)))
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("DeleteFile: path escapes the configured project root"))
			return &DeleteFileOutput{
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
			events.Emit(ctx, events.LLMEventTool, events.NewWarn("DeleteFile: path escapes the configured project root"))
			return &DeleteFileOutput{
				Title:  filepath.ToSlash(filepath.Join(base, p)),
				Output: "Format error: path escapes the configured project root",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		absPath = abs
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("DeleteFile: deleting '%s'", filepath.ToSlash(absPath))))

	// Check if file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: file does not exist: %s", filepath.ToSlash(absPath))))
			return &DeleteFileOutput{
				Title:  filepath.ToSlash(absPath),
				Output: fmt.Sprintf("Error: file does not exist: %s", absPath),
				Metadata: map[string]string{
					"error": "file_not_found",
				},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: stat error: %v", err)))
		return nil, err
	}
	if info.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: path is a directory: %s", filepath.ToSlash(absPath))))
		return &DeleteFileOutput{
			Title:  filepath.ToSlash(absPath),
			Output: fmt.Sprintf("Error: path is a directory: %s", absPath),
			Metadata: map[string]string{
				"error": "is_directory",
			},
		}, nil
	}

	if err := os.Remove(absPath); err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: delete error: %v", err)))
		return nil, err
	}

	outputMsg := fmt.Sprintf("Deleted file: %s", filepath.ToSlash(absPath))
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(outputMsg))
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("DeleteFile: done for '%s'", filepath.ToSlash(absPath))))

	return &DeleteFileOutput{
		Title:  filepath.ToSlash(absPath),
		Output: outputMsg,
		Metadata: map[string]string{
			"filepath": filepath.ToSlash(absPath),
			"deleted":  "true",
		},
	}, nil
}
