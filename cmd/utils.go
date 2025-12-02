package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	tablepretty "github.com/jedib0t/go-pretty/v6/table"
	text "github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	process "dagger/portctl/pkg"
)

var (
	availableStart int
	availableEnd   int
	availableCount int
)

var availableCmd = &cobra.Command{
	Use:   "available",
	Short: "Find available ports in specified ranges",
	Long: `Find available ports that are not currently in use.

This command helps you quickly find free ports for development or testing.
You can specify custom port ranges or use common development port ranges.

Examples:
  portctl available                    # Find 10 ports in development range (3000-9999)
  portctl available --start 8000      # Find ports starting from 8000
  portctl available --end 8100        # Find ports up to 8100
  portctl available --count 5         # Find only 5 available ports
  portctl available --start 3000 --end 4000 --count 20  # Custom range`,
	Aliases: []string{"free", "open"},
	Run:     runAvailable,
}

func runAvailable(cmd *cobra.Command, args []string) {
	pm := process.NewProcessManager()
	ctx := cmd.Context()

	// Set defaults if not specified
	if availableStart == 0 {
		availableStart = 3000
	}
	if availableEnd == 0 {
		availableEnd = 9999
	}
	if availableCount == 0 {
		availableCount = 10
	}

	// Validate range
	if availableStart >= availableEnd {
		fmt.Println("\033[91mStart port must be less than end port\033[0m")
		os.Exit(1)
	}

	fmt.Printf("\033[96m🔍 Searching for available ports in range %d-%d...\033[0m\n", availableStart, availableEnd)

	available, err := pm.FindAvailablePorts(ctx, availableStart, availableEnd, availableCount)
	if err != nil {
		fmt.Printf("\033[91mError finding available ports: %v\033[0m\n", err)
		os.Exit(1)
	}

	if len(available) == 0 {
		fmt.Printf("\033[93mNo available ports found in range %d-%d\033[0m\n", availableStart, availableEnd)
		return
	}

	fmt.Printf("\033[92m✅ Found %d available port(s):\033[0m\n\n", len(available))

	// Create table
	t := tablepretty.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tablepretty.StyleColoredBright)
	t.AppendHeader(tablepretty.Row{"Port", "Suggested Use", "Common Service"})
	t.Style().Color.Header = text.Colors{text.FgHiBlue, text.Bold}
	t.SetColumnConfigs([]tablepretty.ColumnConfig{
		{Number: 1, Align: text.AlignRight, Colors: text.Colors{text.FgCyan, text.Bold}}, // Port
		{Number: 2, Align: text.AlignLeft},                                               // Suggested Use
		{Number: 3, Align: text.AlignLeft, Colors: text.Colors{text.FgYellow}},           // Common Service
	})

	for _, port := range available {
		suggestedUse := getSuggestedUse(port)
		commonService := getCommonService(port)
		row := tablepretty.Row{
			port,
			suggestedUse,
			commonService,
		}
		t.AppendRow(row)
	}
	t.Render()

	// Show quick copy commands
	fmt.Println()
	fmt.Printf("\033[96m💡 Quick commands:\033[0m\n")
	if len(available) > 0 {
		fmt.Printf("  export PORT=%d\n", available[0])
		fmt.Printf("  npm start -- --port %d\n", available[0])
		fmt.Printf("  python -m http.server %d\n", available[0])
	}
}

func getSuggestedUse(port int) string {
	switch {
	case port >= 3000 && port <= 3999:
		return "Development server"
	case port >= 4000 && port <= 4999:
		return "Local services"
	case port >= 5000 && port <= 5999:
		return "Development/Testing"
	case port >= 8000 && port <= 8999:
		return "Web servers/APIs"
	case port >= 9000 && port <= 9999:
		return "Microservices"
	default:
		return "General purpose"
	}
}

func getCommonService(port int) string {
	return process.GetServiceName(port)
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show comprehensive system and port statistics",
	Long: `Display detailed statistics about system resources and port usage.

This command provides insights into:
  • System resource usage (CPU, memory)
  • Total processes and listening ports
  • Top processes by resource usage
  • Port distribution by service type
  • Common development ports status

Examples:
  portctl stats           # Show all statistics
  portctl stats --json   # Output in JSON format`,
	Aliases: []string{"statistics", "info", "system"},
	Run:     runStats,
}

var statsJSON bool

