# Building a Git CLI Tool in Go - Complete Development Guide

## Project Overview
Build a CLI tool in Go that simplifies git operations with features like:
- Interactive file selection with checkboxes
- Simplified merging of main into current branch
- Better UX for common git workflows

## Prerequisites
- Go 1.19+ installed
- Basic git repository for testing
- Terminal that supports ANSI escape codes

## Project Structure
```
git-simplify/
├── cmd/
│   └── root.go          # CLI command definitions
├── internal/
│   ├── git/
│   │   └── operations.go # Git command wrappers
│   ├── ui/
│   │   └── interactive.go # TUI components
│   └── utils/
│       └── helpers.go    # Utility functions
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## Required Dependencies

### Core CLI Framework
```bash
go mod init git-simplify
go get github.com/spf13/cobra@latest
```

### TUI Libraries (choose one)
```bash
# Option 1: Bubble Tea (Recommended - modern, powerful)
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/huh@latest  # for forms/checkboxes

# Option 2: Survey (Simple, good for basic prompts)
go get github.com/AlecAivazis/survey/v2@latest

# Option 3: go-prompt (Autocomplete focused)
go get github.com/c-bata/go-prompt@latest
```

### Git Integration Options
```bash
# Option 1: Shell out to git (simpler)
# No additional dependencies needed - use os/exec

# Option 2: Pure Go git library (more control)
go get github.com/go-git/go-git/v5@latest
```

### Additional Utilities
```bash
go get github.com/fatih/color@latest      # Colored output
go get github.com/spf13/viper@latest      # Configuration management
```

## Core Features Implementation

### 1. Basic CLI Structure with Cobra

**main.go**
```go
package main

import (
    "fmt"
    "os"
    "git-simplify/cmd"
)

