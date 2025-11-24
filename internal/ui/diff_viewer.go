package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type DiffViewerModel struct {
	repo     *git.GitRepo
	filePath string
	content  string
	viewport viewport.Model
	ready    bool
	err      error

	staged bool

	// Styles
	titleStyle   lipgloss.Style
	addedStyle   lipgloss.Style
	removedStyle lipgloss.Style
	contextStyle lipgloss.Style
	headerStyle  lipgloss.Style
	errorStyle   lipgloss.Style
	helpStyle    lipgloss.Style
}

type diffLoadedMsg struct {
	content string
	err     error
}

func NewDiffViewerModel(repo *git.GitRepo, filePath string) DiffViewerModel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	return DiffViewerModel{
		repo:     repo,
		filePath: filePath,
		viewport: vp,

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")),

		addedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")),

		removedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),

		contextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
	}
}

func (m DiffViewerModel) Init() tea.Cmd {
	return m.loadDiff()
}

func (m DiffViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 4 // Title + help + borders
		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, msg.Height-headerHeight)
			m.viewport.Style = lipgloss.NewStyle()
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = msg.Height - headerHeight
		}

		if m.content != "" {
			m.viewport.SetContent(m.formatDiff(m.content))
		}

	case diffLoadedMsg:
		m.content = msg.content
		m.err = msg.err
		if m.ready && m.err == nil {
			formatted := m.formatDiff(m.content)
			m.viewport.SetContent(formatted)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j", "down":
			m.viewport.ScrollDown(1)

		case "k", "up":
			m.viewport.ScrollUp(1)

		case "d", "ctrl+d":
			m.viewport.HalfPageDown()

		case "u", "ctrl+u":
			m.viewport.HalfPageUp()

		case "f", "pgdn":
			m.viewport.PageDown()

		case "b", "pgup":
			m.viewport.PageUp()

		case "g", "home":
			m.viewport.GotoTop()

		case "G", "end":
			m.viewport.GotoBottom()
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiffViewerModel) View() string {
	if m.err != nil {
		var sections []string
		title := m.titleStyle.Render("Diff Viewer - " + m.filePath)
		sections = append(sections, title)
		sections = append(sections, "")
		sections = append(sections, m.errorStyle.Render("Error loading diff: "+m.err.Error()))
		sections = append(sections, "")
		help := m.helpStyle.Render("esc: back")
		sections = append(sections, help)
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	if !m.ready {
		return "Loading diff..."
	}

	var sections []string

	title := m.titleStyle.Render("Diff Viewer - " + m.filePath)
	sections = append(sections, title)

	sections = append(sections, m.viewport.View())

	help := m.helpStyle.Render("j/k: line by line | d/u: half page | f/b: full page | g/G: top/bottom | esc: back")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m DiffViewerModel) loadDiff() tea.Cmd {
	return func() tea.Msg {
		content, err := m.repo.FileDiff(m.filePath, m.staged)
		return diffLoadedMsg{
			content: content,
			err:     err,
		}
	}
}

func (m DiffViewerModel) formatDiff(content string) string {
	if content == "" {
		return m.contextStyle.Render("No differences found for this file.")
	}

	// Return raw content - git diff already has ANSI colors
	return content
}

func ShowDiff(repo *git.GitRepo, filePath string) error {
	m := NewDiffViewerModel(repo, filePath)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
