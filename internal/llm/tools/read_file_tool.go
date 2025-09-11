package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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
	// FileRelativePath is the path to the file relative to the project root.
	// Absolute paths are allowed only if they resolve under the configured base root.
	FileRelativePath string `json:"file_relative_path" jsonschema:"description=The path to the file to read (relative to project root)"`
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
func ReadFile(_ context.Context, input *ReadFileInput) (*ReadFileOutput, error) {
	if in == nil {
		return nil, errors.New("input is required")
	}

	base, err := getListDirectoryBaseRoot()
	if err != nil {
		return nil, err
	}
	pathArg := strings.TrimSpace(in.FileRelativePath)
	if pathArg == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Resolve target path under base, ensuring it cannot escape.
	var absPath string
	if filepath.IsAbs(pathArg) {
		// Ensure absolute path is under base
		absBase, err := filepath.Abs(base)
		if err != nil {
			return nil, err
		}
		absCandidate, err := filepath.Abs(pathArg)
		if err != nil {
			return nil, err
		}
		relToBase, err := filepath.Rel(absBase, absCandidate)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			return nil, fmt.Errorf("file %s is not in the configured project root", pathArg)
		}
		absPath = absCandidate
	} else {
		abs, ok := safeJoinUnderBase(base, pathArg)
		if !ok {
			return nil, fmt.Errorf("path escapes the configured base root")
		}
		absPath = abs
	}

	// Ensure file exists
	file, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Build suggestions from sibling entries
			dir := filepath.Dir(absPath)
			baseName := filepath.Base(absPath)
			suggestions := similarEntries(dir, baseName)

			// Title should be relative to base root
			rel, relErr := filepath.Rel(base, absPath)
			if relErr != nil {
				rel = absPath // fallback
			}
			rel = filepath.ToSlash(rel)

			output := "<file>\nFile not found: " + absPath + "\n"
			if len(suggestions) > 0 {
				output += "\nDid you mean one of these?\n" + strings.Join(suggestions, "\n") + "\n"
			}
			output += "\n</file>"

			return &ReadFileOutput{
				Title:  rel,
				Output: output,
				Metadata: map[string]string{
					"error": "file_not_found",
				},
			}, nil
		}
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", absPath)
	}

	// Image file check
	if img := imageTypeByExt(absPath); img != "" {
		return nil, fmt.Errorf("this is an image file of type: %s\nUse a different tool to process images", img)
	}

	// Binary file check
	isBin, err := isBinaryFile(absPath)
	if err != nil {
		return nil, err
	}
	if isBin {
		return nil, fmt.Errorf("cannot read binary file: %s", absPath)
	}

	// Read all text and split into lines
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	// Normalize to LF for counting; we split on '\n'
	text := string(data)
	lines := strings.Split(text, "\n")

	// Bounds and defaults
	limit := in.Limit
	if limit <= 0 {
		limit = defaultReadLimit
	}
	offset := in.Offset
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

	raw := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		line := lines[i]
		if len(line) > maxLineLength {
			line = line[:maxLineLength] + "..."
		}
		raw = append(raw, line)
	}

	// Build numbered content
	var b strings.Builder
	b.WriteString("<file>\n")
	for i, line := range raw {
		// Line numbers are 1-based, padded to 5 digits
		ln := fmt.Sprintf("%05d| ", i+offset+1)
		b.WriteString(ln)
		b.WriteString(line)
		if i < len(raw)-1 {
			b.WriteByte('\n')
		}
	}

	if len(lines) > offset+len(raw) {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		} else {
			// Ensure a leading newline before the note in empty slice
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("(File has more lines. Use 'offset' parameter to read beyond line %d)", offset+len(raw)))
	}
	b.WriteString("\n</file>")

	// Compute preview (first 20 raw lines)
	previewCount := 20
	if previewCount > len(raw) {
		previewCount = len(raw)
	}
	preview := strings.Join(raw[:previewCount], "\n")

	// Title should be relative to base root
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		rel = absPath // fallback
	}
	rel = filepath.ToSlash(rel)

	println(b.String())

	return &ReadFileOutput{
		Title:  rel,
		Output: b.String(),
		Metadata: map[string]string{
			"preview": preview,
		},
	}, nil
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
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	if fi.Size() == 0 {
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
	needle := strings.ToLower(baseName)
	var candidates []string
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.Contains(lower, needle) || strings.Contains(needle, lower) {
			candidates = append(candidates, filepath.Join(dir, name))
		}
	}
	sort.Strings(candidates)
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	// Normalize to platform path format
	return candidates
}
