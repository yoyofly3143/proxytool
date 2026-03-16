package cmd

import (
	"fmt"
	"os"

	"proxytool/internal/config"
	"proxytool/internal/engine"
	"proxytool/internal/node"
	"path/filepath"

	"github.com/spf13/cobra"
)


var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Control the proxy process",
}

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy (auto-selects fastest node if none selected)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()

		// Ensure mihomo binary is available
		fmt.Println("Checking mihomo binary...")
		if err := engine.EnsureBinary(); err != nil {
			return fmt.Errorf("failed to prepare mihomo: %w", err)
		}

		// Find the node to use
		selectedProxy, err := resolveNode(cfg)
		if err != nil {
			return err
		}

		fmt.Printf("Using node: %s\n", selectedProxy["name"])

		// Generate config
		if err := engine.GenerateConfig(selectedProxy, cfg.HTTPPort, cfg.SocksPort, cfg.AllowLAN); err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}


		// Start!
		return engine.Start()
	},
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		return engine.Stop()
	},
}

var proxyRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		_ = engine.Stop()
		if err := engine.EnsureBinary(); err != nil {
			return err
		}
		selectedProxy, err := resolveNode(cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Using node: %s\n", selectedProxy["name"])
		if err := engine.GenerateConfig(selectedProxy, cfg.HTTPPort, cfg.SocksPort, cfg.AllowLAN); err != nil {
			return err
		}

		return engine.Start()
	},
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show proxy status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Proxy: %s\n", engine.Status())
		return nil
	},
}

var proxyLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show the last 20 lines of mihomo logs",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		logPath := filepath.Join(home, ".proxytool", "mihomo", "mihomo.log")
		data, err := os.ReadFile(logPath)
		if err != nil {
			fmt.Printf("No logs found at %s\n", logPath)
			return
		}
		lines := splitLines(string(data))
		start := 0
		if len(lines) > 20 {
			start = len(lines) - 20
		}
		fmt.Printf("Last 20 log lines from %s:\n", logPath)
		for i := start; i < len(lines); i++ {
			fmt.Println(lines[i])
		}
	},
}

var proxyConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the generated mihomo configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		p := filepath.Join(home, ".proxytool", "mihomo", "config.yaml")
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Printf("Config not found at %s\n", p)
			return
		}
		fmt.Printf("Mihomo Config (%s):\n---\n%s\n", p, string(data))
	},
}

var proxyAllowLanCmd = &cobra.Command{
	Use:   "allow-lan <on|off>",
	Short: "Toggle whether the proxy listens on 0.0.0.0 (on) or 127.0.0.1 (off)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		if args[0] == "on" {
			cfg.AllowLAN = true
		} else if args[0] == "off" {
			cfg.AllowLAN = false
		} else {
			return fmt.Errorf("invalid argument: %s (use 'on' or 'off')", args[0])
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("LAN access set to: %s\n", args[0])
		fmt.Println("Please restart the proxy for changes to take effect.")
		return nil
	},
}




func init() {
	proxyCmd.AddCommand(proxyStartCmd)
	proxyCmd.AddCommand(proxyStopCmd)
	proxyCmd.AddCommand(proxyRestartCmd)
	proxyCmd.AddCommand(proxyStatusCmd)
	proxyCmd.AddCommand(proxyLogsCmd)
	proxyCmd.AddCommand(proxyConfigCmd)
	proxyCmd.AddCommand(proxyAllowLanCmd)
}





// resolveNode finds the configured node, or auto-selects the fastest available
func resolveNode(cfg *config.Config) (map[string]interface{}, error) {
	nodes, err := loadAllNodes(cfg)
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available. Run 'proxytool sub update' first")
	}

	// If a specific node is selected, use it
	if cfg.SelectedNode != "" {
		for _, n := range nodes {
			if n.Name == cfg.SelectedNode {
				return n.Raw, nil
			}
		}
		fmt.Fprintf(os.Stderr, "Warning: selected node '%s' not found, auto-selecting fastest...\n", cfg.SelectedNode)
	}

	// Auto-select fastest node
	fmt.Printf("Testing %d nodes to find the fastest...\n", len(nodes))
	results := node.TestAll(nodes, 5000000000) // 5s
	best := node.BestNode(results)
	if best == nil {
		return nil, fmt.Errorf("no reachable nodes found")
	}

	// Save selection
	cfg.SelectedNode = best.Node.Name
	_ = cfg.Save()
	fmt.Printf("Auto-selected: %s (%dms)\n", best.Node.Name, best.Latency.Milliseconds())

	return best.Node.Raw, nil
}

// runStatus shows all subsystem statuses
func runStatus() error {
	cfg, _ := config.Load()
	fmt.Printf("Proxy:         %s\n", engine.Status())
	fmt.Printf("HTTP Port:     %d\n", cfg.HTTPPort)
	fmt.Printf("SOCKS Port:    %d\n", cfg.SocksPort)
	fmt.Printf("Selected Node: %s\n", func() string {
		if cfg.SelectedNode == "" {
			return "(none)"
		}
		return cfg.SelectedNode
	}())

	// Import here to avoid import cycle
	fmt.Printf("System Proxy:  %s\n", getSysProxyStatus())
	fmt.Printf("Docker Proxy:  %s\n", getDockerProxyStatus())
	return nil
}

func getSysProxyStatus() string {
	// Avoid import cycle by reading directly
	lines, err := readFileLines("/etc/environment")
	if err != nil {
		return "unknown"
	}
	for _, line := range lines {
		if len(line) > 10 && line[:10] == "http_proxy" {
			return "enabled"
		}
	}
	return "disabled"
}

func getDockerProxyStatus() string {
	return "see 'proxytool docker status'"
}

func readFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, l := range splitLines(string(data)) {
		lines = append(lines, l)
	}
	return lines, nil
}

func splitLines(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
