package cmd

import (
	"fmt"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(popCmd)
	rootCmd.AddCommand(storeCmd)
	rootCmd.AddCommand(fullCleanCmd)
}

var popCmd = &cobra.Command{
	Use:   "pop",
	Short: "Interactively select and pop a stash",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := ui.StartStashPicker(repo)
		HandleError("popping stash", err, true)
	},
}

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Store changes in a stash",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		var err error

		if len(args) == 1 {
			var stashName = args[0]
			err = repo.Stash(stashName)
		} else {
			err = repo.Stash("")
		}

		HandleError("stashing changes", err, true)

		fmt.Println("Successfully stored changes.")
	},
}

var fullCleanCmd = &cobra.Command{
	Use:     "full-clean",
	Aliases: []string{"fc"},
	Short:   "Hard reset branch; Clean files and directories",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.FullClean()
		HandleError("performing full clean", err, true)

		fmt.Println("Successfully cleaned repository.")
	},
}
