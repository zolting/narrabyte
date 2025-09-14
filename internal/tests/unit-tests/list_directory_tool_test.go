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

func TestListDirectory_NilInput(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	result, err := tools.ListDirectory(context.Background(), nil)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, tempDir), true)
}

func TestListDirectory_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ListLSInput{
		Path: "",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, tempDir), true)
}

func TestListDirectory_ProjectRootNotSet(t *testing.T) {
	// Reset the base root
	tools.SetListDirectoryBaseRoot("")

	input := &tools.ListLSInput{
		Path: ".",
	}
	_, err := tools.ListDirectory(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "list directory base root not set"), true)
}

func TestListDirectory_PathEscapesBase(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ListLSInput{
		Path: "../../../etc",
	}
	_, err := tools.ListDirectory(context.Background(), input)
	utils.Equal(t, err.Error(), "path escapes the configured base root")
}

func TestListDirectory_PathEscapesBaseAbsolute(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Try to access a directory outside the temp dir
	outsidePath := "/etc"
	input := &tools.ListLSInput{
		Path: outsidePath,
	}
	_, err := tools.ListDirectory(context.Background(), input)
	utils.Equal(t, err.Error(), "path escapes the configured base root")
}

func TestListDirectory_PathIsFile(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: "test.txt",
	}
	_, err = tools.ListDirectory(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "not a directory"), true)
}

func TestListDirectory_PathDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ListLSInput{
		Path: "nonexistent",
	}
	_, err := tools.ListDirectory(context.Background(), input)
	utils.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)
}

func TestListDirectory_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, tempDir), true)
	// Should not contain any files or subdirectories other than the header
	lines := strings.Split(result, "\n")
	hasContent := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, tempDir) {
			hasContent = true
			break
		}
	}
	utils.Equal(t, hasContent, false)
}

func TestListDirectory_WithFiles(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create some files
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file2.go"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "file1.txt"), true)
	utils.Equal(t, strings.Contains(result, "file2.go"), true)
}

func TestListDirectory_WithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create subdirectory structure
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "subdir/"), true)
	utils.Equal(t, strings.Contains(result, "nested.txt"), true)
}

func TestListDirectory_NestedStructure(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create nested structure
	level1 := filepath.Join(tempDir, "level1")
	err := os.Mkdir(level1, 0755)
	utils.NilError(t, err)

	level2 := filepath.Join(level1, "level2")
	err = os.Mkdir(level2, 0755)
	utils.NilError(t, err)

	err = os.WriteFile(filepath.Join(level2, "deep.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "level1/"), true)
	utils.Equal(t, strings.Contains(result, "level2/"), true)
	utils.Equal(t, strings.Contains(result, "deep.txt"), true)
}

func TestListDirectory_IgnoreDefaultPatterns(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files and directories that should be ignored
	ignoredDir := filepath.Join(tempDir, "node_modules")
	err := os.Mkdir(ignoredDir, 0755)
	utils.NilError(t, err)

	err = os.WriteFile(filepath.Join(ignoredDir, "package.json"), []byte("content"), 0644)
	utils.NilError(t, err)

	// Create a normal file that should be shown
	err = os.WriteFile(filepath.Join(tempDir, "normal.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "normal.txt"), true)
	utils.Equal(t, strings.Contains(result, "node_modules"), false)
	utils.Equal(t, strings.Contains(result, "package.json"), false)
}

func TestListDirectory_CustomIgnorePatterns(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files
	err := os.WriteFile(filepath.Join(tempDir, "important.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "temp.log"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path:   ".",
		Ignore: []string{"temp.log"}, // Exact filename match (no wildcards supported)
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "important.txt"), true)
	utils.Equal(t, strings.Contains(result, "temp.log"), false)
}

func TestListDirectory_IgnoreDirectoryWithCustomPattern(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create custom directory to ignore
	customDir := filepath.Join(tempDir, "custom_ignore")
	err := os.Mkdir(customDir, 0755)
	utils.NilError(t, err)

	err = os.WriteFile(filepath.Join(customDir, "file.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	// Create normal file
	err = os.WriteFile(filepath.Join(tempDir, "normal.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path:   ".",
		Ignore: []string{"custom_ignore/"},
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "normal.txt"), true)
	utils.Equal(t, strings.Contains(result, "custom_ignore"), false)
	utils.Equal(t, strings.Contains(result, "file.txt"), false)
}

func TestListDirectory_LimitReached(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create more files than the limit (100)
	for i := 0; i < 110; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%03d.txt", i))
		err := os.WriteFile(filename, []byte("content"), 0644)
		utils.NilError(t, err)
	}

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)

	// Should contain the directory header
	utils.Equal(t, strings.Contains(result, tempDir), true)

	// Count the number of file entries in the result
	lines := strings.Split(result, "\n")
	fileCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ".txt") {
			fileCount++
		}
	}

	// Should be limited to listLimit (100)
	utils.Equal(t, fileCount <= 100, true)
}

func TestListDirectory_SpecificSubdirectory(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create subdirectory structure
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	// Create file in root that should not appear
	err = os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: "subdir",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "file1.txt"), true)
	utils.Equal(t, strings.Contains(result, "file2.txt"), true)
	utils.Equal(t, strings.Contains(result, "root.txt"), false)
}

