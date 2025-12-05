package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"narrabyte/internal/events"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultReadLimit = 2000
	maxLineLength    = 2000
)

// ReadFileInput defines the parameters for the read file tool.
type ReadFileInput struct {
	// Repository specifies which repository the path is relative to.
	Repository Repository `json:"repository" jsonschema:"enum=docs,enum=code,description=Which repository the path is relative to: 'docs' for documentation repository or 'code' for the codebase repository"`
	// FilePath is the relative path to the file within the specified repository.
	FilePath string `json:"file_path" jsonschema:"description=The path to the file relative to the repository root (e.g. 'src/main.go' or 'api/endpoints.md'). NEVER use absolute paths."`
	// Offset is the 0-based line number to start reading from.
	Offset int `json:"offset,omitempty" jsonschema:"description=The line number to start reading from (0-based)"`
	// Limit is the number of lines to read.
	Limit int `json:"limit,omitempty" jsonschema:"description=The number of lines to read (defaults to 2000)"`
}

// ReadFileOutput mirrors the TS tool return shape for downstream consumers.
type ReadFileOutput struct {
	Title    string            `json:"title"`
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ReadFile reads a text file within the specified repository with paging and safety checks.
func ReadFile(ctx context.Context, input *ReadFileInput) (*ReadFileOutput, error) {
	// Start
	snapshot := currentGitSnapshot(ctx)
	snapshotInfo := formatSnapshotInfo(snapshot)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ReadFile: starting [%s]", snapshotInfo)))

	if input == nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError("ReadFile: input is required"))
		return &ReadFileOutput{
			Title:  "",
			Output: "Format error: input is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Validate repository
	if !input.Repository.IsValid() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: invalid repository '%s'", input.Repository)))
		return &ReadFileOutput{
			Title:  "",
			Output: fmt.Sprintf("Format error: invalid repository '%s'; must be 'docs' or 'code'", input.Repository),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	pathArg := strings.TrimSpace(input.FilePath)
	if pathArg == "" {
		events.Emit(ctx, events.LLMEventTool, events.NewError("ReadFile: file_path is required"))
		return &ReadFileOutput{
			Title:  "",
			Output: "Format error: file_path is required",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Resolve path using the repository-scoped resolver
	absPath, err := ResolveRepositoryPath(ctx, input.Repository, pathArg)
	if err != nil {
		displayPath := FormatDisplayPath(input.Repository, pathArg)
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: %v", err)))
		return &ReadFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: %v", err),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Progress: resolved path
	displayPath := FormatDisplayPath(input.Repository, pathArg)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ReadFile: reading '%s' [%s]", displayPath, snapshotInfo)))

	// Use git snapshot only for code repository reads when a snapshot is configured
	if input.Repository == RepositoryCode {
		snapshot := currentGitSnapshot(ctx)
		if snapshot != nil {
			rel, relErr := snapshot.relativeFromAbs(absPath)
			if relErr != nil {
				if errors.Is(relErr, ErrSnapshotEscapes) {
					events.Emit(ctx, events.LLMEventTool, events.NewWarn("ReadFile: path escapes git snapshot root"))
					return &ReadFileOutput{
						Title:    displayPath,
						Output:   "Format error: path escapes the configured project root",
						Metadata: map[string]string{"error": "format_error"},
					}, nil
				}
				events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: snapshot rel path error: %v", relErr)))
				return &ReadFileOutput{
					Title:    displayPath,
					Output:   "Format error: failed to resolve path within repository snapshot",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}

			data, isBinary, readErr := snapshot.readFile(rel)
			if readErr != nil {
				switch {
				case errors.Is(readErr, ErrSnapshotDirectory):
					events.Emit(ctx, events.LLMEventTool, events.NewError("ReadFile: snapshot path is a directory"))
					return &ReadFileOutput{
						Title:    displayPath,
						Output:   fmt.Sprintf("Format error: path is a directory: %s", displayPath),
						Metadata: map[string]string{"error": "format_error"},
					}, nil
				case errors.Is(readErr, ErrSnapshotNotFound):
					events.Emit(ctx, events.LLMEventTool, events.NewWarn("ReadFile: snapshot file not found"))
					dirRel := path.Dir(rel)
					if dirRel == "." {
						dirRel = ""
					}
					suggestions := snapshot.suggestions(dirRel, path.Base(rel), 3)
					output := fmt.Sprintf("Format error: file does not exist in the repository snapshot: %s", displayPath)
					if len(suggestions) > 0 {
						output += fmt.Sprintf("\n\nDid you mean one of these files?\n- %s", strings.Join(suggestions, "\n- "))
					}
					return &ReadFileOutput{
						Title:    displayPath,
						Output:   output,
						Metadata: map[string]string{"error": "format_error"},
					}, nil
				default:
					events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: snapshot read error: %v", readErr)))
					return nil, readErr
				}
			}

			if img := imageTypeByExt(absPath); img != "" {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("ReadFile: unsupported image '%s' (%s) in snapshot", displayPath, img)))
				return &ReadFileOutput{
					Title:  displayPath,
					Output: fmt.Sprintf("Binary image detected (%s). Reading skipped.", img),
					Metadata: map[string]string{
						"error": "unsupported_image",
						"type":  img,
					},
				}, nil
			}
			if isBinary {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("ReadFile: unsupported binary '%s' in snapshot", displayPath)))
				return &ReadFileOutput{
					Title:  displayPath,
					Output: "Binary file detected. Reading skipped.",
					Metadata: map[string]string{
						"error": "unsupported_binary",
					},
				}, nil
			}

			lines := strings.Split(string(data), "\n")
			out, readCount, totalLines := BuildReadFileOutput(displayPath, lines, input.Offset, input.Limit)
			events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ReadFile: read %d/%d lines from '%s' [%s]", readCount, totalLines, displayPath, snapshotInfo)))
			events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("ReadFile: done (%s) [%s]", displayPath, snapshotInfo), "read", displayPath))
			return out, nil
		}
	}

	// Read from filesystem (for docs repository, or code when no snapshot)
	fileInfo, statErr := os.Stat(absPath)
	if statErr != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: stat error: %v", statErr)))

		// Get similar file suggestions
		dir := filepath.Dir(absPath)
		baseName := filepath.Base(absPath)
		suggestions := similarEntries(dir, baseName)

		output := fmt.Sprintf("Format error: file does not exist or is not accessible: %s", displayPath)
		if len(suggestions) > 0 {
			output += fmt.Sprintf("\n\nDid you mean one of these files?\n- %s", strings.Join(suggestions, "\n- "))
		}

		return &ReadFileOutput{
			Title:  displayPath,
			Output: output,
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}
	if fileInfo.IsDir() {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: path is a directory: %s", displayPath)))
		return &ReadFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: path is a directory: %s", displayPath),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}

	// Image file check
	if img := imageTypeByExt(absPath); img != "" {
		// Treat as non-fatal informational output
		events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("ReadFile: unsupported image '%s' (%s)", displayPath, img)))
		return &ReadFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Binary image detected (%s). Reading skipped.", img),
			Metadata: map[string]string{
				"error": "unsupported_image",
				"type":  img,
			},
		}, nil
	}

	// Binary file check
	isBin, binErr := isBinaryFile(absPath)
	if binErr != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: binary check error: %v", binErr)))
		return &ReadFileOutput{
			Title:  displayPath,
			Output: "Format error: failed to check if file is binary",
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}
	if isBin {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("ReadFile: unsupported binary '%s'", displayPath)))
		return &ReadFileOutput{
			Title:  displayPath,
			Output: "Binary file detected. Reading skipped.",
			Metadata: map[string]string{
				"error": "unsupported_binary",
			},
		}, nil
	}

	// Read all text and split into lines
	data, readErr := os.ReadFile(absPath)
	if readErr != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile: read error: %v", readErr)))
		return &ReadFileOutput{
			Title:  displayPath,
			Output: fmt.Sprintf("Format error: failed to read file: %v", readErr),
			Metadata: map[string]string{
				"error": "format_error",
			},
		}, nil
	}
	lines := strings.Split(string(data), "\n")
	out, readCount, totalLines := BuildReadFileOutput(displayPath, lines, input.Offset, input.Limit)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ReadFile: read %d/%d lines from '%s' [%s]", readCount, totalLines, displayPath, snapshotInfo)))
	events.Emit(ctx, events.LLMEventTool, events.NewToolEvent(events.EventInfo, fmt.Sprintf("ReadFile: done (%s) [%s]", displayPath, snapshotInfo), "read", displayPath))
	return out, nil
}

