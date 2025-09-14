package tools

import (
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type WriteFileInput struct {
    // FilePath is the path to the file to write. Absolute paths are allowed only if under the configured project root.
    FilePath string `json:"file_path" jsonschema:"description=The path to the file to write (absolute or relative to project root)"`
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
func WriteFile(_ context.Context, in *WriteFileInput) (*WriteFileOutput, error) {
    if in == nil {
        return nil, errors.New("input is required")
    }

    base, err := getListDirectoryBaseRoot()
    if err != nil {
        return nil, err
    }

    p := strings.TrimSpace(in.FilePath)
    if p == "" {
        return nil, fmt.Errorf("file_path is required")
    }

    // Resolve target path under base, ensuring it cannot escape.
    var absPath string
    if filepath.IsAbs(p) {
        // Ensure absolute path is under base
        absBase, err := filepath.Abs(base)
        if err != nil {
            return nil, err
        }
        absCandidate, err := filepath.Abs(p)
        if err != nil {
            return nil, err
        }
        relToBase, err := filepath.Rel(absBase, absCandidate)
        if err != nil {
            return nil, err
        }
        if strings.HasPrefix(relToBase, "..") {
            return nil, fmt.Errorf("file %s is not in the configured project root", p)
        }
        absPath = absCandidate
    } else {
        abs, ok := safeJoinUnderBase(base, p)
        if !ok {
            return nil, fmt.Errorf("path escapes the configured base root")
        }
        absPath = abs
    }

    // Ensure parent directory exists
    dir := filepath.Dir(absPath)
    info, err := os.Stat(dir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("directory does not exist: %s", dir)
        }
        return nil, err
    }
    if !info.IsDir() {
        return nil, fmt.Errorf("not a directory: %s", dir)
    }

    // Determine if file existed
    existed := false
    if st, err := os.Stat(absPath); err == nil && !st.IsDir() {
        existed = true
    }

    // Write file (creates or truncates)
    if err := os.WriteFile(absPath, []byte(in.Content), 0o644); err != nil {
        return nil, err
    }

    // Build title relative to base
    rel, err := filepath.Rel(base, absPath)
    if err != nil {
        rel = absPath
    }
    rel = filepath.ToSlash(rel)

    outputMsg := ""
    if existed {
        outputMsg = fmt.Sprintf("Overwrote file: %s", rel)
    } else {
        outputMsg = fmt.Sprintf("Created file: %s", rel)
    }

    return &WriteFileOutput{
        Title:  rel,
        Output: outputMsg,
        Metadata: map[string]string{
            "filepath": rel,
            "exists":   fmt.Sprintf("%v", existed),
        },
    }, nil
}