func main() {
    if err := cmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**cmd/root.go**
```go
package cmd

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "git-simplify",
    Short: "A simplified git workflow tool",
    Long:  "Simplifies common git operations with interactive interfaces",
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.AddCommand(addCmd)
    rootCmd.AddCommand(syncCmd)
    rootCmd.AddCommand(mergeMainCmd)
}

// Interactive file selection command
var addCmd = &cobra.Command{
    Use:   "add",
    Short: "Interactively select files to add",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation here
    },
}

// Merge main into current branch
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Fetch latest main and merge into current branch",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation here
    },
}
```

### 2. Git Operations Wrapper

**internal/git/operations.go**
```go
package git

import (
    "os/exec"
    "strings"
    "bufio"
    "errors"
)

type GitRepo struct {
    WorkDir string
}

func New(workDir string) *GitRepo {
    return &GitRepo{WorkDir: workDir}
}

// Get modified files
func (g *GitRepo) GetModifiedFiles() ([]string, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = g.WorkDir
    
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    var files []string
    scanner := bufio.NewScanner(strings.NewReader(string(output)))
    
    for scanner.Scan() {
        line := scanner.Text()
        if len(line) >= 3 {
            files = append(files, strings.TrimSpace(line[2:]))
        }
    }
    
    return files, nil
}

// Add files to staging
func (g *GitRepo) AddFiles(files []string) error {
    if len(files) == 0 {
        return nil
    }
    
    args := append([]string{"add"}, files...)
    cmd := exec.Command("git", args...)
    cmd.Dir = g.WorkDir
    
    return cmd.Run()
}

// Get current branch
func (g *GitRepo) GetCurrentBranch() (string, error) {
    cmd := exec.Command("git", "branch", "--show-current")
    cmd.Dir = g.WorkDir
    
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    
    return strings.TrimSpace(string(output)), nil
}

// Fetch latest changes
func (g *GitRepo) Fetch() error {
    cmd := exec.Command("git", "fetch", "origin")
    cmd.Dir = g.WorkDir
    return cmd.Run()
}

// Merge main into current branch
func (g *GitRepo) MergeMain() error {
    currentBranch, err := g.GetCurrentBranch()
    if err != nil {
        return err
    }
    
    if currentBranch == "main" || currentBranch == "master" {
        return errors.New("already on main branch")
    }
    
    // Try to merge main
    cmd := exec.Command("git", "merge", "origin/main")
    cmd.Dir = g.WorkDir
    
    return cmd.Run()
}

// Check if repository is clean
func (g *GitRepo) IsClean() (bool, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = g.WorkDir
    
    output, err := cmd.Output()
    if err != nil {
        return false, err
    }
    
    return len(strings.TrimSpace(string(output))) == 0, nil
}
```

### 3. Interactive UI Components

**Using Bubble Tea + Huh (Recommended)**

**internal/ui/filepicker.go**
```go
package ui

import (
    "fmt"
    "github.com/charmbracelet/huh"
)

func SelectFiles(files []string) ([]string, error) {
    var selectedFiles []string
    
    // Create options for the multi-select
    var options []huh.Option[string]
    for _, file := range files {
        options = append(options, huh.NewOption(file, file))
    }
    
    // Create the form
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewMultiSelect[string]().
                Title("Select files to add:").
                Options(options...).
                Value(&selectedFiles),
        ),
    )
    
    err := form.Run()
    if err != nil {
        return nil, err
    }
    
    return selectedFiles, nil
}
```

**Alternative with Survey**

**internal/ui/interactive.go**
```go
package ui

import (
    "github.com/AlecAivazis/survey/v2"
)

func SelectFilesWithSurvey(files []string) ([]string, error) {
    var selectedFiles []string
    
    prompt := &survey.MultiSelect{
        Message: "Select files to add:",
        Options: files,
    }
    
    err := survey.AskOne(prompt, &selectedFiles)
    return selectedFiles, err
}

func ConfirmAction(message string) (bool, error) {
    confirm := false
    prompt := &survey.Confirm{
        Message: message,
    }
    
    err := survey.AskOne(prompt, &confirm)
    return confirm, err
}
```

### 4. Command Implementations

**cmd/add.go**
```go
package cmd

import (
    "fmt"
    "os"
    "git-simplify/internal/git"
    "git-simplify/internal/ui"
    "github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
    Use:   "add",
    Short: "Interactively select files to add",
    Run: func(cmd *cobra.Command, args []string) {
        // Initialize git repo
        repo := git.New(".")
        
        // Get modified files
        files, err := repo.GetModifiedFiles()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error getting modified files: %v\n", err)
            os.Exit(1)
        }
        
        if len(files) == 0 {
            fmt.Println("No modified files found")
            return
        }
        
        // Show interactive selection
        selected, err := ui.SelectFiles(files)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error in file selection: %v\n", err)
            os.Exit(1)
        }
        
        if len(selected) == 0 {
            fmt.Println("No files selected")
            return
        }
        
        // Add selected files
        err = repo.AddFiles(selected)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error adding files: %v\n", err)
            os.Exit(1)
        }
        
        fmt.Printf("Added %d files to staging\n", len(selected))
        for _, file := range selected {
            fmt.Printf("  ✓ %s\n", file)
        }
    },
}
```

**cmd/sync.go**
```go
package cmd

import (
    "fmt"
    "os"
    "git-simplify/internal/git"
    "git-simplify/internal/ui"
    "github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Fetch latest main and merge into current branch",
    Run: func(cmd *cobra.Command, args []string) {
        repo := git.New(".")
        
        // Check if repo is clean
        clean, err := repo.IsClean()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error checking repo status: %v\n", err)
            os.Exit(1)
        }
        
        if !clean {
            fmt.Println("Repository has uncommitted changes. Please commit or stash them first.")
            os.Exit(1)
        }
        
        currentBranch, err := repo.GetCurrentBranch()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error getting current branch: %v\n", err)
            os.Exit(1)
        }
        
        fmt.Printf("Current branch: %s\n", currentBranch)
        
        // Confirm action
        confirm, err := ui.ConfirmAction(fmt.Sprintf("Fetch latest changes and merge main into %s?", currentBranch))
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error in confirmation: %v\n", err)
            os.Exit(1)
        }
        
        if !confirm {
            fmt.Println("Operation cancelled")
            return
        }
        
        // Fetch latest changes
        fmt.Println("Fetching latest changes...")
        err = repo.Fetch()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error fetching: %v\n", err)
            os.Exit(1)
        }
        
        // Merge main
        fmt.Println("Merging main...")
        err = repo.MergeMain()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error merging main: %v\n", err)
            fmt.Println("You may need to resolve conflicts manually")
            os.Exit(1)
        }
        
        fmt.Println("✓ Successfully synced with main")
    },
}
```

## Development Workflow

### 1. Setup
```bash
# Create new project
mkdir git-simplify
cd git-simplify
go mod init git-simplify

# Install dependencies (choose your preferred TUI library)
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/huh@latest
go get github.com/fatih/color@latest
```

### 2. Build and Test
```bash
# Build the tool
go build -o git-simplify .

# Test in a git repository
cd /path/to/test/repo
/path/to/git-simplify/git-simplify add
/path/to/git-simplify/git-simplify sync
```

### 3. Install Globally
```bash
# Build and install
go install .

# Or copy to PATH
go build -o git-simplify .
sudo cp git-simplify /usr/local/bin/
```

## Additional Features to Consider

### Configuration File Support
```go
// Using Viper for configuration
type Config struct {
    DefaultBranch string `mapstructure:"default_branch"`
    AutoFetch     bool   `mapstructure:"auto_fetch"`
    ColorOutput   bool   `mapstructure:"color_output"`
}
```

### Enhanced Git Operations
- Stash management
- Branch switching with fuzzy search
- Commit with template messages
- Interactive rebase helpers
- Conflict resolution assistance

### Better Error Handling
- Detect merge conflicts and provide guidance
- Check for unstaged changes before operations
- Validate git repository before running commands

### Testing
```go
// Example test structure
func TestGetModifiedFiles(t *testing.T) {
    // Create temporary git repo
    // Add some files
    // Test the function
}
```

## Offline Development Tips

1. **Create test repositories locally** with various scenarios (clean, dirty, conflicts)
2. **Mock git commands** for unit testing without requiring actual git operations
3. **Use example data** for UI development and testing
4. **Build incrementally** - start with basic file selection, then add merge functionality
5. **Test edge cases** like empty repositories, detached HEAD, merge conflicts

## Packaging and Distribution

### Cross-platform Builds
```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o git-simplify-linux .
GOOS=windows GOARCH=amd64 go build -o git-simplify-windows.exe .
GOOS=darwin GOARCH=amd64 go build -o git-simplify-macos .
```

### Installation Methods
- Direct binary download
- Homebrew formula (for macOS)
- Go install command
- Package managers (APT, RPM)

This guide should give you everything you need to build your git CLI tool during your flight. Start with the basic structure and file selection feature, then expand to the merge functionality. Good luck with your project!