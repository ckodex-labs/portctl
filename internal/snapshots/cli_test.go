package snapshots

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
)

// This test invokes the portctl CLI via the entrypoint at cmd/portctl/main.go.
// If you later move the main entry point, update the path accordingly.
func TestPortctlHelpSnapshot(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/portctl/main.go", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl help: %v\nOutput: %s", err, output)
	}
	
	// Filter out compiler warnings from the output
	lines := strings.Split(string(output), "\n")
	var filteredLines []string
	for _, line := range lines {
		// Skip lines containing compiler warnings
		if !strings.Contains(line, "warning:") && !strings.Contains(line, "go-m1cpu") {
			filteredLines = append(filteredLines, line)
		}
	}
	
	filteredOutput := strings.Join(filteredLines, "\n")
	cupaloy.SnapshotT(t, filteredOutput)
}


// Add more tests for other CLI commands as needed
