package cmd

import (
	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCommand)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(conflictsCmd)
}

var statusCommand = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st"},
	Short:   "Browse repository status in an interactive TUI",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		err := ui.StartStatusViewer(repo)
		HandleError("showing status", err, true)
	},
}

var logCmd = &cobra.Command{
	Use:     "log",
	Aliases: []string{"l"},
	Short:   "Browse commit history in an interactive viewer",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		content, err := repo.GetLog(100)
		HandleError("getting git log", err, true)

		err = ui.StartLogViewer(repo, content)
		HandleError("showing log viewer", err, true)
	},
}

var conflictsCmd = &cobra.Command{
	Use:     "conflicts",
	Aliases: []string{"cf"},
	Short:   "Resolve merge conflicts interactively",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		err := ui.StartConflictsPicker(repo)
		HandleError("resolving conflicts", err, true)
	},
}
