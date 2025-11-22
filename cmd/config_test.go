package cmd

import (
	"os/exec"
	"strings"
	"testing"
)

func TestConfigSet_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "../cmd/portctl/main.go", "config", "set", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl config set --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}
}

func TestList_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "list", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl list --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}
}

func TestQuick_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "quick", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl quick --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}
}

func TestScan_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "scan", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl scan --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}
}

func TestKill_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "kill", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl kill --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}
}

func TestWatch_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "watch", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run portctl watch --help: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Errorf("Expected help output, got empty string")
	}
}

func TestConfigSetAndGet(t *testing.T) {
	// Set a valid config value
	setCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "set", "output.format", "json")
	setOut, setErr := setCmd.CombinedOutput()
	if setErr != nil {
		t.Fatalf("Failed to set config value: %v\nOutput: %s", setErr, setOut)
	}
	// Get the config value
	getCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "get", "output.format")
	getOut, getErr := getCmd.CombinedOutput()
	if getErr != nil {
		t.Fatalf("Failed to get config value: %v\nOutput: %s", getErr, getOut)
	}
	if !strings.Contains(string(getOut), "json") {
		t.Errorf("Expected to get 'json', got: %s", getOut)
	}
}

func TestConfigSetAndGet_OutputColors(t *testing.T) {
	setCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "set", "output.colors", "false")
	setOut, setErr := setCmd.CombinedOutput()
	if setErr != nil {
		t.Fatalf("Failed to set output.colors: %v\nOutput: %s", setErr, setOut)
	}
	getCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "get", "output.colors")
	getOut, getErr := getCmd.CombinedOutput()
	if getErr != nil {
		t.Fatalf("Failed to get output.colors: %v\nOutput: %s", getErr, getOut)
	}
	if !strings.Contains(string(getOut), "false") {
		t.Errorf("Expected to get 'false', got: %s", getOut)
	}
}

func TestConfigSetAndGet_ScanTimeout(t *testing.T) {
	setCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "set", "scan.timeout", "30s")
	setOut, setErr := setCmd.CombinedOutput()
	if setErr != nil {
		t.Fatalf("Failed to set scan.timeout: %v\nOutput: %s", setErr, setOut)
	}
	getCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "get", "scan.timeout")
	getOut, getErr := getCmd.CombinedOutput()
	if getErr != nil {
		t.Fatalf("Failed to get scan.timeout: %v\nOutput: %s", getErr, getOut)
	}
	if !strings.Contains(string(getOut), "30s") {
		t.Errorf("Expected to get '30s', got: %s", getOut)
	}
}

func TestConfigSetAndGet_KillConfirm(t *testing.T) {
	setCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "set", "kill.confirm", "true")
	setOut, setErr := setCmd.CombinedOutput()
	if setErr != nil {
		t.Fatalf("Failed to set kill.confirm: %v\nOutput: %s", setErr, setOut)
	}
	getCmd := exec.Command("go", "run", "cmd/portctl/main.go", "config", "get", "kill.confirm")
	getOut, getErr := getCmd.CombinedOutput()
	if getErr != nil {
		t.Fatalf("Failed to get kill.confirm: %v\nOutput: %s", getErr, getOut)
	}
	if !strings.Contains(string(getOut), "true") {
		t.Errorf("Expected to get 'true', got: %s", getOut)
	}
}

func TestList_Exec(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "list")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "No processes found") {
		t.Fatalf("Failed to run portctl list: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(string(output), "PID") && !strings.Contains(string(output), "No processes found") {
		t.Errorf("Expected process table header or no processes message, got: %s", output)
	}
}

func TestScan_Exec(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "scan", "127.0.0.1", "80")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "Found") && !strings.Contains(string(output), "open port(s)") {
		t.Fatalf("Failed to run portctl scan: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(string(output), "Found") && !strings.Contains(string(output), "open port(s)") && !strings.Contains(string(output), "No open ports found") {
		t.Errorf("Expected scan result, got: %s", output)
	}
}

func TestQuick_Exec(t *testing.T) {
	cmd := exec.Command("go", "run", "cmd/portctl/main.go", "quick", "next-port")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "Next available port") {
		t.Fatalf("Failed to run portctl quick next-port: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(string(output), "Next available port") && !strings.Contains(string(output), "No dev processes found") {
		t.Errorf("Expected quick action result, got: %s", output)
	}
}

func TestKill_Exec(t *testing.T) {
	cmd := exec.Command("go", "run", "./portctl/main.go", "kill")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error when running kill with no PID, got: %s", output)
	}
	if !strings.Contains(string(output), "Usage") && !strings.Contains(string(output), "PID") {
		t.Errorf("Expected usage or PID message, got: %s", output)
	}
}

func TestWatch_Exec(t *testing.T) {
	cmd := exec.Command("go", "run", "./portctl/main.go", "watch", "--interval=1s", "--count=1")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "Watching processes") {
		t.Fatalf("Failed to run portctl watch: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(string(output), "Watching processes") && !strings.Contains(string(output), "No processes found") && !strings.Contains(string(output), "Watch stopped after") {
		t.Errorf("Expected watch output, got: %s", output)
	}
}
