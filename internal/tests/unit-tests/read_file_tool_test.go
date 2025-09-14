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

func TestReadFile_NilInput(t *testing.T) {
	_, err := tools.ReadFile(context.Background(), nil)
	utils.Equal(t, err.Error(), "input is required")
}

func TestReadFile_EmptyFilePath(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ReadFileInput{
		FilePath: "",
	}
	_, err := tools.ReadFile(context.Background(), input)
	utils.Equal(t, err.Error(), "file path is required")
}

func TestReadFile_ProjectRootNotSet(t *testing.T) {
	// Reset the base root
	tools.SetListDirectoryBaseRoot("")

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
	}
	_, err := tools.ReadFile(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "list directory base root not set"), true)
}

func TestReadFile_PathEscapesBase(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ReadFileInput{
		FilePath: "../../../etc/passwd",
	}
	_, err := tools.ReadFile(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "path escapes the configured base root"), true)
}

func TestReadFile_PathEscapesBaseRelative(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ReadFileInput{
		FilePath: "../outside.txt",
	}
	_, err := tools.ReadFile(context.Background(), input)
	utils.Equal(t, err.Error(), "path escapes the configured base root")
}

func TestReadFile_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ReadFileInput{
		FilePath: "nonexistent.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "file_not_found")
	utils.Equal(t, strings.Contains(output.Output, "File not found:"), true)
}

func TestReadFile_FileNotFoundWithSuggestions(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create some files where the search string is a substring
	err := os.WriteFile(filepath.Join(tempDir, "mytest.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "testfile.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "other.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt", // "test" is contained in "mytest" and "testfile"
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "file_not_found")
	utils.Equal(t, strings.Contains(output.Output, "Did you mean one of these?"), true)
	utils.Equal(t, strings.Contains(output.Output, "mytest.txt"), true)
	utils.Equal(t, strings.Contains(output.Output, "testfile.txt"), true)
}

func TestReadFile_PathIsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "subdir",
	}
	_, err = tools.ReadFile(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "path is a directory"), true)
}

func TestReadFile_ImageFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a fake image file
	imageFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(imageFile, []byte("fake image content"), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.jpg",
	}
	_, err = tools.ReadFile(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "this is an image file of type: JPEG"), true)
}

func TestReadFile_BinaryFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a binary file (zip extension)
	binaryFile := filepath.Join(tempDir, "test.zip")
	err := os.WriteFile(binaryFile, []byte{0x50, 0x4B, 0x03, 0x04}, 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.zip",
	}
	_, err = tools.ReadFile(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "cannot read binary file"), true)
}

func TestReadFile_BinaryFileByContent(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file with binary content (null bytes)
	binaryFile := filepath.Join(tempDir, "binary.txt")
	err := os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "binary.txt",
	}
	_, err = tools.ReadFile(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "cannot read binary file"), true)
}

func TestReadFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, strings.Contains(output.Output, "<file>"), true)
	utils.Equal(t, strings.Contains(output.Output, "</file>"), true)
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), true)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), true)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), true)
	utils.Equal(t, strings.Contains(output.Output, "00004| line 4"), true)
	utils.Equal(t, strings.Contains(output.Output, "00005| line 5"), true)
}

func TestReadFile_WithOffset(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
		Offset:   2, // Start from line 3 (0-based)
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), true)
	utils.Equal(t, strings.Contains(output.Output, "00004| line 4"), true)
	utils.Equal(t, strings.Contains(output.Output, "00005| line 5"), true)
	// Should not contain lines 1 and 2
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), false)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), false)
}

func TestReadFile_WithLimit(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
		Limit:    2, // Read only 2 lines
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), true)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), true)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), false)
	utils.Equal(t, strings.Contains(output.Output, "00004| line 4"), false)
	utils.Equal(t, strings.Contains(output.Output, "00005| line 5"), false)
	utils.Equal(t, strings.Contains(output.Output, "File has more lines"), true)
}

func TestReadFile_WithOffsetAndLimit(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
		Offset:   1, // Start from line 2 (0-based)
		Limit:    3, // Read 3 lines
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), true)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), true)
	utils.Equal(t, strings.Contains(output.Output, "00004| line 4"), true)
	utils.Equal(t, strings.Contains(output.Output, "00005| line 5"), false)
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), false)
	utils.Equal(t, strings.Contains(output.Output, "File has more lines"), true)
}

func TestReadFile_OffsetBeyondFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
		Offset:   10, // Beyond file length
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), false)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), false)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), false)
}

func TestReadFile_NegativeOffset(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
		Offset:   -5, // Negative offset should be treated as 0
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), true)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), true)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), true)
}

func TestReadFile_LongLineTruncation(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	// Create a line longer than maxLineLength (2000)
	longLine := strings.Repeat("a", 2500)
	content := "short line\n" + longLine + "\nanother line"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "short line"), true)
	utils.Equal(t, strings.Contains(output.Output, "another line"), true)
	utils.Equal(t, strings.Contains(output.Output, "..."), true)
}

func TestReadFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "empty.txt")
	err := os.WriteFile(testFile, []byte(""), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "empty.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "<file>"), true)
	utils.Equal(t, strings.Contains(output.Output, "</file>"), true)
}

func TestReadFile_SingleLineFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "single.txt")
	err := os.WriteFile(testFile, []byte("single line content"), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "single.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00001| single line content"), true)
}

func TestReadFile_Metadata(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\nline 11\nline 12\nline 13\nline 14\nline 15\nline 16\nline 17\nline 18\nline 19\nline 20\nline 21\nline 22"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "test.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["filepath"], filepath.ToSlash(testFile))
	utils.Equal(t, strings.Contains(output.Metadata["preview"], "line 1"), true)
	utils.Equal(t, strings.Contains(output.Metadata["preview"], "line 20"), true)
	utils.Equal(t, strings.Contains(output.Metadata["preview"], "line 21"), false) // Preview should only include first 20 lines
}

// Additional edge case tests
func TestReadFile_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "large.txt")
	// Create a file with more than defaultReadLimit (2000) lines
	var content strings.Builder
	for i := 1; i <= 2500; i++ {
		content.WriteString(fmt.Sprintf("This is line number %d\n", i))
	}
	err := os.WriteFile(testFile, []byte(content.String()), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "large.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "File has more lines"), true)
	utils.Equal(t, strings.Contains(output.Output, "002001| This is line number 2001"), false) // Should not include beyond limit
}

func TestReadFile_CRLFLineEndings(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "crlf.txt")
	content := "line 1\r\nline 2\r\nline 3\r\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.ReadFileInput{
		FilePath: "crlf.txt",
	}
	output, err := tools.ReadFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(output.Output, "00001| line 1"), true)
	utils.Equal(t, strings.Contains(output.Output, "00002| line 2"), true)
	utils.Equal(t, strings.Contains(output.Output, "00003| line 3"), true)
}
