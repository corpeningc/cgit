package cmd

import (
	"fmt"
	"os"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

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
}

var addCmd = &cobra.Command{
	Use: "add",
	Short: "Interactively add files to staging",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		files, err := repo.GetModifiedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting modified files: %v\n", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Println("No modified files to add.")
			return
		}

		selected, err := ui.SelectFiles(files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error selecting files: %v\n", err)
			os.Exit(1)
		}

		if (len(selected) == 0) {
			fmt.Println("No files selected.")
		}

		err = repo.AddFiles(selected)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding files: %v\n", err)
			os.Exit(1)
		}

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
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error merging latest changes: %v\n", err)
			os.Exit(1)
		}

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

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error committing changes: %v\n", err)
			os.Exit(1)
		}
		
		err = repo.Push()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error pushing changes: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Successfully committed and pushed changes.")
	},
}