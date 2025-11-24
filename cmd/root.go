package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func HandleError(operation string, err error, close bool) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s: %v\n", operation, err)
		if close {
			os.Exit(1)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "cgit",
	Short: "A simplified git workflow tool",
	Long:  "Simplifies common git operations with interactive interfaces",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip validation for shell command
		if cmd.Name() == "shell" {
			return
		}

		_, err := exec.LookPath("git")
		HandleError("checking for git installation", err, true)

		repo := git.New(".")
		_, err = repo.GetCurrentBranch()
		HandleError("checking for git repository", err, true)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set the Run function after initialization to avoid circular dependency
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		// If no subcommand provided, launch interactive shell
		runInteractiveShell()
	}

	rootCmd.AddCommand(shellCmd)

	rootCmd.AddCommand(manageCmd)
	manageCmd.Flags().BoolP("staged", "s", false, "Manage Staged files")

	rootCmd.AddCommand(mergeCommand)
	rootCmd.AddCommand(commitAndPushCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(newBranchCmd)

	switchBranchCmd.Flags().BoolP("remote", "r", false, "Include remote branches in the branch list")
	rootCmd.AddCommand(switchBranchCmd)

	rootCmd.AddCommand(popCmd)
	rootCmd.AddCommand(storeCmd)

	rootCmd.AddCommand(fullCleanCmd)
	rootCmd.AddCommand(pullCmd)

	featureCmd.Flags().StringP("origin", "o", "main", "The branch to pull latest changes from before creating the feature branch")
	featureCmd.Flags().StringP("new", "n", "", "The name of the new feature branch")
	featureCmd.Flags().BoolP("close", "c", false, "The name of the branch to close after creating the new feature branch")
	rootCmd.AddCommand(featureCmd)

	rootCmd.AddCommand(statusCommand)
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
		// Get unstaged files only
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

var mergeCommand = &cobra.Command{
	Use:   "merge",
	Short: "Fetch latest remote changes and merge",
	Run: func(cmd *cobra.Command, args []string) {
		branch := args[0]
		repo := git.New(".")

		err := repo.MergeLatest(branch)
		HandleError("merging latest changes", err, true)

		fmt.Println("Successfully merged latest changes.")
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

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit staged changes with a message",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		commitMsg := args[0]
		err := repo.Commit(commitMsg)
		HandleError("committing changes", err, true)

		fmt.Println("Successfully committed changes.")
	},
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push committed changes to remote",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.Push()
		HandleError("pushing changes", err, true)

		fmt.Println("Successfully pushed changes.")
	},
}

var newBranchCmd = &cobra.Command{
	Use:     "new-branch",
	Aliases: []string{"nb"},
	Short:   "Create and switch to a new branch",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		branchName := args[0]
		err := repo.CreateBranch(branchName)
		HandleError("creating branch", err, true)

		err = repo.SwitchBranch(branchName)
		HandleError("switching branch", err, true)

		fmt.Printf("Successfully created and switched to branch '%s'.\n", branchName)
	},
}

var switchBranchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"sw"},
	Short:   "Switch to an existing branch",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		repo := git.New(".")
		remote, err := cmd.Flags().GetBool("remote")

		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		branches, err := repo.GetAllBranches(remote)

		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return branches, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		branchName := args[0]

		// Check if working directory is clean
		isClean, err := repo.IsClean()
		HandleError("checking repository status", err, true)

		if !isClean {
			fmt.Println("You need to stash or delete your changes before swapping. Press 'd' to delete changes or 's' to enter a stash name")
			reader := bufio.NewReader(os.Stdin)
			// Read s or d input
			input, err := reader.ReadString('\n')
			HandleError("reading stash name", err, true)

			input = strings.TrimSpace(input)
			var stashName string

			switch input {
			case "d":
				err = repo.FullClean()
				HandleError("deleting changes", err, true)
				fmt.Println("Changes deleted.")
				// Proceed to switch branches
			case "s":
				_, err = reader.Discard(0)
				HandleError("discarding input", err, true)

				// Read stash name
				fmt.Print("Enter stash name: ")
				stashName, err = reader.ReadString('\n')
				HandleError("reading stash name", err, true)

				stashName = strings.TrimSpace(stashName)
				if stashName == "" {
					fmt.Println("No stash name provided. Aborting switch.")
					return
				}

				err = repo.Stash(stashName)
				HandleError("stashing changes", err, true)

				fmt.Printf("Changes stashed as '%s'.\n", stashName)
			}
		}

		err = repo.SwitchBranch(branchName)
		HandleError("switching branches", err, true)

		fmt.Printf("Successfully switched to branch '%s'.\n", branchName)
	},
}

