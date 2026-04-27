package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/spf13/cobra"
)

// HandleError prints a styled error and optionally exits.
func HandleError(operation string, err error, close bool) {
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		fmt.Fprintf(os.Stderr, "\033[31;1m✗\033[0m %s: %s\n", operation, msg)
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
	// If no subcommand provided, launch interactive shell.
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runInteractiveShell()
	}
	rootCmd.AddCommand(shellCmd)
}
