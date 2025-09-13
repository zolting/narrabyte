package tools

import (
    "embed"
    "fmt"
    "strings"
)

// toolDescFS embeds all .txt files in this package directory as tool descriptions.
// The convention is: a tool key like "list_directory_tool" maps to "list_directory_tool.txt".
//go:embed *.txt
var toolDescFS embed.FS

// ToolDescription returns the embedded description text for the given tool key.
// It looks up a file named "<toolKey>.txt" in this package. If not found, returns "".
func ToolDescription(toolKey string) string {
    key := strings.TrimSpace(toolKey)
    if key == "" {
        return ""
    }
    // Allow callers to pass either the base key or the filename
    if strings.HasSuffix(key, ".txt") {
        key = strings.TrimSuffix(key, ".txt")
    }
    file := fmt.Sprintf("%s.txt", key)
    b, err := toolDescFS.ReadFile(file)
    if err != nil {
        return ""
    }
    return string(b)
}

