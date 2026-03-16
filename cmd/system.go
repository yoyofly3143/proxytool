package cmd

import (
	"fmt"

	"proxytool/internal/sysproxy"

	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "Control system-wide proxy settings (/etc/environment)",
}

var systemEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable system proxy (writes http_proxy to /etc/environment)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		return sysproxy.Enable("127.0.0.1", cfg.HTTPPort)
	},
}

var systemDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable system proxy (removes proxy vars from /etc/environment)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return sysproxy.Disable()
	},
}

var systemStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system proxy status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("System proxy: %s\n", sysproxy.Status())
		return nil
	},
}

func init() {
	systemCmd.AddCommand(systemEnableCmd)
	systemCmd.AddCommand(systemDisableCmd)
	systemCmd.AddCommand(systemStatusCmd)
}
