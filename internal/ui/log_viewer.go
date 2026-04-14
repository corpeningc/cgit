package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LogViewerModel struct {
	content  string
	viewport viewport.Model
	ready    bool

	titleStyle lipgloss.Style
	helpStyle  lipgloss.Style
}

func NewLogViewerModel(content string) LogViewerModel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	return LogViewerModel{
		content:  content,
		viewport: vp,

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
	}
}

func (m LogViewerModel) Init() tea.Cmd {
	return nil
}

func (m LogViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 3
		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, msg.Height-headerHeight)
			m.viewport.Style = lipgloss.NewStyle()
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = msg.Height - headerHeight
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

func (m LogViewerModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	title := m.titleStyle.Render("Git Log")
	help := m.helpStyle.Render("j/k: scroll | d/u: half page | f/b: full page | g/G: top/bottom | q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, m.viewport.View(), help)
}

func StartLogViewer(content string) error {
	m := NewLogViewerModel(content)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
