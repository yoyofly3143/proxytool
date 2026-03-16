package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"proxytool/internal/config"
	"proxytool/internal/subscription"

	"github.com/spf13/cobra"
)

var subCmd = &cobra.Command{
	Use:   "sub",
	Short: "Manage subscriptions",
}

var subAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add or update a subscription",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, url := args[0], args[1]
		cfg := loadConfig()
		cfg.AddSubscription(name, url)
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Subscription '%s' saved.\n", name)
		return nil
	},
}

var subRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a subscription",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		if !cfg.RemoveSubscription(args[0]) {
			return fmt.Errorf("subscription '%s' not found", args[0])
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Subscription '%s' removed.\n", args[0])
		return nil
	},
}

var subListCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		if len(cfg.Subscriptions) == 0 {
			fmt.Println("No subscriptions. Use 'proxytool sub add <name> <url>'")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tURL")
		for _, s := range cfg.Subscriptions {
			fmt.Fprintf(w, "%s\t%s\n", s.Name, s.URL)
		}
		return w.Flush()
	},
}

var subUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Download/refresh subscriptions (all or specific)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		if len(cfg.Subscriptions) == 0 {
			return fmt.Errorf("no subscriptions configured. Use 'proxytool sub add'")
		}

		targets := cfg.Subscriptions
		if len(args) == 1 {
			name := args[0]
			found := false
			for _, s := range cfg.Subscriptions {
				if s.Name == name {
					targets = []config.Subscription{s}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("subscription '%s' not found", name)
			}
		}

		totalNodes := 0
		for _, s := range targets {
			fmt.Printf("Updating '%s' ...\n", s.Name)
			data, err := subscription.Download(s.URL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Failed: %v\n", err)
				continue
			}
			nodes, err := subscription.Parse(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Parse error: %v\n", err)
				continue
			}
			if err := subscription.SaveCache(s.Name, data); err != nil {
				fmt.Fprintf(os.Stderr, "  Cache save error: %v\n", err)
			}
			fmt.Printf("  %d nodes fetched.\n", len(nodes))
			totalNodes += len(nodes)
		}

		cfg.LastUpdate = time.Now()
		_ = cfg.Save()
		fmt.Printf("Done. Total: %d nodes.\n", totalNodes)
		return nil
	},
}

func init() {
	subCmd.AddCommand(subAddCmd)
	subCmd.AddCommand(subRemoveCmd)
	subCmd.AddCommand(subListCmd)
	subCmd.AddCommand(subUpdateCmd)
}
