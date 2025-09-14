package unit_tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"narrabyte/internal/llm/tools"
	"narrabyte/internal/utils"
)

func TestWriteFile_NilInput(t *testing.T) {
	result, err := tools.WriteFile(context.Background(), nil)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, "")
	utils.Equal(t, result.Output, "Format error: input is required")
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_EmptyFilePath(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.WriteFileInput{
		FilePath: "",
		Content:  "test content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, "")
	utils.Equal(t, result.Output, "Format error: file_path is required")
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_ProjectRootNotSet(t *testing.T) {
	// Clear the base root to simulate it not being set
	tools.SetListDirectoryBaseRoot("")

	input := &tools.WriteFileInput{
		FilePath: "test.txt",
		Content:  "test content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, "")
	utils.Equal(t, result.Output, "Format error: project root not set")
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_PathEscapesBase(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.WriteFileInput{
		FilePath: "../../../etc/passwd",
		Content:  "test content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, filepath.ToSlash(filepath.Join(tempDir, "../../../etc/passwd")))
	utils.Equal(t, result.Output, "Format error: path escapes the configured project root")
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_AbsolutePathEscapesBase(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create an absolute path that escapes the base
	escapePath := filepath.Join(filepath.Dir(tempDir), "escape.txt")
	input := &tools.WriteFileInput{
		FilePath: escapePath,
		Content:  "test content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, filepath.ToSlash(escapePath))
	utils.Equal(t, result.Output, "Format error: file is not in the configured project root")
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_DirectoryDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.WriteFileInput{
		FilePath: "nonexistent/subdir/test.txt",
		Content:  "test content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, filepath.ToSlash(filepath.Join(tempDir, "nonexistent/subdir/test.txt")))
	utils.Equal(t, result.Output, "Format error: directory does not exist: "+filepath.Join(tempDir, "nonexistent", "subdir"))
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_PathIsNotDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file instead of directory
	filePath := filepath.Join(tempDir, "notdir")
	err := os.WriteFile(filePath, []byte("not a dir"), 0644)
	utils.NilError(t, err)

	input := &tools.WriteFileInput{
		FilePath: "notdir/test.txt",
		Content:  "test content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, result.Title, filepath.ToSlash(filepath.Join(tempDir, "notdir/test.txt")))
	utils.Equal(t, result.Output, "Format error: not a directory: "+filePath)
	utils.Equal(t, result.Metadata["error"], "format_error")
}

func TestWriteFile_CreateNewFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	targetPath := "test.txt"
	input := &tools.WriteFileInput{
		FilePath: targetPath,
		Content:  "Hello, World!",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	expectedPath := filepath.Join(tempDir, targetPath)
	utils.Equal(t, result.Title, filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Output, "Created file: "+filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Metadata["filepath"], filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Metadata["exists"], "false")

	// Verify file was created with correct content
	content, err := os.ReadFile(expectedPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "Hello, World!")

	// Verify file permissions
	info, err := os.Stat(expectedPath)
	utils.NilError(t, err)
	utils.Equal(t, info.Mode().Perm(), os.FileMode(0644))
}

func TestWriteFile_OverwriteExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	targetPath := "existing.txt"
	fullPath := filepath.Join(tempDir, targetPath)

	// Create existing file
	err := os.WriteFile(fullPath, []byte("original content"), 0644)
	utils.NilError(t, err)

	input := &tools.WriteFileInput{
		FilePath: targetPath,
		Content:  "new content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	utils.Equal(t, result.Title, filepath.ToSlash(fullPath))
	utils.Equal(t, result.Output, "Overwrote file: "+filepath.ToSlash(fullPath))
	utils.Equal(t, result.Metadata["filepath"], filepath.ToSlash(fullPath))
	utils.Equal(t, result.Metadata["exists"], "true")

	// Verify file was overwritten with correct content
	content, err := os.ReadFile(fullPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "new content")
}

func TestWriteFile_EmptyContent(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	targetPath := "empty.txt"
	input := &tools.WriteFileInput{
		FilePath: targetPath,
		Content:  "",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	expectedPath := filepath.Join(tempDir, targetPath)
	utils.Equal(t, result.Title, filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Output, "Created file: "+filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Metadata["filepath"], filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Metadata["exists"], "false")

	// Verify empty file was created
	content, err := os.ReadFile(expectedPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "")
}

func TestWriteFile_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	targetPath := "special.txt"
	input := &tools.WriteFileInput{
		FilePath: targetPath,
		Content:  "Line 1\nLine 2\tTab\nUnicode: ñáéíóú",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	expectedPath := filepath.Join(tempDir, targetPath)
	utils.Equal(t, result.Title, filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Output, "Created file: "+filepath.ToSlash(expectedPath))

	// Verify content with special characters
	content, err := os.ReadFile(expectedPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "Line 1\nLine 2\tTab\nUnicode: ñáéíóú")
}

func TestWriteFile_InSubdirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	targetPath := "subdir/nested.txt"
	input := &tools.WriteFileInput{
		FilePath: targetPath,
		Content:  "nested file content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	expectedPath := filepath.Join(tempDir, targetPath)
	utils.Equal(t, result.Title, filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Output, "Created file: "+filepath.ToSlash(expectedPath))

	// Verify file was created in correct location
	content, err := os.ReadFile(expectedPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "nested file content")
}

func TestWriteFile_AbsolutePathWithinBase(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "absdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	// Use absolute path within base
	absPath := filepath.Join(subDir, "absfile.txt")
	input := &tools.WriteFileInput{
		FilePath: absPath,
		Content:  "absolute path content",
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	utils.Equal(t, result.Title, filepath.ToSlash(absPath))
	utils.Equal(t, result.Output, "Created file: "+filepath.ToSlash(absPath))

	// Verify file was created
	content, err := os.ReadFile(absPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "absolute path content")
}

func TestWriteFile_LargeContent(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create large content (1MB)
	largeContent := strings.Repeat("This is a test line that will be repeated many times.\n", 20000)
	targetPath := "large.txt"

	input := &tools.WriteFileInput{
		FilePath: targetPath,
		Content:  largeContent,
	}
	result, err := tools.WriteFile(context.Background(), input)
	utils.NilError(t, err)

	expectedPath := filepath.Join(tempDir, targetPath)
	utils.Equal(t, result.Title, filepath.ToSlash(expectedPath))
	utils.Equal(t, result.Output, "Created file: "+filepath.ToSlash(expectedPath))

	// Verify large file was written correctly
	content, err := os.ReadFile(expectedPath)
	utils.NilError(t, err)
	utils.Equal(t, string(content), largeContent)
	utils.Equal(t, len(content), len(largeContent))
}
