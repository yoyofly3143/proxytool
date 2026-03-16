package cmd

import (
	"fmt"
	"os"

	"proxytool/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "proxytool",
	Short: "CLI proxy manager using mihomo engine",
	Long: `proxytool - a CLI tool for managing proxy nodes.

It downloads subscription links, tests node latency, starts a local proxy
via mihomo (Clash-Meta), and controls Docker/system proxy settings.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(subCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(dockerCmd)
	rootCmd.AddCommand(systemCmd)
	rootCmd.AddCommand(statusCmd)
}

// loadConfig is a helper used by subcommands
func loadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

// statusCmd shows overall status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show overall status (proxy + docker + system proxy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}
