package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	process "dagger/portctl/pkg"
)

var (
	killPID     int
	killForce   bool
	killYes     bool
	killRange   string
	killService string
	killUser    string
	killOlder   string
)

var killCmd = &cobra.Command{
	Use:   "kill [port|ports...]",
	Short: "Kill processes running on specific ports with advanced options",
	Long: `Kill processes that are currently listening on specified ports.
Supports killing multiple processes with various filtering options.

Examples:
  # Single operations
  portctl kill 8080                    # Kill processes on port 8080
  portctl kill --pid 12345             # Kill process with PID 12345
  
  # Multiple ports
  portctl kill 8080 3000 5000          # Kill processes on multiple ports
  portctl kill --range "3000-3010"     # Kill processes in port range
  
  # Filtering
  portctl kill --service node          # Kill all Node.js processes
  portctl kill --user john             # Kill processes owned by user 'john'
  portctl kill --older "1h"            # Kill processes older than 1 hour
  
  # Options
  portctl kill 8080 --force            # Force kill (SIGKILL)
  portctl kill 8080 --yes              # Skip confirmation prompt`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow multiple ports or no args if using filters
		if killPID != 0 || killRange != "" || killService != "" || killUser != "" || killOlder != "" {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("specify at least one port or use filtering options")
		}
		return nil
	},
	Run: runKill,
}

func runKill(cmd *cobra.Command, args []string) {
	pm := process.NewProcessManager()

	// Handle single PID kill
	if killPID != 0 {
		killProcessByPID(pm, killPID)
		return
	}

	// Collect all target processes
	var targetProcesses []process.Process
	var err error

	// Handle filtering options
	if killService != "" || killUser != "" || killOlder != "" {
		targetProcesses, err = getFilteredProcesses(pm)
		if err != nil {
			color.Red("Error filtering processes: %v", err)
			os.Exit(1)
		}
	}

	// Handle port range
	if killRange != "" {
		rangeProcesses, err := getProcessesInRange(pm, killRange)
		if err != nil {
			color.Red("Error parsing port range: %v", err)
			os.Exit(1)
		}
		targetProcesses = append(targetProcesses, rangeProcesses...)
	}

	// Handle individual ports
	if len(args) > 0 {
		for _, portStr := range args {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				color.Red("Invalid port number: %s", portStr)
				os.Exit(1)
			}

			processes, err := pm.GetProcessesOnPort(port)
			if err != nil {
				color.Red("Error getting processes on port %d: %v", port, err)
				continue
			}
			targetProcesses = append(targetProcesses, processes...)
		}
	}

	if len(targetProcesses) == 0 {
		color.Yellow("No matching processes found")
		return
	}

	// Remove duplicates
	targetProcesses = removeDuplicateProcesses(targetProcesses)

	// Kill multiple processes
	killMultipleProcesses(pm, targetProcesses)
}

func killProcessByPID(pm *process.ProcessManager, pid int) {
	if !killYes {
		if !confirmKill(fmt.Sprintf("process with PID %d", pid)) {
			color.Yellow("Operation cancelled")
			return
		}
	}

	color.Yellow("Killing process %d...", pid)
	err := pm.KillProcess(pid, killForce)
	if err != nil {
		color.Red("Failed to kill process %d: %v", pid, err)
		os.Exit(1)
	}

	color.Green("Successfully killed process %d", pid)
}

