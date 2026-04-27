package cmd

import (
	"fmt"

	"github.com/corpeningc/cgit/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or edit cgit configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		fmt.Printf("Config file: %s\n\n", config.Path())
		fmt.Printf("log_limit:    %d\n", cfg.LogLimit)
		fmt.Printf("rebase_limit: %d\n", cfg.RebaseLimit)
		fmt.Printf("split_pane:   %v\n", cfg.SplitPane)
		if cfg.Editor != "" {
			fmt.Printf("editor:       %s\n", cfg.Editor)
		} else {
			fmt.Printf("editor:       (uses $EDITOR)\n")
		}
	},
}
