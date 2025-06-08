package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/gen2brain/beeep"
	tablepretty "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	process "dagger/portctl/pkg"
)

var (
	watchInterval   time.Duration
	watchNotify     bool
	watchChanges    bool
	watchContinuous bool
	watchCount      int
)

var watchCmd = &cobra.Command{
	Use:   "watch [port]",
	Short: "Watch processes on ports in real-time",
	Long: `Watch processes on ports with real-time updates and notifications.

Features:
  • Real-time monitoring with configurable refresh intervals
  • Desktop notifications when processes start/stop
  • Change detection with highlighting
  • Filter by specific port or monitor all ports
  • Continuous monitoring until interrupted

Examples:
  portctl watch                    # Watch all processes
  portctl watch 8080               # Watch specific port
  portctl watch --interval 2s     # Update every 2 seconds
  portctl watch --notify           # Send desktop notifications
  portctl watch --changes-only     # Only show when changes occur
`,
	Args: cobra.MaximumNArgs(1),
	Run:  runWatch,
}

type watchState struct {
	processes    map[string]process.Process
	lastUpdate   time.Time
	changes      []string
	totalUpdates int
}

func runWatch(cmd *cobra.Command, args []string) {
	// Parse port if provided
	targetPort := 0
	if len(args) > 0 {
		var err error
		targetPort, err = strconv.Atoi(args[0])
		if err != nil {
			color.Red("Invalid port number: %s", args[0])
			os.Exit(1)
		}
	}

	pm := process.NewProcessManager()
	state := &watchState{
		processes: make(map[string]process.Process),
	}

	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if err := s.Color("cyan"); err != nil {
		color.Red("Spinner color error: %v", err)
	}
	s.Prefix = "🔍 Watching "
	if targetPort > 0 {
		s.Suffix = fmt.Sprintf(" port %d ", targetPort)
	} else {
		s.Suffix = " all ports "
	}

	// Clear screen initially
	fmt.Print("\033[2J\033[H")

	// Initial load
	if err := updateProcesses(pm, state, targetPort, false); err != nil {
		color.Red("Error loading initial processes: %v", err)
		os.Exit(1)
	}

	// Print header
	printWatchHeader(targetPort, state)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	updateCycles := 0

	go func() {
		for {
			select {
			case <-ticker.C:
				if !watchContinuous {
					s.Start()
				}

				if err := updateProcesses(pm, state, targetPort, true); err != nil {
					if !watchContinuous {
						s.Stop()
					}
					color.Red("\nError updating processes: %v", err)
					continue
				}

				if !watchContinuous {
					s.Stop()
				}

				// Only print if we have changes or not in changes-only mode
				if !watchChanges || len(state.changes) > 0 {
					// Clear screen and reprint
					fmt.Print("\033[2J\033[H")
					printWatchHeader(targetPort, state)
					printProcesses(state)

					if len(state.changes) > 0 {
						printChanges(state)

						// Send notification if enabled
						if watchNotify {
							sendNotification(state.changes, targetPort)
						}
					}
				}

				updateCycles++
				if watchCount > 0 && updateCycles >= watchCount {
					if !watchContinuous {
						s.Stop()
					}
					color.Green("\n👋 Watch stopped after %d updates.", updateCycles)
					os.Exit(0)
				}

			case <-c:
				if !watchContinuous {
					s.Stop()
				}
				color.Green("\n👋 Watch stopped. Total updates: %d", state.totalUpdates)
				os.Exit(0)
			}
		}
	}()

	if watchContinuous {
		// Print initial table
		printProcesses(state)
	} else {
		s.Start()
	}

	// Wait for signal
	<-c
	if !watchContinuous {
		s.Stop()
	}
	color.Green("\n👋 Watch stopped. Total updates: %d", state.totalUpdates)
}

