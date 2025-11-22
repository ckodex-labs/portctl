package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	tablepretty "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	process "dagger/portctl/pkg"
)

var (
	listJSON     bool
	listAll      bool
	listService  string
	listUser     string
	listSort     string
	listTree     bool
	listDetails  bool
	listMemLimit float64
	listCPULimit float64
)

var listCmd = &cobra.Command{
	Use:   "list [port]",
	Short: "List processes running on specific ports with advanced filtering",
	Long: `List processes that are currently listening on specified ports.
Supports advanced filtering, sorting, and display options.

Examples:
  # Basic usage
  portctl list 8080              # List processes on port 8080
  portctl list                   # List all processes with open ports
  
  # Filtering
  portctl list --service node    # Filter by service type
  portctl list --user john       # Filter by user
  portctl list --mem-limit 100   # Show processes using >100MB memory
  portctl list --cpu-limit 50    # Show processes using >50% CPU
  
  # Output options
  portctl list --json            # Output in JSON format
  portctl list --details         # Show detailed information
  portctl list --sort port       # Sort by port (port, pid, cpu, memory, command)
  portctl list --tree            # Show process relationships`,
	Args: cobra.MaximumNArgs(1),
	Run:  runList,
}

func runList(cmd *cobra.Command, args []string) {
	pm := process.NewProcessManager()
	ctx := cmd.Context()

	var processes []process.Process
	var err error

	if len(args) == 0 || listAll {
		// List all processes
		processes, err = pm.GetAllProcesses(ctx)
		if err != nil {
			color.Red("Error getting processes: %v", err)
			os.Exit(1)
		}
	} else {
		// List processes on specific port
		port, err := strconv.Atoi(args[0])
		if err != nil {
			color.Red("Invalid port number: %s", args[0])
			os.Exit(1)
		}

		processes, err = pm.GetProcessesOnPort(ctx, port)
		if err != nil {
			color.Red("Error getting processes on port %d: %v", port, err)
			os.Exit(1)
		}
	}

	// Apply filters
	filterOpts := process.FilterOptions{
		Service:     listService,
		User:        listUser,
		MemoryLimit: listMemLimit,
		CPULimit:    listCPULimit,
	}
	processes = pm.FilterProcesses(processes, filterOpts)

	// Apply sorting
	processes = pm.SortProcesses(processes, listSort)

	if len(processes) == 0 {
		if len(args) > 0 {
			color.Yellow("No processes found on port %s matching filters", args[0])
		} else {
			color.Yellow("No processes found matching filters")
		}
		return
	}

	if listJSON {
		outputJSON(processes)
	} else if listDetails {
		outputDetailed(processes)
	} else if listTree {
		outputTree(processes)
	} else {
		outputTable(processes)
	}
}

func outputTable(processes []process.Process) {
	t := tablepretty.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tablepretty.StyleColoredBright)

	// Set header and header color
	t.AppendHeader(tablepretty.Row{"PID", "Port", "Protocol", "Service", "Command", "CPU%", "Mem(MB)", "User"})
	t.Style().Color.Header = text.Colors{text.FgHiBlue, text.Bold}

	// Set column configs for alignment and color
	t.SetColumnConfigs([]tablepretty.ColumnConfig{
		{Number: 1, Align: text.AlignRight},                                              // PID
		{Number: 2, Align: text.AlignRight, Colors: text.Colors{text.FgCyan, text.Bold}}, // Port
		{Number: 3, Align: text.AlignCenter},                                             // Protocol
		{Number: 4, Align: text.AlignCenter},                                             // Service
		{Number: 5, Align: text.AlignLeft},                                               // Command
		{Number: 6, Align: text.AlignRight},                                              // CPU%
		{Number: 7, Align: text.AlignRight},                                              // Mem(MB)
		{Number: 8, Align: text.AlignLeft},                                               // User
	})

	for _, proc := range processes {
		row := tablepretty.Row{
			proc.PID,
			proc.Port,
			proc.Protocol,
			proc.ServiceType,
			proc.Command,
			fmt.Sprintf("%.1f", proc.CPUPercent),
			fmt.Sprintf("%.1f", proc.MemoryMB),
			proc.User,
		}
		t.AppendRow(row)
	}

	t.Render()
	color.Green("\nFound %d process(es)", len(processes))
}

