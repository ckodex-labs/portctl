package process

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// Process represents a process listening on a port with enhanced details
type Process struct {
	PID         int       `json:"pid"`
	Port        int       `json:"port"`
	Command     string    `json:"command"`
	Protocol    string    `json:"protocol"`
	State       string    `json:"state"`
	User        string    `json:"user"`
	StartTime   time.Time `json:"start_time"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemoryMB    float32   `json:"memory_mb"`
	ServiceType string    `json:"service_type"`
	FullCommand string    `json:"full_command"`
	LocalAddr   string    `json:"local_addr"`
	RemoteAddr  string    `json:"remote_addr"`
}

// SystemStats represents system-wide statistics
type SystemStats struct {
	TotalProcesses    int       `json:"total_processes"`
	ListeningPorts    int       `json:"listening_ports"`
	CPUUsagePercent   float64   `json:"cpu_usage_percent"`
	MemoryUsageGB     float64   `json:"memory_usage_gb"`
	AvailableMemoryGB float64   `json:"available_memory_gb"`
	TopPortUsers      []Process `json:"top_port_users"`
}

// FilterOptions defines criteria for filtering processes
type FilterOptions struct {
	Service     string
	User        string
	MemoryLimit float64
	CPULimit    float64
}

// ProcessManager handles process operations with enhanced features
type ProcessManager struct {
	enableMetrics bool
}

// NewProcessManager creates a new ProcessManager
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		enableMetrics: true,
	}
}

// GetProcessesOnPort returns all processes listening on the specified port with enhanced details
func (pm *ProcessManager) GetProcessesOnPort(ctx context.Context, port int) ([]Process, error) {
	processes, err := pm.getBasicProcesses(ctx, port)
	if err != nil {
		return nil, err
	}

	// Enhance with additional metrics
	return pm.enhanceProcesses(ctx, processes), nil
}

// GetAllProcesses returns all processes with open ports with enhanced details
func (pm *ProcessManager) GetAllProcesses(ctx context.Context) ([]Process, error) {
	processes, err := pm.getBasicProcesses(ctx, 0)
	if err != nil {
		return nil, err
	}

	// Enhance with additional metrics
	enhanced := pm.enhanceProcesses(ctx, processes)

	// Sort by port number
	sort.Slice(enhanced, func(i, j int) bool {
		return enhanced[i].Port < enhanced[j].Port
	})

	return enhanced, nil
}

// GetSystemStats returns comprehensive system statistics
func (pm *ProcessManager) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	processes, err := pm.GetAllProcesses(ctx)
	if err != nil {
		return nil, err
	}

	// Get CPU usage
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		cpuPercent = []float64{0}
	}

	// Get memory stats
	memStats, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get top port users (by memory usage)
	topUsers := make([]Process, len(processes))
	copy(topUsers, processes)
	sort.Slice(topUsers, func(i, j int) bool {
		return topUsers[i].MemoryMB > topUsers[j].MemoryMB
	})
	if len(topUsers) > 5 {
		topUsers = topUsers[:5]
	}

	return &SystemStats{
		TotalProcesses:    len(processes),
		ListeningPorts:    pm.countUniquePorts(processes),
		CPUUsagePercent:   cpuPercent[0],
		MemoryUsageGB:     float64(memStats.Used) / 1024 / 1024 / 1024,
		AvailableMemoryGB: float64(memStats.Available) / 1024 / 1024 / 1024,
		TopPortUsers:      topUsers,
	}, nil
}

// GetProcessesByService returns processes filtered by service type
func (pm *ProcessManager) GetProcessesByService(ctx context.Context, serviceType string) ([]Process, error) {
	processes, err := pm.GetAllProcesses(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []Process
	serviceType = strings.ToLower(serviceType)

	for _, proc := range processes {
		if strings.Contains(strings.ToLower(proc.ServiceType), serviceType) ||
			strings.Contains(strings.ToLower(proc.Command), serviceType) {
			filtered = append(filtered, proc)
		}
	}

	return filtered, nil
}

// FindAvailablePorts suggests available ports in common ranges
func (pm *ProcessManager) FindAvailablePorts(ctx context.Context, startPort, endPort int, count int) ([]int, error) {
	processes, err := pm.GetAllProcesses(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map of used ports
	usedPorts := make(map[int]bool)
	for _, proc := range processes {
		usedPorts[proc.Port] = true
	}

	var available []int
	for port := startPort; port <= endPort && len(available) < count; port++ {
		if !usedPorts[port] {
			available = append(available, port)
		}
	}

	return available, nil
}

// KillProcesses kills multiple processes by PID with enhanced error reporting
func (pm *ProcessManager) KillProcesses(ctx context.Context, pids []int, force bool) map[int]error {
	results := make(map[int]error)

	for _, pid := range pids {
		results[pid] = pm.KillProcess(ctx, pid, force)
	}

	return results
}

// KillProcess kills a process by PID
func (pm *ProcessManager) KillProcess(ctx context.Context, pid int, force bool) error {
	if runtime.GOOS == "windows" {
		var cmd *exec.Cmd
		if force {
			// #nosec G204: Arguments are constructed from validated integer pid, not user input
			cmd = exec.CommandContext(ctx, "taskkill", "/F", "/PID", strconv.Itoa(pid))
		} else {
			// #nosec G204: Arguments are constructed from validated integer pid, not user input
			cmd = exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(pid))
		}
		return cmd.Run()
	} else {
		// Unix-like systems
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process %d: %v", pid, err)
		}

		signal := syscall.SIGTERM
		if force {
			signal = syscall.SIGKILL
		}

		return process.Signal(signal)
	}
}

// FilterProcesses filters a list of processes based on options
func (pm *ProcessManager) FilterProcesses(processes []Process, opts FilterOptions) []Process {
	var filtered []Process

	for _, proc := range processes {
		match := true

		// Filter by service type
		if opts.Service != "" {
			if !strings.Contains(strings.ToLower(proc.ServiceType), strings.ToLower(opts.Service)) &&
				!strings.Contains(strings.ToLower(proc.Command), strings.ToLower(opts.Service)) {
				match = false
			}
		}

		// Filter by user
		if opts.User != "" {
			if !strings.Contains(strings.ToLower(proc.User), strings.ToLower(opts.User)) {
				match = false
			}
		}

		// Filter by memory usage
		if opts.MemoryLimit > 0 && proc.MemoryMB <= float32(opts.MemoryLimit) {
			match = false
		}

		// Filter by CPU usage
		if opts.CPULimit > 0 && proc.CPUPercent <= opts.CPULimit {
			match = false
		}

		if match {
			filtered = append(filtered, proc)
		}
	}

	return filtered
}

// SortProcesses sorts a list of processes by a given field
func (pm *ProcessManager) SortProcesses(processes []Process, sortBy string) []Process {
	if sortBy == "" {
		sortBy = "port" // Default sort by port
	}

	sort.Slice(processes, func(i, j int) bool {
		switch strings.ToLower(sortBy) {
		case "pid":
			return processes[i].PID < processes[j].PID
		case "port":
			return processes[i].Port < processes[j].Port
		case "cpu":
			return processes[i].CPUPercent > processes[j].CPUPercent // Descending
		case "memory", "mem":
			return processes[i].MemoryMB > processes[j].MemoryMB // Descending
		case "command", "cmd":
			return processes[i].Command < processes[j].Command
		case "service":
			return processes[i].ServiceType < processes[j].ServiceType
		case "user":
			return processes[i].User < processes[j].User
		default:
			return processes[i].Port < processes[j].Port
		}
	})

	return processes
}

// getBasicProcesses gets basic process information (original functionality)
func (pm *ProcessManager) getBasicProcesses(ctx context.Context, targetPort int) ([]Process, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		return pm.getProcessesUnix(ctx, targetPort)
	case "windows":
		return pm.getProcessesWindows(ctx, targetPort)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// enhanceProcesses adds detailed metrics to processes
func (pm *ProcessManager) enhanceProcesses(ctx context.Context, processes []Process) []Process {
	if !pm.enableMetrics {
		return processes
	}

	for i := range processes {
		pm.enhanceProcess(ctx, &processes[i])
	}

	return processes
}

// enhanceProcess adds detailed metrics to a single process
func (pm *ProcessManager) enhanceProcess(ctx context.Context, proc *Process) {
	// Get detailed process information
	if proc.PID < 0 || proc.PID > 2147483647 {
		return
	}
	if p, err := process.NewProcessWithContext(ctx, int32(proc.PID)); err == nil {
		// Get CPU percent
		if cpuPercent, err := p.CPUPercentWithContext(ctx); err == nil {
			proc.CPUPercent = cpuPercent
		}

		// Get memory info
		if memInfo, err := p.MemoryInfoWithContext(ctx); err == nil {
			proc.MemoryMB = float32(memInfo.RSS) / 1024 / 1024
		}

		// Get user
		if username, err := p.UsernameWithContext(ctx); err == nil {
			proc.User = username
		}

		// Get start time
		if createTime, err := p.CreateTimeWithContext(ctx); err == nil {
			proc.StartTime = time.Unix(createTime/1000, 0)
		}

		// Get full command line
		if cmdline, err := p.CmdlineWithContext(ctx); err == nil {
			proc.FullCommand = cmdline
		}
	}

	// Detect service type
	proc.ServiceType = pm.detectServiceType(proc.Port, proc.Command)
}

// detectServiceType identifies the type of service based on port and command
func (pm *ProcessManager) detectServiceType(port int, command string) string {
	// Check known service ports
	if service, exists := ServiceMap[port]; exists {
		return service
	}

	// Check command patterns
	command = strings.ToLower(command)

	switch {
	case strings.Contains(command, "node"):
		return "Node.js"
	case strings.Contains(command, "python"):
		return "Python"
	case strings.Contains(command, "java"):
		return "Java"
	case strings.Contains(command, "go"):
		return "Go"
	case strings.Contains(command, "ruby"):
		return "Ruby"
	case strings.Contains(command, "php"):
		return "PHP"
	case strings.Contains(command, "postgres"):
		return "PostgreSQL"
	case strings.Contains(command, "mysql"):
		return "MySQL"
	case strings.Contains(command, "redis"):
		return "Redis"
	case strings.Contains(command, "nginx"):
		return "Nginx"
	case strings.Contains(command, "apache"):
		return "Apache"
	case strings.Contains(command, "docker"):
		return "Docker"
	case strings.Contains(command, "code"):
		return "VS Code"
	case strings.Contains(command, "chrome") || strings.Contains(command, "firefox"):
		return "Browser"
	default:
		// Check port ranges
		switch {
		case port >= 3000 && port <= 3999:
			return "Development"
		case port >= 8000 && port <= 8999:
			return "Development"
		case port >= 9000 && port <= 9999:
			return "Development"
		case port < 1024:
			return "System"
		default:
			return "Unknown"
		}
	}
}

// countUniquePorts counts unique ports from process list
func (pm *ProcessManager) countUniquePorts(processes []Process) int {
	ports := make(map[int]bool)
	for _, proc := range processes {
		ports[proc.Port] = true
	}
	return len(ports)
}

// getProcessesUnix gets processes on Unix-like systems
func (pm *ProcessManager) getProcessesUnix(ctx context.Context, port int) ([]Process, error) {
	var cmd *exec.Cmd

	// Try lsof first (more reliable)
	if _, err := exec.LookPath("lsof"); err == nil {
		// #nosec G204: port is an integer, not user input
		cmd = exec.CommandContext(ctx, "lsof", "-i", fmt.Sprintf(":%d", port), "-P", "-n")
		if port == 0 {
			// #nosec G204: no user input
			cmd = exec.CommandContext(ctx, "lsof", "-i", "-P", "-n")
		}
	} else {
		// Fallback to netstat
		// #nosec G204: no user input
		cmd = exec.CommandContext(ctx, "netstat", "-tulpn")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", err)
	}

	return pm.parseUnixOutput(string(output), port)
}

// parseUnixOutput parses output from lsof or netstat
func (pm *ProcessManager) parseUnixOutput(output string, targetPort int) ([]Process, error) {
	var processes []Process
	lines := strings.Split(output, "\n")

	// Check if this is lsof output (contains "COMMAND" header)
	isLsof := strings.Contains(output, "COMMAND")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		if isLsof {
			if process := pm.parseLsofLine(line, targetPort); process != nil {
				processes = append(processes, *process)
			}
		} else {
			if process := pm.parseNetstatLine(line, targetPort); process != nil {
				processes = append(processes, *process)
			}
		}
	}

	return processes, nil
}

// parseLsofLine parses a single line from lsof output
func (pm *ProcessManager) parseLsofLine(line string, targetPort int) *Process {
	// Skip header line
	if strings.HasPrefix(line, "COMMAND") {
		return nil
	}

	fields := strings.Fields(line)
	if len(fields) < 9 {
		return nil
	}

	// Extract PID
	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil
	}

	// Extract port from the NAME field (usually field 8)
	nameField := fields[8]
	portRegex := regexp.MustCompile(`:(\d+)`)
	matches := portRegex.FindStringSubmatch(nameField)
	if len(matches) < 2 {
		return nil
	}

	port, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}

	// If we're looking for a specific port and this isn't it, skip
	if targetPort != 0 && port != targetPort {
		return nil
	}

	// Determine protocol
	protocol := "tcp"
	if strings.Contains(nameField, "UDP") {
		protocol = "udp"
	}

	// Extract addresses
	localAddr := ""
	remoteAddr := ""
	addrParts := strings.Split(nameField, "->")
	if len(addrParts) >= 1 {
		localAddr = addrParts[0]
	}
	if len(addrParts) >= 2 {
		remoteAddr = addrParts[1]
	}

	return &Process{
		PID:        pid,
		Port:       port,
		Command:    fields[0],
		Protocol:   protocol,
		State:      "LISTEN",
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
	}
}

// parseNetstatLine parses a single line from netstat output
func (pm *ProcessManager) parseNetstatLine(line string, targetPort int) *Process {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil
	}

	// Extract protocol
	protocol := strings.ToLower(fields[0])
	if !strings.HasPrefix(protocol, "tcp") && !strings.HasPrefix(protocol, "udp") {
		return nil
	}

	// Extract local address and port
	localAddr := fields[3]
	portIndex := strings.LastIndex(localAddr, ":")
	if portIndex == -1 {
		return nil
	}

	portStr := localAddr[portIndex+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil
	}

	// If we're looking for a specific port and this isn't it, skip
	if targetPort != 0 && port != targetPort {
		return nil
	}

	// Extract PID/Program name (usually last field)
	pidProgram := fields[len(fields)-1]
	pidIndex := strings.Index(pidProgram, "/")
	if pidIndex == -1 {
		return nil
	}

	pidStr := pidProgram[:pidIndex]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil
	}

	command := pidProgram[pidIndex+1:]
	state := "LISTEN"
	if len(fields) > 5 {
		state = fields[5]
	}

	remoteAddr := ""
	if len(fields) > 4 {
		remoteAddr = fields[4]
	}

	return &Process{
		PID:        pid,
		Port:       port,
		Command:    command,
		Protocol:   protocol,
		State:      state,
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
	}
}

func (pm *ProcessManager) getProcessesWindows(ctx context.Context, port int) ([]Process, error) {
	cmd := exec.CommandContext(ctx, "netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute netstat: %v", err)
	}

	return pm.parseWindowsOutput(ctx, string(output), port)
}

func (pm *ProcessManager) parseWindowsOutput(ctx context.Context, output string, targetPort int) ([]Process, error) {
	var processes []Process
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 5 {
			continue
		}

		protocol := strings.ToUpper(fields[0])
		if protocol != "TCP" && protocol != "UDP" {
			continue
		}

		// Parse local address
		localAddr := fields[1]
		portIndex := strings.LastIndex(localAddr, ":")
		if portIndex == -1 {
			continue
		}

		portStr := localAddr[portIndex+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		// If we're looking for a specific port and this isn't it, skip
		if targetPort != 0 && port != targetPort {
			continue
		}

		// Parse PID
		pidStr := fields[len(fields)-1]
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// Get process name
		command := pm.getWindowsProcessName(ctx, pid)

		state := "LISTENING"
		if len(fields) > 3 && protocol == "TCP" {
			state = fields[3]
		}

		remoteAddr := ""
		if len(fields) > 2 {
			remoteAddr = fields[2]
		}

		processes = append(processes, Process{
			PID:        pid,
			Port:       port,
			Command:    command,
			Protocol:   strings.ToLower(protocol),
			State:      state,
			LocalAddr:  localAddr,
			RemoteAddr: remoteAddr,
		})
	}

	return processes, scanner.Err()
}

func (pm *ProcessManager) getWindowsProcessName(ctx context.Context, pid int) string {
	// #nosec G204: pid is an integer, not user input
	cmd := exec.CommandContext(ctx, "tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && lines[0] != "" {
		// Parse CSV output
		fields := strings.Split(lines[0], ",")
		if len(fields) > 0 {
			// Remove quotes
			name := strings.Trim(fields[0], "\"")
			return name
		}
	}

	return "unknown"
}
