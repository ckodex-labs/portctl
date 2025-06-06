package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	process "dagger/portctl/pkg"
)

type sessionState int

const (
	stateLoading sessionState = iota
	stateList
	stateFilter
	stateDetails
	stateKillConfirm
	stateStats
)

type tuiModel struct {
	state         sessionState
	processes     []process.Process
	filteredProcs []process.Process
	list          list.Model
	spinner       spinner.Model
	textInput     textinput.Model
	selectedProc  process.Process
	stats         *process.SystemStats
	pm            *process.ProcessManager
	err           error
	width         int
	height        int
	filterQuery   string
	showHelp      bool
	lastUpdate    time.Time
}

type processItem struct {
	process.Process
}

func (i processItem) FilterValue() string {
	return fmt.Sprintf("%d %s %s %s", i.Port, i.Command, i.ServiceType, i.User)
}

func (i processItem) Title() string {
	return fmt.Sprintf("Port %d", i.Port)
}

func (i processItem) Description() string {
	memStr := fmt.Sprintf("%.1fMB", i.MemoryMB)
	cpuStr := fmt.Sprintf("%.1f%%", i.CPUPercent)
	return fmt.Sprintf("%s • %s • %s • %s", i.Command, i.ServiceType, memStr, cpuStr)
}

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF7CCB")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8700"))
)

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch interactive TUI mode",
	Long: `Launch an interactive terminal user interface for browsing and managing processes.

Features:
  • Browse all processes with arrow keys
  • Filter processes in real-time
  • View detailed process information
  • Kill processes with confirmation
  • Real-time system statistics
  • Keyboard shortcuts for quick actions

Navigation:
  ↑/↓     Navigate process list
  /       Enter filter mode
  Enter   View process details
  k       Kill selected process
  s       Show system statistics
  r       Refresh process list
  q       Quit`,
	Aliases: []string{"tui", "ui", "i"},
	Run:     runInteractive,
}

func runInteractive(cmd *cobra.Command, args []string) {
	pm := process.NewProcessManager()

	m := tuiModel{
		state:      stateLoading,
		pm:         pm,
		lastUpdate: time.Now(),
	}

	// Initialize spinner
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize text input for filtering
	m.textInput = textinput.New()
	m.textInput.Placeholder = "Filter processes..."
	m.textInput.CharLimit = 50

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadProcesses(m.pm))
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 6) // Leave space for header and status

	case tea.KeyMsg:
		switch m.state {
		case stateLoading:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

		case stateList:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "/":
				m.state = stateFilter
				m.textInput.Focus()
				return m, textinput.Blink
			case "enter":
				if len(m.filteredProcs) > 0 {
					m.selectedProc = m.filteredProcs[m.list.Index()]
					m.state = stateDetails
				}
				return m, nil
			case "k":
				if len(m.filteredProcs) > 0 {
					m.selectedProc = m.filteredProcs[m.list.Index()]
					m.state = stateKillConfirm
				}
				return m, nil
			case "s":
				m.state = stateStats
				cmds = append(cmds, loadStats(m.pm))
			case "r":
				m.state = stateLoading
				cmds = append(cmds, loadProcesses(m.pm))
			case "h", "?":
				m.showHelp = !m.showHelp
			}

		case stateFilter:
			switch msg.String() {
			case "esc":
				m.state = stateList
				m.textInput.Blur()
				m.filterQuery = ""
				m.textInput.SetValue("")
				m.updateFilteredList()
				return m, nil
			case "enter":
				m.state = stateList
				m.textInput.Blur()
				m.filterQuery = m.textInput.Value()
				m.updateFilteredList()
				return m, nil
			}

		case stateDetails, stateKillConfirm, stateStats:
			switch msg.String() {
			case "esc", "q":
				m.state = stateList
				return m, nil
			case "y":
				if m.state == stateKillConfirm {
					cmds = append(cmds, killProcess(m.pm, m.selectedProc.PID))
					m.state = stateLoading
					cmds = append(cmds, loadProcesses(m.pm))
				}
			case "n":
				if m.state == stateKillConfirm {
					m.state = stateList
				}
			}
		}

	case processesLoadedMsg:
		m.processes = msg.processes
		m.err = msg.err
		if m.err == nil {
			m.updateFilteredList()
			m.state = stateList
			m.lastUpdate = time.Now()
		}

	case statsLoadedMsg:
		m.stats = msg.stats
		m.err = msg.err

	case processKilledMsg:
		// Process killed, reload list
		cmds = append(cmds, loadProcesses(m.pm))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update list and text input
	if m.state == stateFilter {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content strings.Builder

	// Header
	header := titleStyle.Render("🚀 portctl Interactive Mode")
	if m.lastUpdate.IsZero() {
		header += statusStyle.Render(" • Loading...")
	} else {
		header += statusStyle.Render(fmt.Sprintf(" • %d processes • Last updated: %s",
			len(m.processes), m.lastUpdate.Format("15:04:05")))
	}
	content.WriteString(header + "\n\n")

	// Handle error state
	if m.err != nil {
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		content.WriteString("\n\n" + helpStyle.Render("Press 'q' to quit, 'r' to retry"))
		return content.String()
	}

	switch m.state {
	case stateLoading:
		content.WriteString(m.spinner.View() + " Loading processes...")

	case stateList:
		if m.showHelp {
			content.WriteString(m.renderHelp())
		} else {
			content.WriteString(m.list.View())
		}

	case stateFilter:
		content.WriteString("Filter processes:\n")
		content.WriteString(m.textInput.View() + "\n\n")
		content.WriteString(helpStyle.Render("Press Enter to apply filter, Esc to cancel"))

	case stateDetails:
		content.WriteString(m.renderProcessDetails())

	case stateKillConfirm:
		content.WriteString(m.renderKillConfirm())

	case stateStats:
		content.WriteString(m.renderStats())
	}

	// Footer with shortcuts (except in filter mode)
	if m.state != stateFilter && m.state != stateLoading {
		footer := "\n" + helpStyle.Render("Press 'h' for help, 'q' to quit")
		content.WriteString(footer)
	}

	return content.String()
}

