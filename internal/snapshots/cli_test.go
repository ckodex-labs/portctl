package snapshots

import (
	"os/exec"
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
	cupaloy.SnapshotT(t, string(output))
}


// Add more tests for other CLI commands as needed
