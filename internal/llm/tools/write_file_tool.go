package tools

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"narrabyte/internal/events"
	"os"
	"path/filepath"
	"strings"
)

type WriteFileInput struct {
	// FilePath is the absolute path to the file to write.
	FilePath string `json:"file_path" jsonschema:"description=The absolute path to the file to write"`
	// Content is the content to write to the file.
	Content string `json:"content" jsonschema:"description=The content to write to the file"`
}

type WriteFileOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WriteFile writes content to a file under the configured project root. Creates the file if it does not exist.
// It does not create parent directories; the target directory must already exist.
func WriteFile(ctx context.Context, in *WriteFileInput) (*WriteFileOutput, error) {
	runtime.EventsEmit(ctx, events.EventToolStart, events.NewInfo("WriteFile: starting"))

	if in == nil {
		runtime.EventsEmit(ctx, events.EventToolError, events.NewError("WriteFile: input is required"))
		return &WriteFileOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	base, err := getListDirectoryBaseRoot()
	if err != nil {
		runtime.EventsEmit(ctx, events.EventToolError, events.NewError("WriteFile: project root not set"))
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
		runtime.EventsEmit(ctx, events.EventToolError, events.NewError("WriteFile: file_path is required"))
		return &WriteFileOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Resolve target path under base, ensuring it cannot escape.
	var absPath string
	if filepath.IsAbs(p) {
		absBase, err := filepath.Abs(base)
		if err != nil {
			runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: base resolve error: %v", err)))
			return nil, err
		}
		absCandidate, err := filepath.Abs(p)
		if err != nil {
			runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: abs path error: %v", err)))
			return nil, err
		}
		relToBase, err := filepath.Rel(absBase, absCandidate)
		if err != nil {
			runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: rel error: %v", err)))
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			runtime.EventsEmit(ctx, events.EventToolError, events.NewWarn("WriteFile: path escapes the configured project root"))
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
			runtime.EventsEmit(ctx, events.EventToolError, events.NewWarn("WriteFile: path escapes the configured project root"))
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

	runtime.EventsEmit(ctx, events.EventToolProgress, events.NewInfo(fmt.Sprintf("WriteFile: writing '%s'", filepath.ToSlash(absPath))))

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: directory does not exist: %s", filepath.ToSlash(dir))))
			return &WriteFileOutput{
				Title:  filepath.ToSlash(absPath),
				Output: fmt.Sprintf("Format error: directory does not exist: %s", dir),
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}
		runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: stat error: %v", err)))
		return nil, err
	}
	if !info.IsDir() {
		runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: not a directory: %s", filepath.ToSlash(dir))))
		return &WriteFileOutput{
			Title:  filepath.ToSlash(absPath),
			Output: fmt.Sprintf("Format error: not a directory: %s", dir),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Determine if file existed
	existed := false
	if st, err := os.Stat(absPath); err == nil && !st.IsDir() {
		existed = true
	}

	// Write file (creates or truncates)
	if err := os.WriteFile(absPath, []byte(in.Content), 0o644); err != nil {
		runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("WriteFile: write error: %v", err)))
		return nil, err
	}

	outputMsg := ""
	if existed {
		outputMsg = fmt.Sprintf("Overwrote file: %s", filepath.ToSlash(absPath))
	} else {
		outputMsg = fmt.Sprintf("Created file: %s", filepath.ToSlash(absPath))
	}
	runtime.EventsEmit(ctx, events.EventToolProgress, events.NewInfo(outputMsg))
	runtime.EventsEmit(ctx, events.EventToolDone, events.NewInfo(fmt.Sprintf("WriteFile: done for '%s'", filepath.ToSlash(absPath))))

	return &WriteFileOutput{
		Title:  filepath.ToSlash(absPath),
		Output: outputMsg,
		Metadata: map[string]string{
			"filepath": filepath.ToSlash(absPath),
			"exists":   fmt.Sprintf("%v", existed),
		},
	}, nil
}