// BuildReadFileOutput formats the provided lines into the standard numbered payload
// returned by the read file tool. It applies offset/limit paging, truncates long
// lines, and produces metadata (offset, limit, preview). The function returns the
// constructed output along with the number of lines emitted and the total number
// of lines available.
func BuildReadFileOutput(title string, lines []string, offset, limit int) (*ReadFileOutput, int, int) {
	limitNormalized := limit
	if limitNormalized <= 0 {
		limitNormalized = defaultReadLimit
	}
	offsetNormalized := offset
	if offsetNormalized < 0 {
		offsetNormalized = 0
	}
	start := offsetNormalized
	if start > len(lines) {
		start = len(lines)
	}
	end := start + limitNormalized
	if end > len(lines) {
		end = len(lines)
	}

	raw := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		line := lines[i]
		if len(line) > maxLineLength {
			line = line[:maxLineLength] + " …(truncated)…"
		}
		raw = append(raw, line)
	}

	var b strings.Builder
	b.WriteString("<file>\n")
	for i, line := range raw {
		b.WriteString(fmt.Sprintf("%05d| %s\n", start+i+1, line))
	}
	if len(lines) > offsetNormalized+len(raw) {
		b.WriteString("File has more lines\n")
	}
	b.WriteString("</file>")

	previewCount := 20
	if previewCount > len(raw) {
		previewCount = len(raw)
	}
	preview := strings.Join(raw[:previewCount], "\n")

	meta := map[string]string{
		"filepath": title,
		"error":    "",
		"offset":   fmt.Sprintf("%d", offsetNormalized),
		"limit":    fmt.Sprintf("%d", limitNormalized),
		"preview":  preview,
	}

	return &ReadFileOutput{
		Title:    title,
		Output:   b.String(),
		Metadata: meta,
	}, len(raw), len(lines)
}