func TestListDirectory_RelativePath(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	utils.NilError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "test.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: "./subdir",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "test.txt"), true)
}

func TestListDirectory_ComplexIgnorePatterns(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create various files and directories
	testCases := []struct {
		path         string
		isDir        bool
		shouldIgnore bool
	}{
		{"node_modules", true, true},               // Default ignore pattern
		{"node_modules/package.json", false, true}, // File in ignored directory
		{"__pycache__", true, true},                // Default ignore pattern
		{"__pycache__/module.pyc", false, true},    // File in ignored directory
		{".git", true, true},                       // Default ignore pattern
		{".git/config", false, true},               // File in ignored directory
		{"normal.txt", false, false},               // Normal file
		{"src", true, false},                       // Normal directory
		{"src/main.go", false, false},              // File in normal directory
	}

	for _, tc := range testCases {
		fullPath := filepath.Join(tempDir, tc.path)
		if tc.isDir {
			err := os.MkdirAll(fullPath, 0755)
			utils.NilError(t, err)
		} else {
			err := os.WriteFile(fullPath, []byte("content"), 0644)
			utils.NilError(t, err)
		}
	}

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)

	for _, tc := range testCases {
		if tc.shouldIgnore {
			utils.Equal(t, strings.Contains(result, filepath.Base(tc.path)), false)
		} else {
			utils.Equal(t, strings.Contains(result, filepath.Base(tc.path)), true)
		}
	}
}

// Additional edge case tests
func TestListDirectory_EmptyIgnorePatterns(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path:   ".",
		Ignore: []string{""}, // Empty ignore patterns
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "file1.txt"), true)
	utils.Equal(t, strings.Contains(result, "file2.txt"), true)
}

func TestListDirectory_UnicodeNames(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create files with unicode names
	err := os.WriteFile(filepath.Join(tempDir, "文件.txt"), []byte("content"), 0644)
	utils.NilError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "файл.txt"), []byte("content"), 0644)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "文件.txt"), true)
	utils.Equal(t, strings.Contains(result, "файл.txt"), true)
}

func TestListDirectory_Symlinks(t *testing.T) {
	tempDir := t.TempDir()
	tools.SetListDirectoryBaseRoot(tempDir)

	// Create a file and a symlink to it
	realFile := filepath.Join(tempDir, "real.txt")
	err := os.WriteFile(realFile, []byte("content"), 0644)
	utils.NilError(t, err)

	linkFile := filepath.Join(tempDir, "link.txt")
	err = os.Symlink(realFile, linkFile)
	utils.NilError(t, err)

	input := &tools.ListLSInput{
		Path: ".",
	}
	result, err := tools.ListDirectory(context.Background(), input)
	utils.NilError(t, err)
	utils.Equal(t, strings.Contains(result, "real.txt"), true)
	utils.Equal(t, strings.Contains(result, "link.txt"), true)
}
