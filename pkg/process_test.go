package process

import (
	"context"
	"testing"
)

func TestNewProcessManager(t *testing.T) {
	pm := NewProcessManager()
	if pm == nil {
		t.Error("NewProcessManager should not return nil")
	}
}

func TestGetAllProcesses(t *testing.T) {
	pm := NewProcessManager()
	processes, err := pm.GetAllProcesses(context.Background())

	// We can't guarantee any specific processes will be running,
	// but the function should not error on a normal system
	if err != nil {
		t.Logf("GetAllProcesses returned error (this might be expected in some test environments): %v", err)
	}

	// If processes are found, validate their structure
	for _, proc := range processes {
		if proc.PID <= 0 {
			t.Errorf("Process PID should be positive, got %d", proc.PID)
		}
		if proc.Port <= 0 {
			t.Errorf("Process port should be positive, got %d", proc.Port)
		}
		if proc.Protocol == "" {
			t.Error("Process protocol should not be empty")
		}
		if proc.Command == "" {
			t.Error("Process command should not be empty")
		}
	}
}

func TestGetProcessesOnPort(t *testing.T) {
	pm := NewProcessManager()

	// Test with a port that's very unlikely to be in use
	processes, err := pm.GetProcessesOnPort(context.Background(), 65432)
	if err != nil {
		t.Logf("GetProcessesOnPort returned error (might be expected): %v", err)
		// In case of error, we still expect a valid slice (empty or nil)
		return
	}

	// Should return empty slice for unused port
	if processes == nil {
		processes = []Process{} // Ensure it's at least an empty slice
	}

	// Validate that we got a valid response
	if len(processes) > 0 {
		t.Logf("Found %d processes on port 65432 (unexpected but valid)", len(processes))
	}
}

// Test parsing functions with sample data
func TestParseLsofLine(t *testing.T) {
	pm := NewProcessManager()

	// Sample lsof output line
	line := "node      12345 user   23u  IPv4 0x1234567890      0t0  TCP *:8080 (LISTEN)"

	process := pm.parseLsofLine(line, 0)
	if process == nil {
		t.Error("parseLsofLine should parse valid line")
		return
	}

	if process.PID != 12345 {
		t.Errorf("Expected PID 12345, got %d", process.PID)
	}

	if process.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", process.Port)
	}

	if process.Command != "node" {
		t.Errorf("Expected command 'node', got '%s'", process.Command)
	}
}

func TestParseNetstatLine(t *testing.T) {
	pm := NewProcessManager()

	// Sample netstat output line
	line := "tcp        0      0 0.0.0.0:8080            0.0.0.0:*               LISTEN      12345/node"

	process := pm.parseNetstatLine(line, 0)
	if process == nil {
		t.Error("parseNetstatLine should parse valid line")
		return
	}

	if process.PID != 12345 {
		t.Errorf("Expected PID 12345, got %d", process.PID)
	}

	if process.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", process.Port)
	}

	if process.Command != "node" {
		t.Errorf("Expected command 'node', got '%s'", process.Command)
	}
}

// Benchmark tests
func BenchmarkGetAllProcesses(b *testing.B) {
	pm := NewProcessManager()
	for i := 0; i < b.N; i++ {
		_, _ = pm.GetAllProcesses(context.Background())
	}
}

func BenchmarkGetProcessesOnPort(b *testing.B) {
	pm := NewProcessManager()
	for i := 0; i < b.N; i++ {
		_, _ = pm.GetProcessesOnPort(context.Background(), 8080)
	}
}
