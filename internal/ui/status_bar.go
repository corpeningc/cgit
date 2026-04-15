package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type StatusBar struct {
	Branch      string
	Ahead       int
	Behind      int
	Clean       bool
	HasUpstream bool
}

type StatusBarMsg struct {
	Bar StatusBar
}

func FetchStatusBar(repo *git.GitRepo) tea.Cmd {
	return func() tea.Msg {
		bar := StatusBar{}

		branch, err := repo.GetCurrentBranch()
		if err == nil {
			bar.Branch = branch
		}

		clean, err := repo.IsClean()
		if err == nil {
			bar.Clean = clean
		}

		ahead, behind, err := repo.GetAheadBehind()
		if err == nil {
			bar.HasUpstream = true
			bar.Ahead = ahead
			bar.Behind = behind
		}

		return StatusBarMsg{Bar: bar}
	}
}

func (s StatusBar) Render(style lipgloss.Style) string {
	if s.Branch == "" {
		return ""
	}

	clean := "●"
	if s.Clean {
		clean = "◯"
	}

	if s.HasUpstream && (s.Ahead > 0 || s.Behind > 0) {
		return style.Render(fmt.Sprintf("  %s  ↑%d ↓%d  %s  ", s.Branch, s.Ahead, s.Behind, clean))
	}
	return style.Render(fmt.Sprintf("  %s  %s  ", s.Branch, clean))
}
