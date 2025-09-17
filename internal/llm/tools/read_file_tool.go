package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"narrabyte/internal/events"
	"os"
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
	// FilePath is the absolute path to the file to read.
	FilePath string `json:"file_path" jsonschema:"description=The absolute path to the file to read"`
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

// ReadFile reads a text file within the project root with paging and safety checks.
func ReadFile(ctx context.Context, input *ReadFileInput) (out *ReadFileOutput, err error) {
	// Start
	runtime.EventsEmit(ctx, events.EventToolStart, events.NewInfo("ReadFile: starting"))

	// Emit error/done for every return path
	defer func() {
		if err != nil {
			runtime.EventsEmit(ctx, events.EventToolError, events.NewError(fmt.Sprintf("ReadFile: %v", err)))
			return
		}
		title := ""
		if out != nil {
			title = strings.TrimSpace(out.Title)
		}
		if title == "" {
			title = "success"
		}
		runtime.EventsEmit(ctx, events.EventToolDone, events.NewInfo(fmt.Sprintf("ReadFile: done (%s)", title)))
	}()

	if input == nil {
		err = errors.New("input is required")
		return nil, err
	}

	base, err := getListDirectoryBaseRoot()
	if err != nil {
		// defer will emit error
		return nil, err
	}
	pathArg := strings.TrimSpace(input.FilePath)
	if pathArg == "" {
		err = fmt.Errorf("file path is required")
		return nil, err
	}

	// Resolve target path under base, ensuring it cannot escape base.
	var absPath string
	if filepath.IsAbs(pathArg) {
		// Ensure absolute path is under base
		absBase, e := filepath.Abs(base)
		if e != nil {
			err = e
			return nil, err
		}
		absCandidate, e := filepath.Abs(pathArg)
		if e != nil {
			err = e
			return nil, err
		}
		relToBase, e := filepath.Rel(absBase, absCandidate)
		if e != nil {
			err = e
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			err = fmt.Errorf("file %s is not in the configured project root", pathArg)
			return nil, err
		}
		absPath = absCandidate
	} else {
		abs, ok := safeJoinUnderBase(base, pathArg)
		if !ok {
			err = fmt.Errorf("path escapes the configured base root")
			return nil, err
		}
		absPath = abs
	}

	// Progress: resolved path
	runtime.EventsEmit(ctx, events.EventToolProgress, events.NewInfo(fmt.Sprintf("ReadFile: reading '%s'", filepath.ToSlash(absPath))))

	// Ensure file exists
	fileInfo, statErr := os.Stat(absPath)
	if statErr != nil {
		err = statErr
		return nil, err
	}
	if fileInfo.IsDir() {
		err = fmt.Errorf("path is a directory: %s", filepath.ToSlash(absPath))
		return nil, err
	}

	// Image file check
	if img := imageTypeByExt(absPath); img != "" {
		// Treat as non-fatal informational output
		out = &ReadFileOutput{
			Title:  filepath.ToSlash(absPath),
			Output: fmt.Sprintf("Binary image detected (%s). Reading skipped.", img),
			Metadata: map[string]string{
				"error": "unsupported_image",
				"type":  img,
			},
		}
		// Optional progress notice
		runtime.EventsEmit(ctx, events.EventToolProgress, events.NewWarn(fmt.Sprintf("ReadFile: unsupported image '%s' (%s)", filepath.ToSlash(absPath), img)))
		return out, nil
	}

	// Binary file check
	isBin, binErr := isBinaryFile(absPath)
	if binErr != nil {
		err = binErr
		return nil, err
	}
	if isBin {
		out = &ReadFileOutput{
			Title:  filepath.ToSlash(absPath),
			Output: "Binary file detected. Reading skipped.",
			Metadata: map[string]string{
				"error": "unsupported_binary",
			},
		}
		runtime.EventsEmit(ctx, events.EventToolProgress, events.NewWarn(fmt.Sprintf("ReadFile: unsupported binary '%s'", filepath.ToSlash(absPath))))
		return out, nil
	}

	// Read all text and split into lines
	data, readErr := os.ReadFile(absPath)
	if readErr != nil {
		err = readErr
		return nil, err
	}
	text := string(data)
	lines := strings.Split(text, "\n")

	// Bounds and defaults
	limit := input.Limit
	if limit <= 0 {
		limit = defaultReadLimit
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	start := offset
	if start > len(lines) {
		start = len(lines)
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}

	// Prepare output raw lines with truncation of very long lines
	raw := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		line := lines[i]
		if len(line) > maxLineLength {
			line = line[:maxLineLength] + " …(truncated)…"
		}
		raw = append(raw, line)
	}

	// Build numbered content
	var b strings.Builder
	b.WriteString("<file>\n")
	for i, line := range raw {
		// 1-based line numbering in the excerpt
		b.WriteString(fmt.Sprintf("%6d: %s\n", start+i+1, line))
	}
	if len(lines) > offset+len(raw) {
		b.WriteString("... (truncated)\n")
	}
	b.WriteString("</file>")

	// Compute preview (first 20 raw lines)
	previewCount := 20
	if previewCount > len(raw) {
		previewCount = len(raw)
	}
	preview := strings.Join(raw[:previewCount], "\n")

	out = &ReadFileOutput{
		Title:  filepath.ToSlash(absPath),
		Output: b.String(),
		Metadata: map[string]string{
			"offset":  fmt.Sprintf("%d", offset),
			"limit":   fmt.Sprintf("%d", limit),
			"preview": preview,
		},
	}

	runtime.EventsEmit(ctx, events.EventToolProgress, events.NewInfo(fmt.Sprintf("ReadFile: read %d/%d lines from '%s'", len(raw), len(lines), filepath.ToSlash(absPath))))
	return out, nil
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
