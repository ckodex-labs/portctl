package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	tablepretty "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/briandowns/spinner"
)

var (
	scanTimeout    time.Duration
	scanConcurrent int
	scanRange      string
	scanCommon     bool
	scanUDP        bool
)

type ScanResult struct {
	Port     int
	Host     string
	Protocol string
	Status   string
	Service  string
	Banner   string
	Error    error
}

var scanCmd = &cobra.Command{
	Use:   "scan [host] [port|port-range]",
	Short: "Scan ports on local or remote hosts",
	Long: `Scan for open ports on local or remote hosts with service detection.

This command performs TCP/UDP port scans with banner grabbing and service
identification. Useful for network discovery and security assessment.

Examples:
  # Scan common ports on localhost
  portctl scan localhost --common
  
  # Scan specific ports
  portctl scan 192.168.1.1 80,443,22
  portctl scan localhost 3000-4000
  
  # Advanced scanning
  portctl scan example.com 1-1000 --timeout 2s
  portctl scan localhost --udp --range "53,67,68"
  
  # Fast concurrent scan
  portctl scan 192.168.1.0/24 --common --concurrent 100`,
	Aliases: []string{"portscan", "nmap"},
	Args:    cobra.RangeArgs(1, 2),
	Run:     runScan,
}

func runScan(cmd *cobra.Command, args []string) {
	host := args[0]
	if host == "" {
		host = "localhost"
	}

	var ports []int
	var err error

	if scanCommon {
		ports = []int{21, 22, 23, 25, 53, 80, 110, 135, 139, 143, 443, 993, 995, 1433, 1521, 3306, 3389, 5432, 5900, 8080}
	} else if scanRange != "" {
		ports, err = parsePortRange(scanRange)
		if err != nil {
			color.Red("Error parsing port range: %v", err)
			os.Exit(1)
		}
	} else if len(args) > 1 {
		ports, err = parsePortRange(args[1])
		if err != nil {
			color.Red("Error parsing ports: %v", err)
			os.Exit(1)
		}
	} else {
		color.Red("Please specify ports to scan or use --common")
		os.Exit(1)
	}

	color.Cyan("🔍 Scanning %s for %d port(s)...", host, len(ports))

	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if err := s.Color("cyan"); err != nil {
		color.Red("Spinner color error: %v", err)
	}
	s.Suffix = fmt.Sprintf(" Scanning %d ports ", len(ports))
	s.Start()

	results := scanPorts(host, ports)
	s.Stop()

	// Filter open ports
	var openPorts []ScanResult
	for _, result := range results {
		if result.Status == "open" {
			openPorts = append(openPorts, result)
		}
	}

	if len(openPorts) == 0 {
		color.Yellow("No open ports found on %s", host)
		return
	}

	color.Green("✅ Found %d open port(s) on %s:", len(openPorts), host)
	displayScanResults(openPorts)
}

func parsePortRange(portStr string) ([]int, error) {
	var ports []int

	ranges := strings.Split(portStr, ",")
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		
		if strings.Contains(r, "-") {
			// Handle range like "80-90"
			parts := strings.Split(r, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", r)
			}

			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port: %s", parts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port: %s", parts[1])
			}

			if start > end {
				return nil, fmt.Errorf("start port must be less than end port")
			}

			for port := start; port <= end; port++ {
				ports = append(ports, port)
			}
		} else {
			// Single port
			port, err := strconv.Atoi(r)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", r)
			}
			ports = append(ports, port)
		}
	}

	return ports, nil
}

func scanPorts(host string, ports []int) []ScanResult {
	results := make([]ScanResult, len(ports))
	sem := make(chan struct{}, scanConcurrent)
	var wg sync.WaitGroup

	for i, port := range ports {
		wg.Add(1)
		go func(idx, p int) {
			defer wg.Done()
			sem <- struct{}{} // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			results[idx] = scanPort(host, p)
		}(i, port)
	}

	wg.Wait()
	return results
}

func scanPort(host string, port int) ScanResult {
	result := ScanResult{
		Port:     port,
		Host:     host,
		Protocol: "tcp",
		Status:   "closed",
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, scanTimeout)
	if err != nil {
		result.Error = err
		return result
	}
	defer conn.Close()

	result.Status = "open"
	result.Service = getServiceName(port)
	
	// Try to grab banner
	banner := grabBanner(conn, port)
	if banner != "" {
		result.Banner = banner
	}

	return result
}

func getServiceName(port int) string {
	services := map[int]string{
		21:   "FTP",
		22:   "SSH", 
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		80:   "HTTP",
		110:  "POP3",
		135:  "RPC",
		139:  "NetBIOS",
		143:  "IMAP",
		443:  "HTTPS",
		993:  "IMAPS",
		995:  "POP3S",
		1433: "MSSQL",
		1521: "Oracle",
		3306: "MySQL",
		3389: "RDP",
		5432: "PostgreSQL",
		5900: "VNC",
		8080: "HTTP-Alt",
	}

	if service, exists := services[port]; exists {
		return service
	}
	return "Unknown"
}

func grabBanner(conn net.Conn, port int) string {
	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return ""
	}
	
	// Send HTTP request for web services
	if port == 80 || port == 8080 || port == 443 {
		if _, err := conn.Write([]byte("HEAD / HTTP/1.0\r\n\r\n")); err != nil {
			return ""
		}
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	banner := string(buffer[:n])
	// Clean up banner
	banner = strings.TrimSpace(banner)
	if len(banner) > 100 {
		banner = banner[:100] + "..."
	}
	
	return banner
}

func displayScanResults(results []ScanResult) {
	t := tablepretty.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tablepretty.StyleColoredBright)

	// Set header and header color
	t.AppendHeader(tablepretty.Row{"Port", "Protocol", "Service", "Status", "Banner"})
	t.Style().Color.Header = text.Colors{text.FgHiBlue, text.Bold}

	// Set column configs for alignment and color
	t.SetColumnConfigs([]tablepretty.ColumnConfig{
		{Number: 1, Align: text.AlignRight, Colors: text.Colors{text.FgCyan, text.Bold}}, // Port
		{Number: 2, Align: text.AlignCenter}, // Protocol
		{Number: 3, Align: text.AlignLeft, Colors: text.Colors{text.Bold}}, // Service
		{Number: 4, Align: text.AlignCenter}, // Status
		{Number: 5, Align: text.AlignLeft, Colors: text.Colors{text.FgYellow}}, // Banner
	})

	for _, result := range results {
		banner := result.Banner
		if len(banner) > 50 {
			banner = banner[:50] + "..."
		}

		row := tablepretty.Row{
			result.Port,
			result.Protocol,
			result.Service,
			result.Status,
			banner,
		}
		t.AppendRow(row)
	}

	t.Render()
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().DurationVarP(&scanTimeout, "timeout", "t", 3*time.Second,
		"Connection timeout for each port")
	scanCmd.Flags().IntVarP(&scanConcurrent, "concurrent", "c", 50,
		"Number of concurrent scans")
	scanCmd.Flags().StringVarP(&scanRange, "range", "r", "",
		"Port range to scan (e.g., '80,443,1000-2000')")
	scanCmd.Flags().BoolVar(&scanCommon, "common", false,
		"Scan common ports (21,22,23,25,53,80,110,135,139,143,443,993,995,1433,1521,3306,3389,5432,5900,8080)")
	scanCmd.Flags().BoolVar(&scanUDP, "udp", false,
		"Scan UDP ports instead of TCP")
}
