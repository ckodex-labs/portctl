package cmd

import (
	"os/exec"
	"strings"
	"testing"
)

func TestMCP_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "./portctl/main.go", "mcp", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl mcp --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "MCP server") {
		t.Errorf("Expected MCP description in help, got: %s", outputStr)
	}
}

func TestMCP_CommandExists(t *testing.T) {
	// Verify the mcp command is registered
	cmd := exec.Command("go", "run", "./portctl/main.go", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl --help: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "mcp") {
		t.Errorf("Expected 'mcp' command in help output, got: %s", outputStr)
	}
}

// Note: Testing the actual MCP server functionality requires a JSON-RPC client
// which is beyond the scope of basic CLI tests. The server communicates over stdio
// using the MCP protocol, so integration tests would need to:
// 1. Start the server process
// 2. Send JSON-RPC requests over stdin
// 3. Read JSON-RPC responses from stdout
// 4. Verify tool calls work correctly
//
// For now, we verify that:
// - The command exists and is registered
// - Help text is available
// - The command can be invoked without crashing
