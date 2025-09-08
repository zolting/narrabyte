package integration_tests

import (
	"testing"

	"narrabyte/internal/services"
	"narrabyte/internal/utils"
)

// TestTreeJSON_ProjectRoot generates the tree JSON for the project root
// and prints it to the test output (use `go test -v` to see it).
func TestTreeJSON_ProjectRoot(t *testing.T) {
	root, err := utils.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	g := &services.GitService{}

	// Use generous limits so the structure is useful while avoiding huge outputs.
	opts := services.TreeOptions{
		MaxDepth:           2,    // increase from default to capture more structure
		MaxEntries:         5000, // allow more nodes if needed
		ExtraExcludeGlobs:  nil,  // rely on defaults (.git, node_modules, etc.)
		CollapseDirEntries: 200,  // collapse very large directories
	}

	data, err := g.ListTreeJSON(root, opts)
	if err != nil {
		t.Fatalf("ListTreeJSON error: %v", err)
	}

	// Print the JSON in test logs. Run with `go test -v` to see full output.
	t.Logf("\nProject Tree JSON (root=%s):\n%s\n", root, string(data))
}
