package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/corpeningc/cgit/internal/git"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive cgit shell",
	Long:  "Launch an interactive shell for running cgit commands without repeating 'cgit' prefix",
	Run: func(cmd *cobra.Command, args []string) {
		runInteractiveShell()
	},
}

func runInteractiveShell() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	// Load command history
	historyFile := getHistoryFilePath()
	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	// Setup tab completion for command names
	line.SetCompleter(func(line string) (c []string) {
		commands := getCommandNames()
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, strings.ToLower(line)) {
				c = append(c, cmd)
			}
		}
		return
	})

	fmt.Println("cgit interactive shell. Type 'exit' or press Ctrl+D to quit.")
	fmt.Println("Type 'help' to see available commands.")

	for {
		// Get current branch for prompt
		repo := git.New(".")
		branch, err := repo.GetCurrentBranch()
		if err != nil {
			branch = "unknown"
		}

		prompt := fmt.Sprintf("[%s]> ", branch)
		input, err := line.Prompt(prompt)

		if err != nil {
			// EOF or error (Ctrl+D)
			fmt.Println()
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Add to history
		line.AppendHistory(input)

		// Handle special shell commands
		if handleSpecialCommand(input) {
			continue
		}

		// Handle help command separately to avoid initialization cycle
		if strings.ToLower(input) == "help" {
			rootCmd.Help()
			continue
		}

		// Execute the command through Cobra
		executeCommand(input)
	}

	// Save history on exit
	if f, err := os.Create(historyFile); err == nil {
		line.WriteHistory(f)
		f.Close()
	}
}

func handleSpecialCommand(input string) bool {
	switch strings.ToLower(input) {
	case "exit", "quit":
		fmt.Println("Goodbye!")
		os.Exit(0)
		return true
	case "clear", "cls":
		fmt.Print("\033[H\033[2J")
		return true
	case "help":
		// Pass the rootCmd as parameter instead of referencing directly
		return true
	}
	return false
}

func executeCommand(input string) {
	// Parse input into command and args
	parts := parseCommandLine(input)
	if len(parts) == 0 {
		return
	}

	// Reset rootCmd args and execute
	rootCmd.SetArgs(parts)

	// Capture the command execution
	// We need to handle errors differently in shell mode - don't exit
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	// Reset args for next command
	rootCmd.SetArgs([]string{})
}

func parseCommandLine(input string) []string {
	// Simple parsing - split on spaces but respect quotes
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, char := range input {
		switch {
		case (char == '"' || char == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = char
		case char == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0
		case char == ' ' && !inQuotes:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func getCommandNames() []string {
	var names []string
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "shell" {
			continue
		}
		names = append(names, cmd.Name())
	}
	return names
}

func getHistoryFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".cgit_history"
	}
	return filepath.Join(homeDir, ".cgit_history")
}
