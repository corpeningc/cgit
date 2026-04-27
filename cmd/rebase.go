package cmd

import (
	"github.com/corpeningc/cgit/internal/config"
	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rebaseCmd.Flags().IntP("limit", "n", 0, "Number of commits to show (default from config)")
	rootCmd.AddCommand(rebaseCmd)
}

var rebaseCmd = &cobra.Command{
	Use:   "rebase",
	Short: "Interactively rebase the last N commits",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		cfg := config.Load()
		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = cfg.RebaseLimit
		}
		err := ui.StartRebasePicker(repo, limit)
		HandleError("rebasing", err, true)
	},
}