func confirmKill(target string) bool {
	reader := bufio.NewReader(os.Stdin)

	var prompt string
	if killForce {
		prompt = color.YellowString("Are you sure you want to FORCE KILL %s? [y/N]: ", target)
	} else {
		prompt = color.YellowString("Are you sure you want to kill %s? [y/N]: ", target)
	}

	fmt.Print(prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func getFilteredProcesses(pm *process.ProcessManager) ([]process.Process, error) {
	allProcesses, err := pm.GetAllProcesses()
	if err != nil {
		return nil, err
	}

	var filtered []process.Process

	for _, proc := range allProcesses {
		match := true

		// Filter by service type
		if killService != "" {
			if !strings.Contains(strings.ToLower(proc.ServiceType), strings.ToLower(killService)) &&
				!strings.Contains(strings.ToLower(proc.Command), strings.ToLower(killService)) {
				match = false
			}
		}

		// Filter by user
		if killUser != "" {
			if !strings.Contains(strings.ToLower(proc.User), strings.ToLower(killUser)) {
				match = false
			}
		}

		// Filter by age
		if killOlder != "" {
			duration, err := time.ParseDuration(killOlder)
			if err != nil {
				return nil, fmt.Errorf("invalid duration format: %s", killOlder)
			}
			if proc.StartTime.IsZero() || time.Since(proc.StartTime) < duration {
				match = false
			}
		}

		if match {
			filtered = append(filtered, proc)
		}
	}

	return filtered, nil
}

func getProcessesInRange(pm *process.ProcessManager, rangeStr string) ([]process.Process, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format, use 'start-end' (e.g., '3000-3010')")
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid start port: %s", parts[0])
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid end port: %s", parts[1])
	}

	if start >= end {
		return nil, fmt.Errorf("start port must be less than end port")
	}

	var processes []process.Process
	for port := start; port <= end; port++ {
		procs, err := pm.GetProcessesOnPort(port)
		if err != nil {
			continue // Skip errors for individual ports
		}
		processes = append(processes, procs...)
	}

	return processes, nil
}

func removeDuplicateProcesses(processes []process.Process) []process.Process {
	seen := make(map[int]bool)
	var unique []process.Process

	for _, proc := range processes {
		if !seen[proc.PID] {
			seen[proc.PID] = true
			unique = append(unique, proc)
		}
	}

	return unique
}

func killMultipleProcesses(pm *process.ProcessManager, processes []process.Process) {
	if len(processes) == 0 {
		color.Yellow("No processes to kill")
		return
	}

	// Show what will be killed
	color.Cyan("Found %d process(es) to kill:", len(processes))
	for i, proc := range processes {
		uptime := ""
		if !proc.StartTime.IsZero() {
			uptime = fmt.Sprintf(" (uptime: %s)", time.Since(proc.StartTime).Round(time.Second))
		}
		fmt.Printf("  %d. PID %d: %s on port %d [%s]%s\n",
			i+1, proc.PID, proc.Command, proc.Port, proc.ServiceType, uptime)
	}
	fmt.Println()

	if !killYes {
		prompt := fmt.Sprintf("Are you sure you want to kill %d process(es)?", len(processes))
		if killForce {
			prompt = fmt.Sprintf("Are you sure you want to FORCE KILL %d process(es)?", len(processes))
		}

		if !confirmKill(prompt) {
			color.Yellow("Operation cancelled")
			return
		}
	}

	// Kill processes
	color.Yellow("Killing %d process(es)...", len(processes))

	pids := make([]int, len(processes))
	for i, proc := range processes {
		pids[i] = proc.PID
	}

	results := pm.KillProcesses(pids, killForce)

	// Report results
	var succeeded, failed []int
	for pid, err := range results {
		if err == nil {
			succeeded = append(succeeded, pid)
		} else {
			failed = append(failed, pid)
			color.Red("  Failed to kill PID %d: %v", pid, err)
		}
	}

	// Summary
	if len(succeeded) > 0 {
		color.Green("✅ Successfully killed %d process(es): %v", len(succeeded), succeeded)
	}

	if len(failed) > 0 {
		color.Red("❌ Failed to kill %d process(es): %v", len(failed), failed)
		color.Yellow("Tip: Try using --force or run with elevated privileges")
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(killCmd)

	killCmd.Flags().IntVarP(&killPID, "pid", "p", 0,
		"Kill process by PID instead of port")
	killCmd.Flags().BoolVarP(&killForce, "force", "f", false,
		"Force kill (SIGKILL on Unix, /F on Windows)")
	killCmd.Flags().BoolVarP(&killYes, "yes", "y", false,
		"Skip confirmation prompt")
	killCmd.Flags().StringVarP(&killRange, "range", "r", "",
		"Kill processes in port range (e.g., '3000-3010')")
	killCmd.Flags().StringVarP(&killService, "service", "s", "",
		"Kill processes by service type or command name")
	killCmd.Flags().StringVarP(&killUser, "user", "u", "",
		"Kill processes owned by specific user")
	killCmd.Flags().StringVar(&killOlder, "older", "",
		"Kill processes older than duration (e.g., '1h', '30m', '2h30m')")
}
