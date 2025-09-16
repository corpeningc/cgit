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
	err       error
	
	// Styles
	titleStyle  lipgloss.Style
	errorStyle  lipgloss.Style
	helpStyle   lipgloss.Style
}

type commitCompleteMsg struct {
	success bool
	error   error
}

func NewCommitInputModel(repo *git.GitRepo) CommitInputModel {
	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 50
	
	return CommitInputModel{
		repo:      repo,
		textInput: ti,
		
		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
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
		
	case commitCompleteMsg:
		m.committed = true
		m.err = msg.error
		if msg.success {
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
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("Commit successful!") + "\n"
	}
	
	var sections []string
	
	// Title
	title := m.titleStyle.Render("Commit Changes")
	sections = append(sections, title)
	sections = append(sections, "")
	
	// Input
	sections = append(sections, m.textInput.View())
	sections = append(sections, "")
	
	// Help
	help := m.helpStyle.Render("enter: commit | esc: cancel")
	sections = append(sections, help)
	
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m CommitInputModel) commitWithMessage(message string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Commit(message)
		return commitCompleteMsg{
			success: err == nil,
			error:   err,
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
	
	// Check if commit was successful
	if finalModel, ok := model.(CommitInputModel); ok {
		return finalModel.err
	}
	
	return nil
}