func (m *tuiModel) updateFilteredList() {
	if m.filterQuery == "" {
		m.filteredProcs = m.processes
	} else {
		m.filteredProcs = nil
		query := strings.ToLower(m.filterQuery)
		for _, proc := range m.processes {
			if strings.Contains(strings.ToLower(proc.Command), query) ||
				strings.Contains(strings.ToLower(proc.ServiceType), query) ||
				strings.Contains(strings.ToLower(proc.User), query) ||
				strings.Contains(strconv.Itoa(proc.Port), query) {
				m.filteredProcs = append(m.filteredProcs, proc)
			}
		}
	}

	// Update list items
	items := make([]list.Item, len(m.filteredProcs))
	for i, proc := range m.filteredProcs {
		items[i] = processItem{proc}
	}

	m.list.SetItems(items)
}

func (m tuiModel) renderHelp() string {
	var help strings.Builder
	help.WriteString(highlightStyle.Render("Keyboard Shortcuts:") + "\n\n")
	help.WriteString("  ↑/↓        Navigate process list\n")
	help.WriteString("  /          Filter processes\n")
	help.WriteString("  Enter      View process details\n")
	help.WriteString("  k          Kill selected process\n")
	help.WriteString("  s          Show system statistics\n")
	help.WriteString("  r          Refresh process list\n")
	help.WriteString("  h/?        Toggle this help\n")
	help.WriteString("  q          Quit\n\n")
	help.WriteString(helpStyle.Render("Press 'h' again to hide this help"))
	return help.String()
}

func (m tuiModel) renderProcessDetails() string {
	proc := m.selectedProc
	var details strings.Builder

	details.WriteString(highlightStyle.Render(fmt.Sprintf("Process Details - PID %d", proc.PID)) + "\n\n")

	details.WriteString(fmt.Sprintf("Command:      %s\n", proc.Command))
	details.WriteString(fmt.Sprintf("Full Command: %s\n", proc.FullCommand))
	details.WriteString(fmt.Sprintf("Port:         %d (%s)\n", proc.Port, proc.Protocol))
	details.WriteString(fmt.Sprintf("Service Type: %s\n", proc.ServiceType))
	details.WriteString(fmt.Sprintf("User:         %s\n", proc.User))
	details.WriteString(fmt.Sprintf("State:        %s\n", proc.State))
	details.WriteString(fmt.Sprintf("Local Addr:   %s\n", proc.LocalAddr))
	details.WriteString(fmt.Sprintf("Remote Addr:  %s\n", proc.RemoteAddr))
	details.WriteString(fmt.Sprintf("CPU Usage:    %.1f%%\n", proc.CPUPercent))
	details.WriteString(fmt.Sprintf("Memory:       %.1f MB\n", proc.MemoryMB))

	if !proc.StartTime.IsZero() {
		details.WriteString(fmt.Sprintf("Started:      %s\n", proc.StartTime.Format("2006-01-02 15:04:05")))
		details.WriteString(fmt.Sprintf("Uptime:       %s\n", time.Since(proc.StartTime).Round(time.Second)))
	}

	details.WriteString("\n" + helpStyle.Render("Press Esc to go back, 'k' to kill this process"))
	return details.String()
}

