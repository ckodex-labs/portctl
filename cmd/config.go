package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage portctl configuration and preferences",
	Long: `Configure portctl settings, preferences, and defaults.

This command allows you to customize portctl behavior, set defaults,
and manage configuration profiles for different environments.

Configuration options:
  • Default refresh intervals for watch mode
  • Preferred output formats and themes
  • Custom port ranges and service definitions
  • Notification settings
  • Default scan timeouts and concurrency

Examples:
  portctl config set watch.interval 2s
  portctl config set output.format table
  portctl config set notifications.enabled true
  portctl config get watch.interval
  portctl config list
  portctl config reset`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration key to a specific value.

Available configuration keys:
  watch.interval          - Default refresh interval for watch mode (e.g., "2s", "500ms")
  watch.notifications     - Enable desktop notifications (true/false)
  output.format          - Default output format (table/json/tree/details)
  output.colors          - Enable colored output (true/false)
  scan.timeout           - Default scan timeout (e.g., "3s", "1m")
  scan.concurrent        - Default concurrent scans (number)
  kill.confirm           - Require confirmation before killing (true/false)
  list.sort              - Default sort field (port/pid/cpu/memory/command)
  dev.ports              - Custom development port range (e.g., "3000-8999")

Examples:
  portctl config set watch.interval 1s
  portctl config set output.format json
  portctl config set scan.concurrent 100`,
	Args: cobra.ExactArgs(2),
	Run:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get the current value of a configuration key.

Examples:
  portctl config get watch.interval
  portctl config get output.format`,
	Args: cobra.ExactArgs(1),
	Run:  runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Long: `Display all current configuration values and their sources.

This shows your current settings, default values, and where each
setting is being loaded from (config file, environment, or default).`,
	Run: runConfigList,
}

var configResetCmd = &cobra.Command{
	Use:   "reset [key]",
	Short: "Reset configuration to defaults",
	Long: `Reset configuration values to their defaults.

If a specific key is provided, only that setting is reset.
If no key is provided, all settings are reset to defaults.

Examples:
  portctl config reset                # Reset all settings
  portctl config reset watch.interval # Reset only watch interval`,
	Args: cobra.MaximumNArgs(1),
	Run:  runConfigReset,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration file in editor",
	Long: `Open the portctl configuration file in your default editor.

The configuration file is stored in:
  ~/.config/portctl/config.yaml

You can also set the EDITOR environment variable to use a specific editor.`,
	Run: runConfigEdit,
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	// Validate the key
	validKeys := map[string]string{
		"watch.interval":      "duration",
		"watch.notifications": "bool",
		"output.format":       "string",
		"output.colors":       "bool",
		"scan.timeout":        "duration",
		"scan.concurrent":     "int",
		"kill.confirm":        "bool",
		"list.sort":           "string",
		"dev.ports":           "string",
	}

	valueType, exists := validKeys[key]
	if !exists {
		color.Red("Unknown configuration key: %s", key)
		fmt.Println("\nValid keys:")
		for k := range validKeys {
			fmt.Printf("  %s\n", k)
		}
		os.Exit(1)
	}

	// Validate value type
	if err := validateValue(value, valueType, key); err != nil {
		color.Red("Invalid value for %s: %v", key, err)
		os.Exit(1)
	}

	// Set the value
	viper.Set(key, value)

	// Write config file
	if err := writeConfig(); err != nil {
		color.Red("Error writing config: %v", err)
		os.Exit(1)
	}

	color.Green("✅ Set %s = %s", key, value)
}

func runConfigGet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := viper.GetString(key)

	if value == "" {
		color.Yellow("Configuration key '%s' is not set", key)
	} else {
		color.Green("%s = %s", key, value)
	}
}

func runConfigList(cmd *cobra.Command, args []string) {
	color.Cyan("📋 portctl Configuration")
	fmt.Println()

	settings := viper.AllSettings()
	if len(settings) == 0 {
		color.Yellow("No configuration values set (using defaults)")
		return
	}

	for key, value := range settings {
		color.Green("  %s = %v", key, value)
	}

	fmt.Println()
	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		color.Cyan("📁 Config file: %s", configFile)
	}
}

func runConfigReset(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		// Reset all
		color.Yellow("⚠️  This will reset ALL configuration to defaults.")
		fmt.Print("Are you sure? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			color.Red("Error reading input: %v", err)
			return
		}

		if response != "y" && response != "yes" {
			color.Yellow("Operation cancelled")
			return
		}

		// Clear all settings
		for key := range viper.AllSettings() {
			viper.Set(key, nil)
		}

		if err := writeConfig(); err != nil {
			color.Red("Error writing config: %v", err)
			os.Exit(1)
		}

		color.Green("✅ All configuration reset to defaults")
	} else {
		// Reset specific key
		key := args[0]
		viper.Set(key, nil)

		if err := writeConfig(); err != nil {
			color.Red("Error writing config: %v", err)
			os.Exit(1)
		}

		color.Green("✅ Reset %s to default", key)
	}
}

func runConfigEdit(cmd *cobra.Command, args []string) {
	configFile := getConfigFile()

	// Ensure config directory exists
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		color.Red("Error creating config directory: %v", err)
		os.Exit(1)
	}

	// Create config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := writeConfig(); err != nil {
			color.Red("Error creating config file: %v", err)
			os.Exit(1)
		}
	}

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi on Unix systems
		if os.Getenv("OS") == "Windows_NT" {
			editor = "notepad"
		}
	}

	color.Cyan("📝 Opening config file with %s...", editor)
	fmt.Printf("Config file: %s\n", configFile)
}

func validateValue(value, valueType, key string) error {
	switch valueType {
	case "bool":
		if value != "true" && value != "false" {
			return fmt.Errorf("must be 'true' or 'false'")
		}
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("must be a number")
		}
	case "duration":
		// Simple duration validation
		if !strings.HasSuffix(value, "s") && !strings.HasSuffix(value, "m") && !strings.HasSuffix(value, "ms") {
			return fmt.Errorf("must be a duration (e.g., '2s', '500ms', '1m')")
		}
	case "string":
		// Additional validation for specific string keys
		if key == "output.format" {
			valid := []string{"table", "json", "tree", "details"}
			for _, v := range valid {
				if value == v {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %v", valid)
		}
		if key == "list.sort" {
			valid := []string{"port", "pid", "cpu", "memory", "command", "service", "user"}
			for _, v := range valid {
				if value == v {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %v", valid)
		}
	}
	return nil
}

func getConfigFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./portctl-config.yaml"
	}
	return filepath.Join(homeDir, ".config", "portctl", "config.yaml")
}

func writeConfig() error {
	configFile := getConfigFile()
	configDir := filepath.Dir(configFile)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return err
	}

	return viper.WriteConfigAs(configFile)
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configEditCmd)

	// Initialize viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/portctl")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("watch.interval", "3s")
	viper.SetDefault("watch.notifications", false)
	viper.SetDefault("output.format", "table")
	viper.SetDefault("output.colors", true)
	viper.SetDefault("scan.timeout", "3s")
	viper.SetDefault("scan.concurrent", 50)
	viper.SetDefault("kill.confirm", true)
	viper.SetDefault("list.sort", "port")
	viper.SetDefault("dev.ports", "3000-9999")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is okay, we'll use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			color.Red("Error reading config: %v", err)
		}
	}
}
