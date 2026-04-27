package cmd

import (
	"fmt"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(commitAndPushCmd)
	rootCmd.AddCommand(amendCmd)
	rootCmd.AddCommand(undoCmd)

	amendCmd.Flags().BoolP("no-edit", "n", false, "Amend staged changes without changing the commit message")
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Args:  cobra.RangeArgs(0, 1),
	Short: "Commit staged changes with a message",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		if len(args) == 0 {
			err := ui.StartCommitInput(repo)
			HandleError("committing changes", err, true)
			return
		}

		commitMsg := args[0]
		err := repo.Commit(commitMsg)
		HandleError("committing changes", err, true)

		fmt.Println("Successfully committed changes.")
	},
}

var commitAndPushCmd = &cobra.Command{
	Use:     "commit-and-push",
	Aliases: []string{"cap"},
	Short:   "Commit and push changes",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		commitMsg := args[0]
		err := repo.Commit(commitMsg)
		HandleError("committing changes", err, true)

		err = repo.Push()
		HandleError("pushing changes", err, true)

		fmt.Println("Successfully committed and pushed changes.")
	},
}

var amendCmd = &cobra.Command{
	Use:   "amend",
	Short: "Amend the last commit",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		noEdit, _ := cmd.Flags().GetBool("no-edit")
		if noEdit {
			err := repo.AmendCommit("", true)
			HandleError("amending commit", err, true)
			fmt.Println("Successfully amended commit.")
			return
		}

		err := ui.StartAmendInput(repo)
		HandleError("amending commit", err, true)
	},
}

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Soft-reset the last commit, keeping changes staged",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		err := repo.UndoLastCommit()
		HandleError("undoing last commit", err, true)
		fmt.Println("Last commit undone. Changes are still staged.")
	},
}