func outputDetailed(processes []process.Process) {
	for i, proc := range processes {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 50))
		}

		color.Cyan("Process #%d", i+1)
		fmt.Printf("  PID:           %d\n", proc.PID)
		fmt.Printf("  Port:          %d (%s)\n", proc.Port, proc.Protocol)
		fmt.Printf("  Command:       %s\n", proc.Command)
		fmt.Printf("  Full Command:  %s\n", proc.FullCommand)
		fmt.Printf("  Service Type:  %s\n", proc.ServiceType)
		fmt.Printf("  User:          %s\n", proc.User)
		fmt.Printf("  State:         %s\n", proc.State)
		fmt.Printf("  Local Addr:    %s\n", proc.LocalAddr)
		fmt.Printf("  Remote Addr:   %s\n", proc.RemoteAddr)
		fmt.Printf("  CPU Usage:     %.1f%%\n", proc.CPUPercent)
		fmt.Printf("  Memory:        %.1f MB\n", proc.MemoryMB)

		if !proc.StartTime.IsZero() {
			fmt.Printf("  Started:       %s\n", proc.StartTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Uptime:        %s\n", time.Since(proc.StartTime).Round(time.Second))
		}
	}
}

func outputTree(processes []process.Process) {
	// Group processes by service type
	serviceGroups := make(map[string][]process.Process)
	for _, proc := range processes {
		serviceGroups[proc.ServiceType] = append(serviceGroups[proc.ServiceType], proc)
	}

	color.Cyan("📊 Process Tree by Service Type\n")

	for serviceType, procs := range serviceGroups {
		color.Yellow("├─ %s (%d processes)", serviceType, len(procs))

		for i, proc := range procs {
			symbol := "├─"
			if i == len(procs)-1 {
				symbol = "└─"
			}

			uptime := ""
			if !proc.StartTime.IsZero() {
				uptime = fmt.Sprintf(" [%s]", time.Since(proc.StartTime).Round(time.Second))
			}

			fmt.Printf("   %s PID %d: %s (Port %d) - %.1fMB%s\n",
				symbol, proc.PID, proc.Command, proc.Port, proc.MemoryMB, uptime)
		}
		fmt.Println()
	}
}

func outputJSON(processes []process.Process) {
	// Enhanced JSON output with all fields
	fmt.Println("[")
	for i, proc := range processes {
		fmt.Printf(`  {
    "pid": %d,
    "port": %d,
    "protocol": "%s",
    "state": "%s",
    "command": "%s",
    "full_command": "%s",
    "service_type": "%s",
    "user": "%s",
    "local_addr": "%s",
    "remote_addr": "%s",
    "cpu_percent": %.1f,
    "memory_mb": %.1f,
    "start_time": "%s"
  }`, proc.PID, proc.Port, proc.Protocol, proc.State, proc.Command,
			proc.FullCommand, proc.ServiceType, proc.User, proc.LocalAddr,
			proc.RemoteAddr, proc.CPUPercent, proc.MemoryMB, proc.StartTime.Format(time.RFC3339))

		if i < len(processes)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	fmt.Println("]")
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listJSON, "json", "j", false,
		"Output in JSON format")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false,
		"List all processes (same as not specifying a port)")
	listCmd.Flags().StringVarP(&listService, "service", "s", "",
		"Filter by service type or command name")
	listCmd.Flags().StringVarP(&listUser, "user", "u", "",
		"Filter by user")
	listCmd.Flags().StringVar(&listSort, "sort", "port",
		"Sort by field (port, pid, cpu, memory, command, service, user)")
	listCmd.Flags().BoolVarP(&listTree, "tree", "t", false,
		"Show process tree grouped by service type")
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false,
		"Show detailed information for each process")
	listCmd.Flags().Float64Var(&listMemLimit, "mem-limit", 0,
		"Show only processes using more than X MB of memory")
	listCmd.Flags().Float64Var(&listCPULimit, "cpu-limit", 0,
		"Show only processes using more than X% CPU")
}
