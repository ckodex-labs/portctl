package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	process "dagger/portctl/pkg"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol (MCP) server",
	Long: `Start the MCP server to allow AI agents to interact with portctl.
This command runs a JSON-RPC server over stdio.`,
	Run: runMCP,
}

func runMCP(cmd *cobra.Command, args []string) {
	// Create MCP server
	s := server.NewMCPServer(
		"portctl",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Register tools
	registerListProcessesTool(s)
	registerKillProcessTool(s)
	registerScanPortsTool(s)
	registerSystemStatsTool(s)

	// Serve stdio
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func registerListProcessesTool(s *server.MCPServer) {
	tool := mcp.NewTool("list_processes",
		mcp.WithDescription("List running processes, optionally filtered by port or service"),
		mcp.WithNumber("port",
			mcp.Description("Specific port to check"),
		),
		mcp.WithString("service",
			mcp.Description("Filter by service name (e.g., 'node', 'python')"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pm := process.NewProcessManager()

		var processes []process.Process
		var err error

		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			args = make(map[string]any)
		}

		port, ok := args["port"].(float64)
		if ok && port > 0 {
			processes, err = pm.GetProcessesOnPort(ctx, int(port))
		} else {
			processes, err = pm.GetAllProcesses(ctx)
		}

		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting processes: %v", err)), nil
		}

		// Apply service filter if present
		service, ok := args["service"].(string)
		if ok && service != "" {
			filterOpts := process.FilterOptions{Service: service}
			processes = pm.FilterProcesses(processes, filterOpts)
		}

		return mcp.NewToolResultText(fmt.Sprintf("%v", processes)), nil
	})
}

func registerKillProcessTool(s *server.MCPServer) {
	tool := mcp.NewTool("kill_process",
		mcp.WithDescription("Kill a process by PID or Port"),
		mcp.WithNumber("pid",
			mcp.Description("Process ID to kill"),
		),
		mcp.WithNumber("port",
			mcp.Description("Port number to kill processes on"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force kill (SIGKILL)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pm := process.NewProcessManager()

		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			args = make(map[string]any)
		}

		force, _ := args["force"].(bool)
		pid, pidOk := args["pid"].(float64)
		port, portOk := args["port"].(float64)

		if !pidOk && !portOk {
			return mcp.NewToolResultError("Must provide either 'pid' or 'port'"), nil
		}

		if pidOk {
			err := pm.KillProcess(ctx, int(pid), force)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to kill PID %d: %v", int(pid), err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Successfully killed process with PID %d", int(pid))), nil
		}

		if portOk {
			processes, err := pm.GetProcessesOnPort(ctx, int(port))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error finding processes on port %d: %v", int(port), err)), nil
			}

			if len(processes) == 0 {
				return mcp.NewToolResultText(fmt.Sprintf("No processes found on port %d", int(port))), nil
			}

			var pids []int
			for _, p := range processes {
				pids = append(pids, p.PID)
			}

			results := pm.KillProcesses(ctx, pids, force)

			// Summarize results
			successCount := 0
			var errors []string
			for _, err := range results {
				if err == nil {
					successCount++
				} else {
					errors = append(errors, err.Error())
				}
			}

			msg := fmt.Sprintf("Killed %d/%d processes on port %d", successCount, len(pids), int(port))
			if len(errors) > 0 {
				msg += fmt.Sprintf("\nErrors: %v", errors)
			}
			return mcp.NewToolResultText(msg), nil
		}

		return mcp.NewToolResultError("Invalid arguments"), nil
	})
}

func registerScanPortsTool(s *server.MCPServer) {
	tool := mcp.NewTool("scan_ports",
		mcp.WithDescription("Scan for open ports on a host"),
		mcp.WithString("host",
			mcp.Description("Host to scan (default: localhost)"),
		),
		mcp.WithNumber("start_port",
			mcp.Description("Start of port range"),
		),
		mcp.WithNumber("end_port",
			mcp.Description("End of port range"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			args = make(map[string]any)
		}

		host, _ := args["host"].(string)
		if host == "" {
			host = "localhost"
		}

		startPort, ok := args["start_port"].(float64)
		if !ok {
			startPort = 1
		}
		endPort, ok := args["end_port"].(float64)
		if !ok {
			endPort = 1000
		}

		// Use the scan logic from scan.go (we need to expose it or duplicate it slightly since it's in the same package 'cmd')
		// Since we are in package 'cmd', we can call scanPorts directly if it's exported or just reuse the logic.
		// scanPorts is in scan.go but it's not exported (lowercase).
		// However, since we are in the same package `cmd`, we CAN access `scanPorts`!

		var ports []int
		for p := int(startPort); p <= int(endPort); p++ {
			ports = append(ports, p)
		}

		results := scanPorts(host, ports)

		var openPorts []ScanResult
		for _, r := range results {
			if r.Status == "open" {
				openPorts = append(openPorts, r)
			}
		}

		return mcp.NewToolResultText(fmt.Sprintf("Open ports on %s: %v", host, openPorts)), nil
	})
}

func registerSystemStatsTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_system_stats",
		mcp.WithDescription("Get system resource usage and statistics"),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pm := process.NewProcessManager()
		stats, err := pm.GetSystemStats(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting stats: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("%+v", stats)), nil
	})
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
