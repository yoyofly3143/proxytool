package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"proxytool/internal/config"
	"proxytool/internal/node"
	"proxytool/internal/subscription"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage proxy nodes",
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available nodes from cached subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		nodes, err := loadAllNodes(cfg)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			fmt.Println("No nodes found. Run 'proxytool sub update' first.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tSERVER\tPORT\tSELECTED")
		for _, n := range nodes {
			selected := ""
			if n.Name == cfg.SelectedNode {
				selected = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", n.Name, n.Type, n.Server, n.Port, selected)
		}
		return w.Flush()
	},
}

var testTimeout int

var nodeTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test latency of all nodes and display results",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		nodes, err := loadAllNodes(cfg)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			return fmt.Errorf("no nodes found. Run 'proxytool sub update' first")
		}

		fmt.Printf("Testing %d nodes (timeout: %ds)...\n", len(nodes), testTimeout)
		timeout := time.Duration(testTimeout) * time.Second
		results := node.TestAll(nodes, timeout)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "#\tNAME\tTYPE\tLATENCY\tSTATUS")
		reachable := 0
		for i, r := range results {
			status := "ok"
			latency := fmt.Sprintf("%dms", r.Latency.Milliseconds())
			if r.Error != nil {
				status = "timeout"
				latency = "-"
			} else {
				reachable++
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, r.Node.Name, r.Node.Type, latency, status)
		}
		_ = w.Flush()
		fmt.Printf("\n%d/%d nodes reachable.\n", reachable, len(nodes))
		return nil
	},
}

var nodeSelectCmd = &cobra.Command{
	Use:   "select <name>",
	Short: "Select a node to use for proxying",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		nodes, err := loadAllNodes(cfg)
		if err != nil {
			return err
		}
		name := args[0]
		for _, n := range nodes {
			if n.Name == name {
				cfg.SelectedNode = name
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Printf("Selected node: %s (%s %s:%d)\n", n.Name, n.Type, n.Server, n.Port)
				return nil
			}
		}
		return fmt.Errorf("node '%s' not found. Use 'proxytool node list' to see available nodes", name)
	},
}

func init() {
	nodeTestCmd.Flags().IntVarP(&testTimeout, "timeout", "t", 5, "Timeout per node in seconds")
	nodeCmd.AddCommand(nodeListCmd)
	nodeCmd.AddCommand(nodeTestCmd)
	nodeCmd.AddCommand(nodeSelectCmd)
}

// loadAllNodes loads nodes from all cached subscriptions
func loadAllNodes(cfg *config.Config) ([]subscription.Node, error) {
	var allNodes []subscription.Node
	for _, s := range cfg.Subscriptions {
		data, err := subscription.LoadCache(s.Name)
		if err != nil {
			continue
		}
		nodes, err := subscription.Parse(data)
		if err != nil {
			continue
		}
		allNodes = append(allNodes, nodes...)
	}
	return allNodes, nil
}