// imageTypeByExt returns a human-readable image type for common image extensions, else "".
func imageTypeByExt(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg":
		return "JPEG"
	case ".png":
		return "PNG"
	case ".gif":
		return "GIF"
	case ".bmp":
		return "BMP"
	case ".webp":
		return "WebP"
	default:
		return ""
	}
}

// isBinaryFile performs a quick extension check, then a heuristic byte-scan
// of up to the first 4096 bytes to decide if a file is binary.
func isBinaryFile(p string) (bool, error) {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".zip", ".tar", ".gz", ".exe", ".dll", ".so", ".class", ".jar", ".war",
		".7z", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".ods", ".odp",
		".bin", ".dat", ".obj", ".o", ".a", ".lib", ".wasm", ".pyc", ".pyo":
		return true, nil
	}

	f, err := os.Open(p)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Stat to determine size and clamp buffer
	file, err := f.Stat()
	if err != nil {
		return false, err
	}
	if file.Size() == 0 {
		return false, nil
	}

	// Read up to 4096 bytes
	const maxBuf = 4096
	r := bufio.NewReader(f)
	buf := make([]byte, 0, maxBuf)
	for len(buf) < maxBuf {
		chunk := maxBuf - len(buf)
		tmp := make([]byte, chunk)
		n, readErr := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return false, readErr
		}
	}
	if len(buf) == 0 {
		return false, nil
	}

	nonPrintable := 0
	for _, b := range buf {
		if b == 0x00 {
			return true, nil
		}
		if b < 9 || (b > 13 && b < 32) {
			nonPrintable++
		}
	}
	// If >30% non-printable characters, consider it binary
	return float64(nonPrintable)/float64(len(buf)) > 0.3, nil
}

// similarEntries returns up to 3 suggestions in the same directory based on substring matching.
func similarEntries(dir string, baseName string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	needle := strings.ToLower(strings.TrimSuffix(baseName, filepath.Ext(baseName)))
	var candidates []string
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
		// Check if needle is a substring of the file name or vice versa
		if strings.Contains(lower, needle) || strings.Contains(needle, lower) {
			candidates = append(candidates, name) // Just return the filename, not the full path
		}
	}
	sort.Strings(candidates)
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	return candidates
}