var popCmd = &cobra.Command{
	Use:   "pop",
	Short: "Pop the most recent stash",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.StashPop()
		HandleError("popping stash", err, true)

		fmt.Println("Successfully popped stash.")
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

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull latest changes from remote",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		// If no branch provided, use current branch
		branchName, err := repo.GetCurrentBranch()
		HandleError("getting current branch", err, true)

		if len(args) > 0 {
			branchName = args[0]
		}

		err = repo.PullLatestRemote(branchName)
		HandleError("pulling latest changes", err, true)

		fmt.Println("Successfully pulled latest changes for branch", branchName)
	},
}

var featureCmd = &cobra.Command{
	Use:     "feature",
	Aliases: []string{"feat"},
	Short:   "Pull latest from main, create and switch to a new feature branch",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		origin, err := cmd.Flags().GetString("origin")
		new := cmd.Flags().Changed("new")
		close := cmd.Flags().Changed("close")

		if !new && !close {
			HandleError("using feature command", fmt.Errorf("either -new or -close flag must be provided"), true)
		}

		HandleError("getting origin flag", err, true)

		if new {
			branchName, err := cmd.Flags().GetString("new")
			HandleError("getting new flag", err, true)

			err = repo.PullLatestRemote(origin)
			HandleError("pulling latest changes", err, true)

			err = repo.SwitchBranch(origin)
			HandleError("switching to origin branch", err, true)

			err = repo.CreateBranch(branchName)
			HandleError("creating feature branch", err, true)

			fmt.Println("Successfully created and switched to feature branch", branchName)
		} else if close {
			// Merge the branch to the origin branch and delete it
			branchName, err := repo.GetCurrentBranch()
			HandleError("getting close flag", err, true)

			err = repo.SwitchBranch(origin)
			HandleError("switching to origin branch", err, true)
			fmt.Printf("Switching to %s\n", origin)

			err = repo.PullLatestRemote(origin)
			HandleError("pulling latest changes", err, true)

			err = repo.MergeLocalBranch(branchName)
			HandleError("closing feature branch", err, true)
			fmt.Printf("Successfully merged %s into %s\n", branchName, origin)

			err = repo.DeleteBranch(branchName)
			HandleError("deleting feature branch\n", err, true)
			fmt.Printf("Deleting branch %s\n", branchName)

			err = repo.Push()
			HandleError("pushing changes", err, true)
			fmt.Println("Successfully pushed changes.")
		}
	},
}

var statusCommand = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st"},
	Short:   "Get the status of the current branch",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		repoStatus, err := repo.GetRepositoryStatus()
		HandleError("using status command", err, true)

		fmt.Printf("Fetching repo status for %s\n\n", repoStatus.CurrentBranch)

		if len(repoStatus.StagedFiles) > 0 {
			fmt.Printf("Staged Changes: \n")
			for _, file := range repoStatus.StagedFiles {
				fmt.Printf("%s \t %s\n", file.Status, file.Path)
			}
		} else {
			fmt.Println("No staged changes")
		}

		fmt.Println()

		if len(repoStatus.UnstagedFiles) > 0 {
			fmt.Printf("Unstaged Files: \n")
			for _, file := range repoStatus.UnstagedFiles {
				fmt.Printf("%s \t %s\n", file.Status, file.Path)
			}
		} else {
			fmt.Println("No unstaged changes")
		}

		fmt.Println()

	},
}
