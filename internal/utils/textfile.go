package utils

import (
	"bufio"
	"os"
	"strings"
)

// ReadNonEmptyLines reads a text file and returns all non-empty, trimmed lines.
// Lines consisting only of whitespace or starting with # (comments) are ignored.
func ReadNonEmptyLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
