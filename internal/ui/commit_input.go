package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type CommitInputModel struct {
	repo      *git.GitRepo
	textInput textinput.Model
	committed bool
	amend     bool
	err       error

	// When true, the model is embedded inside another TUI and must not call
	// tea.Quit on its own — the parent observes committed/canceled and
	// transitions away from the modal itself.
	embedded bool
	canceled bool

	// Styles
	titleStyle lipgloss.Style
	errorStyle lipgloss.Style
	helpStyle  lipgloss.Style
}

type CommitCompleteMsg struct {
	Success bool
	Err     error
}

func NewCommitInputModel(repo *git.GitRepo) CommitInputModel {
	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 50

	return CommitInputModel{
		repo:       repo,
		textInput:  ti,
		titleStyle: TitlePinkStyle,
		errorStyle: ErrorStyle,
		helpStyle:  HelpStyle,
	}
}

func (m CommitInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CommitInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.embedded {
				m.canceled = true
				return m, nil
			}
			return m, tea.Quit

		case "enter":
			message := m.textInput.Value()
			if message == "" {
				return m, nil
			}
			return m, m.commitWithMessage(message)

		default:
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

	case CommitCompleteMsg:
		m.committed = true
		m.err = msg.Err
		if m.embedded {
			return m, nil
		}
		if msg.Success {
			return m, tea.Quit
		}

	default:
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CommitInputModel) View() string {
	if m.committed {
		if m.err != nil {
			return m.errorStyle.Render(fmt.Sprintf("Commit failed: %v", m.err)) + "\n"
		}
		return StagedStyle.Render("Commit successful!") + "\n"
	}

	var sections []string

	// Title
	titleText := "Commit Changes"
	helpText := "enter: commit | esc: cancel"
	if m.amend {
		titleText = "Amend Last Commit"
		helpText = "enter: amend | esc: cancel"
	}
	sections = append(sections, m.titleStyle.Render(titleText))
	sections = append(sections, "")

	// Input
	sections = append(sections, m.textInput.View())
	sections = append(sections, "")

	// Help
	help := m.helpStyle.Render(helpText)
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m CommitInputModel) commitWithMessage(message string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if m.amend {
			err = m.repo.AmendCommit(message, false)
		} else {
			err = m.repo.Commit(message)
		}
		return CommitCompleteMsg{
			Success: err == nil,
			Err:     err,
		}
	}
}

func StartCommitInput(repo *git.GitRepo) error {
	m := NewCommitInputModel(repo)
	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return err
	}

	if finalModel, ok := model.(CommitInputModel); ok {
		return finalModel.err
	}

	return nil
}

func StartAmendInput(repo *git.GitRepo) error {
	lastMsg, err := repo.GetLastCommitMessage()
	if err != nil {
		return err
	}

	m := NewCommitInputModel(repo)
	m.textInput.SetValue(lastMsg)
	m.textInput.CursorEnd()
	m.amend = true

	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return err
	}

	if finalModel, ok := model.(CommitInputModel); ok {
		return finalModel.err
	}

	return nil
}

