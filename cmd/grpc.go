package cmd

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	process "dagger/portctl/pkg"
	pb "dagger/portctl/proto"
)

// safeIntToInt32 converts an int to int32 safely, clamping to int32 bounds to prevent overflow.
func safeIntToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

var (
	grpcPort string
)

var grpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "Start the gRPC API server",
	Long: `Start a gRPC server to allow network-based access to portctl functionality.

This command runs a gRPC server on localhost:57251 (by default) that exposes
all portctl operations via a network API. Useful for automation, testing,
and integration with other tools.

Examples:
  portctl grpc                    # Start on default port 57251
  portctl grpc --port 9090        # Start on custom port`,
	Run: runGRPC,
}

func init() {
	rootCmd.AddCommand(grpcCmd)
	grpcCmd.Flags().StringVarP(&grpcPort, "port", "p", "57251", "Port to listen on")
}

type portctlServer struct {
	pb.UnimplementedPortctlServiceServer
	startTime time.Time
}

func newPortctlServer() *portctlServer {
	return &portctlServer{
		startTime: time.Now(),
	}
}

func (s *portctlServer) ListProcesses(ctx context.Context, req *pb.ListProcessesRequest) (*pb.ListProcessesResponse, error) {
	pm := process.NewProcessManager()

	var processes []process.Process
	var err error

	if req.Port != nil && *req.Port > 0 {
		processes, err = pm.GetProcessesOnPort(ctx, int(*req.Port))
	} else {
		processes, err = pm.GetAllProcesses(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	// Apply filters
	if req.Service != nil || req.User != nil {
		filterOpts := process.FilterOptions{}
		if req.Service != nil {
			filterOpts.Service = *req.Service
		}
		if req.User != nil {
			filterOpts.User = *req.User
		}
		processes = pm.FilterProcesses(processes, filterOpts)
	}

	// Convert to proto
	pbProcesses := make([]*pb.Process, len(processes))
	for i, p := range processes {
		pbProcesses[i] = &pb.Process{
			Pid:         safeIntToInt32(p.PID),
			Port:        safeIntToInt32(p.Port),
			Command:     p.Command,
			ServiceType: p.ServiceType,
			User:        p.User,
			CpuPercent:  p.CPUPercent,
			MemoryMb:    float64(p.MemoryMB),
			StartTime:   p.StartTime.Unix(),
		}
	}

	return &pb.ListProcessesResponse{
		Processes: pbProcesses,
	}, nil
}

func (s *portctlServer) KillProcess(ctx context.Context, req *pb.KillProcessRequest) (*pb.KillProcessResponse, error) {
	pm := process.NewProcessManager()

	switch target := req.Target.(type) {
	case *pb.KillProcessRequest_Pid:
		err := pm.KillProcess(ctx, int(target.Pid), req.Force)
		if err != nil {
			return &pb.KillProcessResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to kill PID %d: %v", target.Pid, err),
			}, nil
		}
		return &pb.KillProcessResponse{
			Success:     true,
			Message:     fmt.Sprintf("Successfully killed process %d", target.Pid),
			KilledCount: 1,
		}, nil

	case *pb.KillProcessRequest_Port:
		processes, err := pm.GetProcessesOnPort(ctx, int(target.Port))
		if err != nil {
			return &pb.KillProcessResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to find processes on port %d: %v", target.Port, err),
			}, nil
		}

		if len(processes) == 0 {
			return &pb.KillProcessResponse{
				Success: true,
				Message: fmt.Sprintf("No processes found on port %d", target.Port),
			}, nil
		}

		var pids []int
		for _, p := range processes {
			pids = append(pids, p.PID)
		}

		results := pm.KillProcesses(ctx, pids, req.Force)

		successCount := 0
		var errors []string
		for _, err := range results {
			if err == nil {
				successCount++
			} else {
				errors = append(errors, err.Error())
			}
		}

		msg := fmt.Sprintf("Killed %d/%d processes on port %d", successCount, len(pids), target.Port)
		if len(errors) > 0 {
			msg += fmt.Sprintf(". Errors: %v", errors)
		}

		return &pb.KillProcessResponse{
			Success:     successCount > 0,
			Message:     msg,
			KilledCount: safeIntToInt32(successCount),
		}, nil

	default:
		return &pb.KillProcessResponse{
			Success: false,
			Message: "Must provide either pid or port",
		}, nil
	}
}

func (s *portctlServer) ScanPorts(ctx context.Context, req *pb.ScanPortsRequest) (*pb.ScanPortsResponse, error) {
	host := req.Host
	if host == "" {
		host = "localhost"
	}

	// Validate port range to prevent resource exhaustion
	startPort := int(req.StartPort)
	endPort := int(req.EndPort)

	if startPort < 1 || startPort > 65535 {
		return nil, fmt.Errorf("invalid start port: %d (must be 1-65535)", startPort)
	}
	if endPort < 1 || endPort > 65535 {
		return nil, fmt.Errorf("invalid end port: %d (must be 1-65535)", endPort)
	}
	if startPort > endPort {
		return nil, fmt.Errorf("start port (%d) must be less than or equal to end port (%d)", startPort, endPort)
	}
	// Limit scan range to prevent DoS
	const maxPortRange = 10000
	portCount := endPort - startPort + 1 // inclusive range
	if portCount > maxPortRange {
		return nil, fmt.Errorf("port range too large: %d (max %d ports allowed)", portCount, maxPortRange)
	}

	var ports []int
	for p := startPort; p <= endPort; p++ {
		ports = append(ports, p)
	}

	results := scanPorts(host, ports)

	pbResults := make([]*pb.PortScanResult, len(results))
	for i, r := range results {
		pbResults[i] = &pb.PortScanResult{
			Port:    safeIntToInt32(r.Port),
			Status:  r.Status,
			Service: r.Service,
		}
	}

	return &pb.ScanPortsResponse{
		Results: pbResults,
	}, nil
}

func (s *portctlServer) GetSystemStats(ctx context.Context, req *pb.SystemStatsRequest) (*pb.SystemStatsResponse, error) {
	pm := process.NewProcessManager()
	stats, err := pm.GetSystemStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system stats: %w", err)
	}

	// Calculate memory percent safely to avoid division by zero
	memoryPercent := 0.0
	totalMemory := stats.MemoryUsageGB + stats.AvailableMemoryGB
	if totalMemory > 0 {
		memoryPercent = (stats.MemoryUsageGB / totalMemory) * 100
	}

	return &pb.SystemStatsResponse{
		CpuPercent:     stats.CPUUsagePercent,
		MemoryPercent:  memoryPercent,
		TotalProcesses: safeIntToInt32(stats.TotalProcesses),
		ListeningPorts: safeIntToInt32(stats.ListeningPorts),
	}, nil
}

func (s *portctlServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	uptime := time.Since(s.startTime).Seconds()
	return &pb.StatusResponse{
		Version:       "1.0.0",
		UptimeSeconds: int64(uptime),
		ServerType:    "grpc",
	}, nil
}

func runGRPC(cmd *cobra.Command, args []string) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		color.Red("Failed to listen on port %s: %v", grpcPort, err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPortctlServiceServer(grpcServer, newPortctlServer())

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		color.Yellow("\nShutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	color.Green("🚀 gRPC server listening on :%s", grpcPort)
	color.Cyan("Test with: grpcurl -plaintext localhost:%s list", grpcPort)

	if err := grpcServer.Serve(lis); err != nil {
		color.Red("Server error: %v", err)
		os.Exit(1)
	}
}