func runStats(cmd *cobra.Command, args []string) {
	pm := process.NewProcessManager()
	ctx := cmd.Context()

	fmt.Printf("\033[96m📊 Gathering system statistics...\033[0m\n")

	stats, err := pm.GetSystemStats(ctx)
	if err != nil {
		fmt.Printf("\033[91mError getting system statistics: %v\033[0m\n", err)
		os.Exit(1)
	}

	if statsJSON {
		// Output JSON
		fmt.Printf(`{
  "total_processes": %d,
  "listening_ports": %d,
  "cpu_usage_percent": %.1f,
  "memory_usage_gb": %.1f,
  "available_memory_gb": %.1f,
  "top_port_users": [`,
			stats.TotalProcesses,
			stats.ListeningPorts,
			stats.CPUUsagePercent,
			stats.MemoryUsageGB,
			stats.AvailableMemoryGB)

		for i, proc := range stats.TopPortUsers {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`
    {
      "pid": %d,
      "port": %d,
      "command": "%s",
      "service_type": "%s",
      "memory_mb": %.1f,
      "cpu_percent": %.1f
    }`, proc.PID, proc.Port, proc.Command, proc.ServiceType, proc.MemoryMB, proc.CPUPercent)
		}
		fmt.Println(`
  ]
}`)
		return
	}

	// Pretty output
	fmt.Print("\033[2J\033[H") // Clear screen

	fmt.Printf("\033[92m🚀 portctl System Statistics\033[0m\n")
	fmt.Println(strings.Repeat("═", 50))

	// System overview
	fmt.Printf("\033[96m📈 System Overview:\033[0m\n")
	fmt.Printf("  Total Processes:    %d\n", stats.TotalProcesses)
	fmt.Printf("  Listening Ports:    %d\n", stats.ListeningPorts)
	fmt.Printf("  CPU Usage:          %.1f%%\n", stats.CPUUsagePercent)
	fmt.Printf("  Memory Used:        %.1f GB\n", stats.MemoryUsageGB)
	fmt.Printf("  Memory Available:   %.1f GB\n", stats.AvailableMemoryGB)

	// Memory usage bar - prevent division by zero
	totalMemory := stats.MemoryUsageGB + stats.AvailableMemoryGB
	memoryPercent := 0.0
	if totalMemory > 0 {
		memoryPercent = (stats.MemoryUsageGB / totalMemory) * 100
	}
	fmt.Printf("  Memory Usage:       %s (%.1f%%)\n",
		getProgressBar(memoryPercent), memoryPercent)

	// Top processes
	if len(stats.TopPortUsers) > 0 {
		fmt.Printf("\033[96m🔥 Top Memory Users:\033[0m\n")
		t := tablepretty.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(tablepretty.StyleColoredBright)
		t.AppendHeader(tablepretty.Row{"Rank", "PID", "Port", "Command", "Service", "Memory", "CPU%"})
		t.Style().Color.Header = text.Colors{text.FgHiBlue, text.Bold}
		t.SetColumnConfigs([]tablepretty.ColumnConfig{
			{Number: 1, Align: text.AlignRight, Colors: text.Colors{text.FgCyan, text.Bold}}, // Rank
			{Number: 2, Align: text.AlignRight},                                              // PID
			{Number: 3, Align: text.AlignRight},                                              // Port
			{Number: 4, Align: text.AlignLeft},                                               // Command
			{Number: 5, Align: text.AlignLeft},                                               // Service
			{Number: 6, Align: text.AlignRight, Colors: text.Colors{text.FgYellow}},          // Memory
			{Number: 7, Align: text.AlignRight},                                              // CPU%
		})

		for i, proc := range stats.TopPortUsers {
			if i >= 5 {
				break
			}
			row := tablepretty.Row{
				fmt.Sprintf("#%d", i+1),
				proc.PID,
				proc.Port,
				proc.Command,
				proc.ServiceType,
				fmt.Sprintf("%.1f MB", proc.MemoryMB),
				fmt.Sprintf("%.1f", proc.CPUPercent),
			}
			t.AppendRow(row)
		}
		t.Render()
	}

	// Development ports status
	fmt.Printf("\033[96m🛠️  Common Development Ports:\033[0m\n")
	checkCommonPorts(ctx, pm)
}

func getProgressBar(percent float64) string {
	width := 20
	filled := int((percent / 100) * float64(width))

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < width; i++ {
		if i < filled {
			if percent > 80 {
				bar.WriteString("\033[91m█\033[0m")
			} else if percent > 60 {
				bar.WriteString("\033[93m█\033[0m")
			} else {
				bar.WriteString("\033[92m█\033[0m")
			}
		} else {
			bar.WriteString("░")
		}
	}

	bar.WriteString("]")
	return bar.String()
}

func checkCommonPorts(ctx context.Context, pm *process.ProcessManager) {
	commonPorts := []int{3000, 3001, 4000, 5000, 8000, 8080, 8081, 9000}

	t := tablepretty.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tablepretty.StyleColoredBright)
	t.AppendHeader(tablepretty.Row{"Port", "Status", "Process", "Service"})
	t.Style().Color.Header = text.Colors{text.FgHiBlue, text.Bold}
	t.SetColumnConfigs([]tablepretty.ColumnConfig{
		{Number: 1, Align: text.AlignRight, Colors: text.Colors{text.FgCyan, text.Bold}}, // Port
		{Number: 2, Align: text.AlignCenter},                                             // Status
		{Number: 3, Align: text.AlignLeft},                                               // Process
		{Number: 4, Align: text.AlignLeft, Colors: text.Colors{text.FgYellow}},           // Service
	})

	for _, port := range commonPorts {
		processes, _ := pm.GetProcessesOnPort(ctx, port)
		status := ""
		if len(processes) > 0 {
			proc := processes[0]
			status = text.FgRed.Sprint("IN USE")
			row := tablepretty.Row{
				port,
				status,
				proc.Command,
				proc.ServiceType,
			}
			t.AppendRow(row)
		} else {
			status = text.FgGreen.Sprint("AVAILABLE")
			row := tablepretty.Row{
				port,
				status,
				"-",
				"-",
			}
			t.AppendRow(row)
		}
	}
	t.Render()
}

func init() {
	rootCmd.AddCommand(availableCmd)
	rootCmd.AddCommand(statsCmd)

	// Available command flags
	availableCmd.Flags().IntVarP(&availableStart, "start", "s", 0,
		"Start of port range (default: 3000)")
	availableCmd.Flags().IntVarP(&availableEnd, "end", "e", 0,
		"End of port range (default: 9999)")
	availableCmd.Flags().IntVarP(&availableCount, "count", "c", 0,
		"Number of ports to find (default: 10)")

	// Stats command flags
	statsCmd.Flags().BoolVarP(&statsJSON, "json", "j", false,
		"Output statistics in JSON format")
}
