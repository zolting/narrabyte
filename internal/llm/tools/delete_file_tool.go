package tools

import (
	"context"
	"fmt"
	"narrabyte/internal/events"
	"os"
	"strings"
)

type DeleteFileInput struct {
	// Repository must be "docs" - deleting from the code repository is not allowed.
	Repository Repository `json:"repository" jsonschema:"enum=docs,description=Must be 'docs' - deleting from the code repository is not allowed"`
	// FilePath is the relative path to the file within the docs repository.
	FilePath string `json:"file_path" jsonschema:"description=The path to the file relative to the docs repository root (e.g. 'api/old-endpoints.md'). NEVER use absolute paths."`
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

	// Enforce docs-only repository
	if in.Repository != RepositoryDocs {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: repository must be 'docs', got '%s'", in.Repository)))
		return &DeleteFileOutput{
			Title:  "",
			Output: fmt.Sprintf("Format error: deleting is only allowed in the 'docs' repository, got '%s'", in.Repository),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	pathArg := strings.TrimSpace(in.FilePath)
	if pathArg == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile: file_path is required"))
		return &DeleteFileOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Resolve path using the repository-scoped resolver
	absPath, err := ResolveRepositoryPath(ctx, in.Repository, pathArg)
	if err != nil {
		displayPath := FormatDisplayPath(in.Repository, pathArg)
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: %v", err)))
		return &DeleteFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: %v", err),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	displayPath := FormatDisplayPath(in.Repository, pathArg)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("DeleteFile: deleting '%s'", displayPath)))

	// Check if file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: file does not exist: %s", displayPath)))
			return &DeleteFileOutput{
				Title:  displayPath,
				Output: fmt.Sprintf("Error: file does not exist: %s", displayPath),
				Metadata: map[string]string{
					"error": "file_not_found",
				},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: stat error: %v", err)))
		return nil, err
	}
	if info.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: path is a directory: %s", displayPath)))
		return &DeleteFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Error: path is a directory: %s", displayPath),
			Metadata: map[string]string{
				"error": "is_directory",
			},
		}, nil
	}

	if err := os.Remove(absPath); err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile: delete error: %v", err)))
		return nil, err
	}

	outputMsg := fmt.Sprintf("Deleted file: %s", displayPath)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(outputMsg))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("DeleteFile: done for '%s'", displayPath), "delete", displayPath))

	return &DeleteFileOutput{
		Title:  displayPath,
		Output: outputMsg,
		Metadata: map[string]string{
			"filepath": displayPath,
			"deleted":  "true",
		},
	}, nil
}
