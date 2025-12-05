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
	// Repository must be "docs" - writing to the code repository is not allowed.
	Repository Repository `json:"repository" jsonschema:"enum=docs,description=Must be 'docs' - writing to the code repository is not allowed"`
	// FilePath is the relative path to the file within the docs repository.
	FilePath string `json:"file_path" jsonschema:"description=The path to the file relative to the docs repository root (e.g. 'api/endpoints.md'). NEVER use absolute paths."`
	// Content is the content to write to the file.
	Content string `json:"content" jsonschema:"description=The content to write to the file"`
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

	// Enforce docs-only repository
	if in.Repository != RepositoryDocs {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: repository must be 'docs', got '%s'", in.Repository)))
		return &WriteFileOutput{
			Title:  "",
			Output: fmt.Sprintf("Format error: writing is only allowed to the 'docs' repository, got '%s'", in.Repository),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	pathArg := strings.TrimSpace(in.FilePath)
	if pathArg == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile: file_path is required"))
		return &WriteFileOutput{
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
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: %v", err)))
		return &WriteFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: %v", err),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	displayPath := FormatDisplayPath(in.Repository, pathArg)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("WriteFile: writing '%s'", displayPath)))

	dir := filepath.Dir(absPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: directory does not exist: %s", displayPath)))
			return &WriteFileOutput{
				Title:  displayPath,
				Output: fmt.Sprintf("Format error: directory does not exist for path: %s", displayPath),
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: stat error: %v", err)))
		return nil, err
	}
	if !info.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile: parent is not a directory: %s", displayPath)))
		return &WriteFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: parent path is not a directory: %s", displayPath),
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
		outputMsg = fmt.Sprintf("Overwrote file: %s", displayPath)
	} else {
		outputMsg = fmt.Sprintf("Created file: %s", displayPath)
	}
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(outputMsg))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("WriteFile: done for '%s'", displayPath), "write", displayPath))

	return &WriteFileOutput{
		Title:  displayPath,
		Output: outputMsg,
		Metadata: map[string]string{
			"filepath": displayPath,
			"exists":   fmt.Sprintf("%v", existed),
		},
	}, nil
}
