package cmd

import (
	"fmt"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	manageCmd.Flags().BoolP("staged", "s", false, "Manage Staged files")
	rootCmd.AddCommand(manageCmd)
}

var manageCmd = &cobra.Command{
	Use:     "manage",
	Aliases: []string{"m"},
	Short:   "Interactively manage files with search support",
	Long: "Launch an interactive file picker for selecting and staging/restoring files with fuzzy search capabilities. " +
		"Use /: to search, space: to select files, c: to stage selected files, and r to restore selected files.",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		staged, err := cmd.Flags().GetBool("staged")
		HandleError("getting staged flag", err, true)

		repoStatus, err := repo.GetRepositoryStatus()
		HandleError("getting repository status", err, true)

		if len(repoStatus.StagedFiles) == 0 && len(repoStatus.UnstagedFiles) == 0 {
			fmt.Println("No files to manage.")
			return
		}

		_, _, err = ui.SelectFiles(repo, repoStatus.StagedFiles, repoStatus.UnstagedFiles, staged)
		HandleError("selecting files", err, true)
	},
}
