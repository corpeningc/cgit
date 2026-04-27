package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/corpeningc/cgit/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newBranchCmd)
	rootCmd.AddCommand(branchesCmd)

	switchBranchCmd.Flags().BoolP("remote", "r", false, "Include remote branches in the branch list")
	rootCmd.AddCommand(switchBranchCmd)

	featureCmd.Flags().StringP("origin", "o", "", "The branch to pull latest changes from before creating the feature branch (defaults to repo's primary branch)")
	featureCmd.Flags().StringP("new", "n", "", "The name of the new feature branch")
	featureCmd.Flags().BoolP("close", "c", false, "The name of the branch to close after creating the new feature branch")
	rootCmd.AddCommand(featureCmd)
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
		branchName := ""

		remote, err := cmd.Flags().GetBool("remote")
		HandleError("Getting remote flag", err, true)

		if len(args) == 1 {
			branchName = args[0]
			isClean, err := repo.IsClean()
			HandleError("checking repository status", err, true)

			if !isClean {
				fmt.Println("You need to stash or delete your changes before swapping. Press 'd' to delete changes or 's' to enter a stash name")
				reader := bufio.NewReader(os.Stdin)
				input, err := reader.ReadString('\n')
				HandleError("reading stash name", err, true)

				input = strings.TrimSpace(input)
				var stashName string

				switch input {
				case "d":
					err = repo.FullClean()
					HandleError("deleting changes", err, true)
					fmt.Println("Changes deleted.")
				case "s":
					_, err = reader.Discard(0)
					HandleError("discarding input", err, true)

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
		} else {
			_, err := ui.SwitchBranches(repo, remote)
			HandleError("switching branches", err, true)
		}
	},
}

var branchesCmd = &cobra.Command{
	Use:     "branches",
	Aliases: []string{"br"},
	Short:   "Browse and manage branches in an interactive TUI",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		err := ui.StartBranchManager(repo)
		HandleError("managing branches", err, true)
	},
}

var featureCmd = &cobra.Command{
	Use:     "feature",
	Aliases: []string{"feat"},
	Short:   "Pull latest from main, create and switch to a new feature branch",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
		origin, err := cmd.Flags().GetString("origin")
		if origin == "" {
			origin = repo.GetDefaultBranch()
		}
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
