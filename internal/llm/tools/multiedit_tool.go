// go
// file: internal/llm/tools/multiedit_tool.go
package tools

import (
	"context"
	"fmt"
	"narrabyte/internal/events"
)

// MultiEditEdit represents a single edit operation within a multi-edit request.
type MultiEditEdit struct {
	// OldString is the exact text to replace. Leave empty to overwrite the entire file.
	OldString string `json:"old_string" jsonschema:"description=The exact text to replace. Leave empty to overwrite the entire file with new_string."`
	// NewString is the replacement text.
	NewString string `json:"new_string" jsonschema:"description=The replacement text"`
	// ReplaceAll replaces all occurrences of old_string instead of just the first.
	ReplaceAll bool `json:"replace_all,omitempty" jsonschema:"description=Replace all occurrences of old_string instead of a single instance"`
}

// MultiEditInput represents the input for the multi-edit tool.
type MultiEditInput struct {
	// Repository must be "docs" - editing the code repository is not allowed.
	Repository Repository `json:"repository" jsonschema:"enum=docs,description=Must be 'docs' - editing the code repository is not allowed"`
	// FilePath is the relative path to the file within the docs repository.
	FilePath string `json:"file_path" jsonschema:"description=The path to the file relative to the docs repository root (e.g. 'api/endpoints.md'). NEVER use absolute paths."`
	// Edits is an array of edit operations to perform sequentially on the file.
	Edits []MultiEditEdit `json:"edits" jsonschema:"description=Array of edit operations to perform sequentially on the file"`
}

// MultiEditResult represents the result of a single edit operation.
type MultiEditResult struct {
	Replaced    bool   `json:"replaced"`
	Occurrences int    `json:"occurrences"`
	Error       string `json:"error,omitempty"`
}

// MultiEditOutput represents the output of the multi-edit tool.
type MultiEditOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Results  []MultiEditResult `json:"results,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MultiEdit performs multiple sequential edit operations on a single file.
// It wraps the standard Edit function and applies each edit in order.
// If any edit fails, the operation stops and returns the error.
func MultiEdit(ctx context.Context, in *MultiEditInput) (*MultiEditOutput, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("MultiEdit: starting"))

	if in == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("MultiEdit: input is required"))
		return &MultiEditOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	if len(in.Edits) == 0 {
		events.Emit(ctx, events.LLMEventTool, events.NewError("MultiEdit: at least one edit is required"))
		return &MultiEditOutput{
			Title:  "",
			Output: "Format error: at least one edit is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	displayPath := FormatDisplayPath(in.Repository, in.FilePath)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("MultiEdit: processing %d edits for '%s'", len(in.Edits), displayPath)))

	var results []MultiEditResult
	var lastOutput *EditOutput

	for i, edit := range in.Edits {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("MultiEdit: applying edit %d/%d", i+1, len(in.Edits))))

		editInput := &EditInput{
			Repository: in.Repository,
			FilePath:   in.FilePath,
			OldString:  edit.OldString,
			NewString:  edit.NewString,
			ReplaceAll: edit.ReplaceAll,
		}

		output, err := Edit(ctx, editInput)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("MultiEdit: edit %d failed with error: %v", i+1, err)))
			return nil, fmt.Errorf("edit %d failed: %w", i+1, err)
		}

		lastOutput = output

		// Parse the metadata to build the result
		result := MultiEditResult{
			Replaced:    output.Metadata["replaced"] == "true",
			Occurrences: 0,
		}

		if output.Metadata["error"] != "" {
			result.Error = output.Metadata["error"]
		}

		// Try to parse occurrences
		if occ := output.Metadata["occurrences"]; occ != "" {
			fmt.Sscanf(occ, "%d", &result.Occurrences)
		}

		results = append(results, result)

		// If this edit failed (not replaced), stop and return the error
		if !result.Replaced {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("MultiEdit: edit %d failed: %s", i+1, output.Output)))
			return &MultiEditOutput{
				Title:   displayPath,
				Output:  fmt.Sprintf("Edit %d failed: %s", i+1, output.Output),
				Results: results,
				Metadata: map[string]string{
					"error":       result.Error,
					"failed_at":   fmt.Sprintf("%d", i+1),
					"total_edits": fmt.Sprintf("%d", len(in.Edits)),
				},
			}, nil
		}
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("MultiEdit: all %d edits applied successfully", len(in.Edits))))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("MultiEdit: done for '%s'", displayPath), "multiedit", displayPath))

	// Return the final output with all results
	metadata := map[string]string{
		"error":       "",
		"total_edits": fmt.Sprintf("%d", len(in.Edits)),
	}

	// Include diff information from the last edit if available
	if lastOutput != nil && lastOutput.Metadata != nil {
		if diff := lastOutput.Metadata["diff"]; diff != "" {
			metadata["diff"] = diff
		}
		if additions := lastOutput.Metadata["additions"]; additions != "" {
			metadata["additions"] = additions
		}
		if deletions := lastOutput.Metadata["deletions"]; deletions != "" {
			metadata["deletions"] = deletions
		}
	}

	return &MultiEditOutput{
		Title:    displayPath,
		Output:   fmt.Sprintf("MultiEdit success: applied %d edits", len(in.Edits)),
		Results:  results,
		Metadata: metadata,
	}, nil
}
