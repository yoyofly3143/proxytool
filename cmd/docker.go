package cmd

import (
	"fmt"

	"proxytool/internal/dockerproxy"

	"github.com/spf13/cobra"
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Control Docker proxy settings",
}

var dockerEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Docker proxy (points to local proxy port)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		return dockerproxy.Enable(cfg.HTTPPort)
	},
}

var dockerDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Docker proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		return dockerproxy.Disable()
	},
}

var dockerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Docker proxy status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Docker proxy: %s\n", dockerproxy.Status())
		return nil
	},
}

func init() {
	dockerCmd.AddCommand(dockerEnableCmd)
	dockerCmd.AddCommand(dockerDisableCmd)
	dockerCmd.AddCommand(dockerStatusCmd)
}
