package cmd

import (
	"fmt"
	"os"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func handleError(operation string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s: %v\n", operation, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use: "cgit",
	Short: "A simplified git workflow tool",
	Long: "Simplifies common git operations with interactive interfaces",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(mergeCommand)
	rootCmd.AddCommand(commitAndPushCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(newBranchCmd)
	rootCmd.AddCommand(statusCmd)
}

var addCmd = &cobra.Command{
	Use: "add",
	Short: "Interactively add files to staging",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		files, err := repo.GetModifiedFiles()
		handleError("getting modified files", err)

		if len(files) == 0 {
			fmt.Println("No modified files to add.")
			return
		}

		selected, err := ui.SelectFiles(files)
		handleError("selecting files", err)

		if (len(selected) == 0) {
			fmt.Println("No files selected.")
		}

		err = repo.AddFiles(selected)
		handleError("adding files", err)

		fmt.Printf("Added %d files to staging.\n", len(selected))
		for _, file := range selected {
			fmt.Printf(" - %s\n", file)
		}
	},
}

var mergeCommand = &cobra.Command{
	Use: "merge",
	Short: "Fetch latest remote changes and merge",
	Run: func(cmd *cobra.Command, args []string) {
		branch := args[0]
		repo := git.New(".")

		err := repo.MergeLatest(branch)
		handleError("merging latest changes", err)

		fmt.Println("Successfully merged latest changes.")
	},
}

var commitAndPushCmd = &cobra.Command{
	Use: "cap",
	Short: "Commit and push changes",

	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")


		commitMsg := args[0]
		err := repo.Commit(commitMsg)
		handleError("committing changes", err)
		
		err = repo.Push()
		handleError("pushing changes", err)

		fmt.Println("Successfully committed and pushed changes.")
	},
}

var commitCmd = &cobra.Command{
	Use: "commit",
	Short: "Commit staged changes with a message",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		commitMsg := args[0]
		err := repo.Commit(commitMsg)
		handleError("committing changes", err)

		fmt.Println("Successfully committed changes.")
	},
}

var pushCmd = &cobra.Command{
	Use: "push",
	Short: "Push committed changes to remote",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.Push()
		handleError("pushing changes", err)

		fmt.Println("Successfully pushed changes.")
	},
}

var newBranchCmd = &cobra.Command {
	Use: "nb",
	Short: "Create and switch to a new branch",
	Run: func (cmd *cobra.Command, args []string) {
		repo := git.New(".")

		branchName := args[0]
		err := repo.CreateBranch(branchName)
		err = repo.SwitchBranch(branchName)
		handleError("creating and switching to new branch", err)

		fmt.Printf("Successfully created and switched to branch '%s'.\n", branchName)
	},
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st"},
	Short:   "Interactive git status with staging capabilities",
	Long:    "Launch an interactive TUI for viewing repository status, staging/unstaging files, and committing changes with vim-style navigation",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := ui.StartStatusTUI(repo)
		handleError("starting status TUI", err)
	},
}