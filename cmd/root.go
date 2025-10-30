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

func handleError(operation string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s: %v\n", operation, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "cgit",
	Short: "A simplified git workflow tool",
	Long:  "Simplifies common git operations with interactive interfaces",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		_, err := exec.LookPath("git")
		handleError("checking for git installation", err)

		repo := git.New(".")
		_, err = repo.GetCurrentBranch()
		handleError("checking for git repository", err)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(manageCmd)
	manageCmd.Flags().BoolP("staged", "s", false, "Manage Staged files")

	rootCmd.AddCommand(mergeCommand)
	rootCmd.AddCommand(commitAndPushCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(newBranchCmd)

	switchBranchCmd.Flags().BoolP("remote", "r", false, "Include remote branches in the branch list")
	rootCmd.AddCommand(switchBranchCmd)

	rootCmd.AddCommand(stashPopCmd)
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
		files := []git.FileStatus{}

		staged, err := cmd.Flags().GetBool("staged")
		// Get unstaged files only
		repoStatus, err := repo.GetRepositoryStatus()
		handleError("getting repository status", err)

		if !staged {
			files = repoStatus.UnstagedFiles
		} else {
			files = repoStatus.StagedFiles
		}

		if len(files) == 0 {
			fmt.Println("No files to manage.")
			return
		}

		selected, removing, err := ui.SelectFiles(repo, files, staged)
		handleError("selecting files", err)

		if len(selected) == 0 {
			fmt.Println("No files selected.")
			return
		}

		if removing {
			err = repo.RemoveFiles(selected, staged)
			handleError("removing files", err)
			fmt.Printf("Removed %d files.\n", len(selected))
			for _, file := range selected {
				fmt.Printf(" - %s\n", file)
			}
		} else {
			if !staged {
				err = repo.AddFiles(selected)
				handleError("adding files", err)
				fmt.Printf("Added %d files to staging.\n", len(selected))
				for _, file := range selected {
					fmt.Printf(" - %s\n", file)
				}
			}
		}
	},
}

var mergeCommand = &cobra.Command{
	Use:   "merge",
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
	Use:     "commit-and-push",
	Aliases: []string{"cap"},
	Short:   "Commit and push changes",

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
	Use:   "commit",
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
	Use:   "push",
	Short: "Push committed changes to remote",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.Push()
		handleError("pushing changes", err)

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
		handleError("creating branch", err)

		err = repo.SwitchBranch(branchName)
		handleError("switching branch", err)

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
		handleError("checking repository status", err)

		if !isClean {
			fmt.Println("You need to stash or delete your changes before swapping. Press 'd' to delete changes or 's' to enter a stash name")
			reader := bufio.NewReader(os.Stdin)
			// Read s or d input
			input, err := reader.ReadString('\n')
			handleError("reading stash name", err)

			input = strings.TrimSpace(input)
			var stashName string

			switch input {
			case "d":
				err = repo.FullClean()
				handleError("deleting changes", err)
				fmt.Println("Changes deleted.")
				// Proceed to switch branches
			case "s":
				_, err = reader.Discard(0)

				handleError("discarding input", err)
				// Read stash name
				fmt.Print("Enter stash name: ")
				handleError("reading stash name", err)
				stashName, err = reader.ReadString('\n')
				stashName = strings.TrimSpace(stashName)
				if stashName == "" {
					fmt.Println("No stash name provided. Aborting switch.")
					return
				}
				err = repo.Stash(stashName)
				handleError("stashing changes", err)
				fmt.Printf("Changes stashed as '%s'.\n", stashName)
			}
		}

		err = repo.SwitchBranch(branchName)
		handleError("switching branches", err)

		fmt.Printf("Successfully switched to branch '%s'.\n", branchName)
	},
}

var stashPopCmd = &cobra.Command{
	Use:     "stash-pop",
	Aliases: []string{"sp"},
	Short:   "Pop the most recent stash",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.StashPop()
		handleError("popping stash", err)

		fmt.Println("Successfully popped stash.")
	},
}

var fullCleanCmd = &cobra.Command{
	Use:     "full-clean",
	Aliases: []string{"fc"},
	Short:   "Hard reset branch; Clean files and directories",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		err := repo.FullClean()
		handleError("performing full clean", err)

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
		handleError("getting current branch", err)

		if len(args) > 0 {
			branchName = args[0]
		}

		err = repo.PullLatestRemote(branchName)
		handleError("pulling latest changes", err)

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
			handleError("using feature command", fmt.Errorf("either -new or -close flag must be provided"))
		}

		handleError("getting origin flag", err)

		if new {
			branchName, err := cmd.Flags().GetString("new")
			handleError("getting close flag", err)

			err = repo.PullLatestRemote(origin)
			handleError("pulling latest changes", err)

			err = repo.SwitchBranch(origin)
			handleError("switching to origin branch", err)

			err = repo.CreateBranch(branchName)
			handleError("creating feature branch", err)

			fmt.Println("Successfully created and switched to feature branch", branchName)
		} else if close {
			// Merge the branch to the origin branch and delete it
			branchName, err := repo.GetCurrentBranch()
			handleError("getting close flag", err)

			err = repo.SwitchBranch(origin)
			handleError("switching to origin branch", err)
			fmt.Printf("Switching to %s\n", origin)

			err = repo.PullLatestRemote(origin)
			handleError("pulling latest changes", err)

			err = repo.MergeLocalBranch(branchName)
			handleError("closing feature branch", err)
			fmt.Printf("Successfully merged %s into %s\n", branchName, origin)

			err = repo.DeleteBranch(branchName)
			handleError("deleting feature branch\n", err)
			fmt.Printf("Deleting branch %s\n", branchName)

			err = repo.Push()
			handleError("pushing changes", err)
			fmt.Printf("Successfully pushed changes.")
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
		if err != nil {
			handleError("using status command", nil)
		}

		fmt.Printf("Fetching repo status for %s\n", repoStatus.CurrentBranch)

		fmt.Printf("Staged Files: \n")
		for _, file := range repoStatus.StagedFiles {
			fmt.Printf("%s \t %s\n", file.Status, file.Path)
		}

		fmt.Printf("Unstaged Files: \n")
		for _, file := range repoStatus.UnstagedFiles {
			fmt.Printf("%s \t %s\n", file.Status, file.Path)
		}
	},
}
