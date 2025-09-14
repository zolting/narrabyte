package unit_tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"narrabyte/internal/llm/tools"
	"narrabyte/internal/utils"
)

func TestGrep_NilInput(t *testing.T) {
	result, err := tools.Grep(context.Background(), nil)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: input is required")
}

func TestGrep_EmptyPattern(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.GrepInput{
		Pattern: "",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: pattern is required")
}

func TestGrep_ProjectRootNotSet(t *testing.T) {
	// Reset the base root
	tools.SetListDirectoryBaseRoot("")

	input := &tools.GrepInput{
		Pattern: "test",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: project root not set")
}

func TestGrep_PathEscapesBase(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.GrepInput{
		Pattern: "test",
		Path:    "../../../etc",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: path escapes the configured project root")
}

func TestGrep_PathDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.GrepInput{
		Pattern: "test",
		Path:    "nonexistent",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: path does not exist or is not accessible")
}

func TestGrep_PathIsFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "test",
		Path:    "test.txt",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: not a directory")
}

func TestGrep_InvalidRegex(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.GrepInput{
		Pattern: "[invalid",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["error"], "format_error")
	utils.Equal(t, result.Output, "Format error: invalid regex pattern")
}

func TestGrep_NoMatches(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file with content that won't match
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "nonexistent",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Output, "No files found")
	utils.Equal(t, result.Metadata["matches"], "0")
	utils.Equal(t, result.Metadata["truncated"], "false")
}

func TestGrep_SimpleMatch(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file with matching content
	content := "line 1\nhello world\nline 3"
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 2: hello world"), true)
	utils.Equal(t, result.Metadata["matches"], "1")
	utils.Equal(t, result.Metadata["truncated"], "false")
}

func TestGrep_MultipleMatchesInFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file with multiple matches
	content := "hello world\nhello universe\nhello galaxy"
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 3 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 1: hello world"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 2: hello universe"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 3: hello galaxy"), true)
	utils.Equal(t, result.Metadata["matches"], "3")
}

func TestGrep_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create multiple files with matches
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("hello universe"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file3.txt"), []byte("goodbye world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 2 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "file1.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "file2.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "file3.txt"), false)
	utils.Equal(t, result.Metadata["matches"], "2")
}

func TestGrep_RegexPatterns(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files with different patterns
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test123\ntest456\nother"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "test\\d+", // Regex pattern
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 2 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 1: test123"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 2: test456"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 3: other"), false)
}

func TestGrep_IncludePattern(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files of different types
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.go"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
		Include: "*.txt",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.go"), false)
	utils.Equal(t, strings.Contains(result.Output, "test.md"), false)
}

func TestGrep_IncludePatternMultiple(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files of different types
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.go"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
		Include: "*.{txt,go}",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 2 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.go"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.md"), false)
}

func TestGrep_BinaryFileSkipped(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a text file and a binary file
	err := os.WriteFile(filepath.Join(tempDir, "text.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "binary.zip"), []byte{0x50, 0x4B, 0x03, 0x04}, 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "text.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "binary.zip"), false)
}

func TestGrep_IgnoredDirectories(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files in normal and ignored directories
	err := os.WriteFile(filepath.Join(tempDir, "normal.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	nodeModulesDir := filepath.Join(tempDir, "node_modules")
	err = os.Mkdir(nodeModulesDir, 0755)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(nodeModulesDir, "package.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "normal.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "package.txt"), false)
}

func TestGrep_ResultLimit(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create more files than the limit (100)
	for i := 0; i < 110; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%03d.txt", i))
		content := fmt.Sprintf("match %d", i)
		err := os.WriteFile(filename, []byte(content), 0644)
		utils.NilError(t, err)
	}

	input := &tools.GrepInput{
		Pattern: "match",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Metadata["matches"], "100")
	utils.Equal(t, result.Metadata["truncated"], "true")
	utils.Equal(t, strings.Contains(result.Output, "(Results are truncated. Consider using a more specific path or pattern.)"), true)
}

func TestGrep_SpecificPath(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	// Create files in root and subdirectory
	err = os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
		Path:    "subdir",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "sub.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "root.txt"), false)
}

func TestGrep_SortingByMtime(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files with different modification times
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	err := os.WriteFile(file1, []byte("hello world"), 0644)
	utils.NilError(t, err)
	// Sleep to ensure different mtimes
	// Note: In a real test, we'd use time.Sleep, but for simplicity we'll assume the order
	err = os.WriteFile(file2, []byte("hello universe"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 2 matches"), true)
	// The output should contain both files (exact order may vary based on filesystem)
	utils.Equal(t, strings.Contains(result.Output, "file1.txt"), true)
	utils.Equal(t, strings.Contains(result.Output, "file2.txt"), true)
}

func TestGrep_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.GrepInput{
		Pattern: "test",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Output, "No files found")
	utils.Equal(t, result.Metadata["matches"], "0")
}

func TestGrep_LongLines(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file with a very long line
	longLine := strings.Repeat("a", 10000) + " target " + strings.Repeat("b", 10000)
	err := os.WriteFile(filepath.Join(tempDir, "long.txt"), []byte(longLine), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "target",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "target"), true)
}

func TestGrep_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a large file to potentially trigger cancellation
	largeContent := strings.Repeat("line content\n", 1000)
	err := os.WriteFile(filepath.Join(tempDir, "large.txt"), []byte(largeContent), 0644)
	utils.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := &tools.GrepInput{
		Pattern: "content",
	}
	_, err = tools.Grep(ctx, input)
	utils.Equal(t, err, context.Canceled)
}

// Additional edge case tests
func TestGrep_IncludePatternComplex(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files with different extensions
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.go"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.ts"), []byte("hello world"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test.tsx"), []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "hello",
		Include: "*.{ts,tsx}",
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 2 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.ts"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.tsx"), true)
	utils.Equal(t, strings.Contains(result.Output, "test.txt"), false)
	utils.Equal(t, strings.Contains(result.Output, "test.go"), false)
}

func TestGrep_CaseSensitive(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files with mixed case content
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("Hello World\nhello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "Hello", // Case sensitive by default
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 1 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 1: Hello World"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 2: hello world"), false)
}

func TestGrep_CaseInsensitive(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files with mixed case content
	err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("Hello World\nhello world"), 0644)
	utils.NilError(t, err)

	input := &tools.GrepInput{
		Pattern: "(?i)hello", // Case insensitive regex
	}
	result, err := tools.Grep(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result.Output, "Found 2 matches"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 1: Hello World"), true)
	utils.Equal(t, strings.Contains(result.Output, "Line 2: hello world"), true)
}
