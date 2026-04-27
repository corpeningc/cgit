package cmd

import (
	"fmt"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	pushCmd.Flags().BoolP("force-with-lease", "f", false, "Force push with lease (safer force push)")
	pushCmd.Flags().BoolP("set-upstream", "u", false, "Set upstream tracking for current branch")
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(mergeCommand)
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push committed changes to remote",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")

		force, _ := cmd.Flags().GetBool("force-with-lease")
		upstream, _ := cmd.Flags().GetBool("set-upstream")

		err := repo.PushWithOptions(git.PushOptions{
			ForceWithLease: force,
			SetUpstream:    upstream,
		})
		HandleError("pushing changes", err, true)

		fmt.Println("Successfully pushed changes.")
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull latest changes from remote",
	Run: func(cmd *cobra.Command, args []string) {
		repo := git.New(".")
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
