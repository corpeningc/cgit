package cmd

import (
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
	// Add sub commands here
}

var addCmd = &cobra.Command{
	Use: "add",
	Short: "Interactively add files to staging",
	Run: func(cmd *cobra.Command, args []string) {
		// Implementation here
	},
}

var mergeCommand = &cobra.Command{
	Use: "merge",
	Short: "Fetch latest remote changes and merge",
	Run: func(cmd *cobra.Command, args []string) {
		// Implement
	},
}