func updateProcesses(pm *process.ProcessManager, state *watchState, targetPort int, detectChanges bool) error {
	var processes []process.Process
	var err error

	if targetPort > 0 {
		processes, err = pm.GetProcessesOnPort(targetPort)
	} else {
		processes, err = pm.GetAllProcesses()
	}

	if err != nil {
		return err
	}

	// Detect changes if this is an update
	if detectChanges {
		state.changes = detectProcessChanges(state.processes, processes)
		state.totalUpdates++
	}

	// Update state
	newProcessMap := make(map[string]process.Process)
	for _, proc := range processes {
		key := fmt.Sprintf("%d:%d", proc.PID, proc.Port)
		newProcessMap[key] = proc
	}
	state.processes = newProcessMap
	state.lastUpdate = time.Now()

	return nil
}

func detectProcessChanges(oldProcs map[string]process.Process, newProcs []process.Process) []string {
	var changes []string

	// Create new process map
	newProcMap := make(map[string]process.Process)
	for _, proc := range newProcs {
		key := fmt.Sprintf("%d:%d", proc.PID, proc.Port)
		newProcMap[key] = proc
	}

	// Check for new processes
	for key, proc := range newProcMap {
		if _, exists := oldProcs[key]; !exists {
			changes = append(changes, fmt.Sprintf("➕ NEW: %s (PID %d) on port %d",
				proc.Command, proc.PID, proc.Port))
		}
	}

	// Check for removed processes
	for key, proc := range oldProcs {
		if _, exists := newProcMap[key]; !exists {
			changes = append(changes, fmt.Sprintf("➖ GONE: %s (PID %d) from port %d",
				proc.Command, proc.PID, proc.Port))
		}
	}

	return changes
}

func printWatchHeader(targetPort int, state *watchState) {
	// Title
	title := "🔍 portctl Watch Mode"
	if targetPort > 0 {
		title += fmt.Sprintf(" - Port %d", targetPort)
	}
	color.Cyan(title)

	// Status line
	status := fmt.Sprintf("Last Update: %s | Processes: %d | Updates: %d",
		state.lastUpdate.Format("15:04:05"),
		len(state.processes),
		state.totalUpdates)

	if watchInterval > 0 {
		status += fmt.Sprintf(" | Interval: %s", watchInterval)
	}

	color.White(status)
	fmt.Println(strings.Repeat("─", 80))
}

func printProcesses(state *watchState) {
	if len(state.processes) == 0 {
		fmt.Printf("\033[93mNo processes found\033[0m\n")
		return
	}

	// Convert map to slice and sort
	processes := make([]process.Process, 0, len(state.processes))
	for _, proc := range state.processes {
		processes = append(processes, proc)
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Port < processes[j].Port
	})

	t := tablepretty.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tablepretty.StyleColoredBright)
	t.AppendHeader(tablepretty.Row{"PID", "Port", "Protocol", "Service", "Command", "CPU%", "Mem(MB)", "User"})
	t.Style().Color.Header = text.Colors{text.FgHiBlue, text.Bold}
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
}

func printChanges(state *watchState) {
	if len(state.changes) == 0 {
		return
	}

	fmt.Println("\n📊 Changes Detected:")
	for _, change := range state.changes {
		if strings.Contains(change, "NEW") {
			color.Green("  %s", change)
		} else {
			color.Red("  %s", change)
		}
	}
}

func sendNotification(changes []string, targetPort int) {
	if len(changes) == 0 {
		return
	}

	title := "portctl - Process Changes"
	if targetPort > 0 {
		title += fmt.Sprintf(" (Port %d)", targetPort)
	}

	message := fmt.Sprintf("%d process changes detected", len(changes))
	if len(changes) <= 3 {
		message = strings.Join(changes, "\n")
	}

	// Send desktop notification
	_ = beeep.Notify(title, message, "")
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().DurationVarP(&watchInterval, "interval", "i", 3*time.Second,
		"Refresh interval (e.g., 1s, 500ms, 2m)")
	watchCmd.Flags().BoolVarP(&watchNotify, "notify", "n", false,
		"Send desktop notifications on changes")
	watchCmd.Flags().BoolVarP(&watchChanges, "changes-only", "c", false,
		"Only display output when changes are detected")
	watchCmd.Flags().BoolVar(&watchContinuous, "continuous", false,
		"Continuous output without clearing screen")
	watchCmd.Flags().IntVar(&watchCount, "count", 0,
		"Number of update cycles before exiting (default: unlimited)")
}
