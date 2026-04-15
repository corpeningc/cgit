package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

// StatusBar holds the fetched data for the persistent top-line status display.
type StatusBar struct {
	Branch      string
	Ahead       int
	Behind      int
	Clean       bool
	HasUpstream bool
}

// StatusBarMsg is returned by FetchStatusBar.
type StatusBarMsg struct {
	Bar StatusBar
}

// FetchStatusBar fetches branch, ahead/behind, and clean state asynchronously.
func FetchStatusBar(repo *git.GitRepo) tea.Cmd {
	return func() tea.Msg {
		var bar StatusBar

		if branch, err := repo.GetCurrentBranch(); err == nil {
			bar.Branch = branch
		}
		if clean, err := repo.IsClean(); err == nil {
			bar.Clean = clean
		}
		if ahead, behind, err := repo.GetAheadBehind(); err == nil {
			bar.HasUpstream = true
			bar.Ahead = ahead
			bar.Behind = behind
		}

		return StatusBarMsg{Bar: bar}
	}
}

// Render returns the formatted status bar string using style.
func (s StatusBar) Render(style lipgloss.Style) string {
	if s.Branch == "" {
		return ""
	}
	clean := "●"
	if s.Clean {
		clean = "◯"
	}
	if s.HasUpstream && (s.Ahead > 0 || s.Behind > 0) {
		return style.Render(fmt.Sprintf("  %s  ↑%d ↓%d  %s", s.Branch, s.Ahead, s.Behind, clean))
	}
	return style.Render(fmt.Sprintf("  %s  %s", s.Branch, clean))
}