func (m tuiModel) renderKillConfirm() string {
	proc := m.selectedProc
	var confirm strings.Builder

	confirm.WriteString(warningStyle.Render("⚠️  Kill Process Confirmation") + "\n\n")
	confirm.WriteString("Are you sure you want to kill:\n")
	confirm.WriteString(fmt.Sprintf("  PID:     %d\n", proc.PID))
	confirm.WriteString(fmt.Sprintf("  Command: %s\n", proc.Command))
	confirm.WriteString(fmt.Sprintf("  Port:    %d\n", proc.Port))
	confirm.WriteString(fmt.Sprintf("  Service: %s\n", proc.ServiceType))
	confirm.WriteString("\n")
	confirm.WriteString(errorStyle.Render("Press 'y' to kill, 'n' to cancel"))

	return confirm.String()
}

func (m tuiModel) renderStats() string {
	if m.stats == nil {
		return m.spinner.View() + " Loading system statistics..."
	}

	var stats strings.Builder
	stats.WriteString(highlightStyle.Render("📊 System Statistics") + "\n\n")

	stats.WriteString(fmt.Sprintf("Total Processes:    %s\n",
		infoStyle.Render(strconv.Itoa(m.stats.TotalProcesses))))
	stats.WriteString(fmt.Sprintf("Listening Ports:    %s\n",
		infoStyle.Render(strconv.Itoa(m.stats.ListeningPorts))))
	stats.WriteString(fmt.Sprintf("CPU Usage:          %s\n",
		infoStyle.Render(fmt.Sprintf("%.1f%%", m.stats.CPUUsagePercent))))
	stats.WriteString(fmt.Sprintf("Memory Used:        %s\n",
		infoStyle.Render(fmt.Sprintf("%.1f GB", m.stats.MemoryUsageGB))))
	stats.WriteString(fmt.Sprintf("Memory Available:   %s\n",
		infoStyle.Render(fmt.Sprintf("%.1f GB", m.stats.AvailableMemoryGB))))

	if len(m.stats.TopPortUsers) > 0 {
		stats.WriteString("\n" + highlightStyle.Render("Top Memory Users:") + "\n")
		for i, proc := range m.stats.TopPortUsers {
			if i >= 5 {
				break
			}
			stats.WriteString(fmt.Sprintf("  %d. %s (Port %d) - %.1f MB\n",
				i+1, proc.Command, proc.Port, proc.MemoryMB))
		}
	}

	stats.WriteString("\n" + helpStyle.Render("Press Esc to go back"))
	return stats.String()
}

// Messages
type processesLoadedMsg struct {
	processes []process.Process
	err       error
}

type statsLoadedMsg struct {
	stats *process.SystemStats
	err   error
}

type processKilledMsg struct {
	pid int
	err error
}

// Commands
func loadProcesses(pm *process.ProcessManager) tea.Cmd {
	return func() tea.Msg {
		processes, err := pm.GetAllProcesses()
		return processesLoadedMsg{processes: processes, err: err}
	}
}

func loadStats(pm *process.ProcessManager) tea.Cmd {
	return func() tea.Msg {
		stats, err := pm.GetSystemStats()
		return statsLoadedMsg{stats: stats, err: err}
	}
}

func killProcess(pm *process.ProcessManager, pid int) tea.Cmd {
	return func() tea.Msg {
		err := pm.KillProcess(pid, false)
		return processKilledMsg{pid: pid, err: err}
	}
}

func init() {
	rootCmd.AddCommand(interactiveCmd)

	// Configure list delegate
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FF7CCB")).
		BorderLeftForeground(lipgloss.Color("#FF7CCB"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#AD58B4"))

	// This will be set properly when the model is initialized
	items := []list.Item{}
	tuiList := list.New(items, delegate, 0, 0)
	tuiList.Title = ""
	tuiList.SetShowStatusBar(false)
	tuiList.SetFilteringEnabled(false) // We'll handle filtering ourselves
	tuiList.SetShowHelp(false)
}
