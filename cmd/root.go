package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portctl",
	Short: "A CLI tool to manage processes on specific ports",
	Long: `portctl is a command-line tool that helps developers manage processes 
running on specific ports. You can list processes, kill them, and get detailed 
information about what's using your ports.

Examples:
  portctl list 8080          # List processes on port 8080
  portctl list               # List all processes with open ports
  portctl kill 8080          # Kill processes on port 8080
  portctl kill --pid 12345   # Kill process by PID`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Show version")
}
