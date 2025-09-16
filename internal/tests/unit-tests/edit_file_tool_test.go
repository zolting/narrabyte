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

func TestEdit_NilInput(t *testing.T) {
	_, err := tools.Edit(context.Background(), nil)
	utils.NilError(t, err)
	// Should return format error
}

func TestEdit_EmptyFilePath(t *testing.T) {
	// Set a temporary project root
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.EditInput{
		FilePath:  "",
		OldString: "old",
		NewString: "new",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, output.Output, "Format error: file_path is required")
}

func TestEdit_SameOldNewString(t *testing.T) {
	input := &tools.EditInput{
		FilePath:  "/tmp/test.txt",
		OldString: "same",
		NewString: "same",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, output.Output, "Format error: old_string and new_string must be different")
}

func TestEdit_ProjectRootNotSet(t *testing.T) {
	// Reset the base root
	tools.SetListDirectoryBaseRoot("")

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "old",
		NewString: "new",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, output.Output, "Format error: project root not set")
}

func TestEdit_PathEscapesBase(t *testing.T) {
	// Set up a temporary directory as base
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.EditInput{
		FilePath:  "../../../etc/passwd",
		OldString: "old",
		NewString: "new",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, output.Output, "Format error: path escapes the configured project root")
}

func TestEdit_DirectoryDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.EditInput{
		FilePath:  "nonexistent/dir/file.txt",
		OldString: "old",
		NewString: "new",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, strings.Contains(output.Output, "directory does not exist"), true)
}

func TestEdit_PathIsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "subdir",
		OldString: "old",
		NewString: "new",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, strings.Contains(output.Output, "cannot edit directory"), true)
}

func TestEdit_BinaryFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a binary file (zip extension)
	binaryFile := filepath.Join(tempDir, "test.zip")
	err := os.WriteFile(binaryFile, []byte{0x50, 0x4B, 0x03, 0x04}, 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.zip",
		OldString: "old",
		NewString: "new",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "format_error")
	utils.Equal(t, strings.Contains(output.Output, "cannot edit binary file"), true)
}

func TestEdit_OverwriteEntireFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "",
		NewString: "new content",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")
	utils.Equal(t, output.Metadata["occurrences"], "1")

	// Verify file content
	content, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "new content")
}

func TestEdit_CreateNewFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.EditInput{
		FilePath:  "newfile.txt",
		OldString: "",
		NewString: "new file content",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")
	utils.Equal(t, output.Metadata["occurrences"], "1")

	// Verify file was created
	testFile := filepath.Join(tempDir, "newfile.txt")
	content, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "new file content")
}

func TestEdit_SimpleReplace(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "world",
		NewString: "universe",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")
	utils.Equal(t, output.Metadata["occurrences"], "1")

	// Verify file content
	content, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "hello universe")
}

func TestEdit_ReplaceAll(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("foo bar foo baz foo"), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:   "test.txt",
		OldString:  "foo",
		NewString:  "qux",
		ReplaceAll: true,
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")
	utils.Equal(t, output.Metadata["occurrences"], "3")

	// Verify file content
	content, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, string(content), "qux bar qux baz qux")
}

func TestEdit_OldStringNotFound(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "nonexistent",
		NewString: "replacement",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "search_not_found")
	utils.Equal(t, output.Output, "Edit error: old_string not found in content")
}

func TestEdit_AmbiguousMatch(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("foo bar foo baz"), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "foo",
		NewString: "qux",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "ambiguous_match")
	utils.Equal(t, output.Output, "Edit error: old_string found multiple times and requires more code context to uniquely identify the intended match")
}

// Test more complex replacement scenarios
func TestEdit_LineTrimmedReplace(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "function test() {\n    console.log('hello');\n    return true;\n}"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "console.log('hello')",
		NewString: "console.log('world')",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")

	// Verify file content
	result, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	expected := "function test() {\n    console.log('world');\n    return true;\n}"
	utils.Equal(t, string(result), expected)
}

func TestEdit_BlockAnchorReplace(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "function oldFunc() {\n    // some code\n    return false;\n}\n\nfunction newFunc() {\n    // other code\n    return true;\n}"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "function oldFunc() {\n    // some code\n    return false;\n}",
		NewString: "function updatedFunc() {\n    // updated code\n    return true;\n}",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")

	// Verify file content
	result, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(string(result), "function updatedFunc()"), true)
	utils.Equal(t, strings.Contains(string(result), "// updated code"), true)
}

func TestEdit_WhitespaceNormalizedReplace(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "key1=value1\nkey2  =  value2\nkey3=value3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "key2  =  value2",
		NewString: "key2=updated",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")

	// Verify file content
	result, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(string(result), "key2=updated"), true)
}

func TestEdit_IndentationFlexibleReplace(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "if (condition) {\n        doSomething();\n    }"
	err := os.WriteFile(testFile, []byte(content), 0644)
	utils.NilError(t, err)

	input := &tools.EditInput{
		FilePath:  "test.txt",
		OldString: "if (condition) {\n    doSomething();\n}",
		NewString: "if (condition) {\n    doSomethingElse();\n}",
	}
	output, err := tools.Edit(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, output.Metadata["error"], "")
	utils.Equal(t, output.Metadata["replaced"], "true")

	// Verify file content
	result, err := os.ReadFile(testFile)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(string(result), "doSomethingElse()"), true)
}
