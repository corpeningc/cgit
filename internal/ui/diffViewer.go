package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type DiffViewerModel struct {
	repo       *git.GitRepo
	filePath   string
	content    string
	viewport   viewport.Model
	ready      bool
	err        error

	// Styles
	titleStyle    lipgloss.Style
	addedStyle    lipgloss.Style
	removedStyle  lipgloss.Style
	contextStyle  lipgloss.Style
	headerStyle   lipgloss.Style
	errorStyle    lipgloss.Style
	helpStyle     lipgloss.Style
}

type diffLoadedMsg struct {
	content string
	err     error
}

func NewDiffViewerModel(repo *git.GitRepo, filePath string) DiffViewerModel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return DiffViewerModel{
		repo:     repo,
		filePath: filePath,
		viewport: vp,

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),

		addedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("40")).
			Background(lipgloss.Color("22")),

		removedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("52")),

		contextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

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
		// Ensure minimum dimensions to prevent rendering issues
		minWidth := 10
		minHeight := 5

		viewportWidth := msg.Width - 4  // More padding for borders
		viewportHeight := msg.Height - headerHeight

		if viewportWidth < minWidth {
			viewportWidth = minWidth
		}
		if viewportHeight < minHeight {
			viewportHeight = minHeight
		}

		if !m.ready {
			m.viewport = viewport.New(viewportWidth, viewportHeight)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
		} else {
			m.viewport.Width = viewportWidth
			m.viewport.Height = viewportHeight
		}

		if m.content != "" {
			// TEMPORARY: Test with raw content bypass
			if strings.Contains(m.filePath, "Cognito") {
				m.viewport.SetContent("RAW CONTENT TEST:\n\n" + m.content)
			} else {
				m.viewport.SetContent(m.formatDiff(m.content))
			}
		}

	case diffLoadedMsg:
		m.content = msg.content
		m.err = msg.err
		if m.ready && m.err == nil {
			// TEMPORARY: Test with raw content bypass
			if strings.Contains(m.filePath, "Cognito") {
				m.viewport.SetContent("RAW CONTENT TEST:\n\n" + m.content)
			} else {
				m.viewport.SetContent(m.formatDiff(m.content))
			}
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

	// Title
	title := m.titleStyle.Render("Diff Viewer - " + m.filePath)
	sections = append(sections, title)

	// Viewport with diff content
	sections = append(sections, m.viewport.View())

	// Help
	help := m.helpStyle.Render("j/k: line by line | d/u: half page | f/b: full page | g/G: top/bottom | esc: back")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m DiffViewerModel) loadDiff() tea.Cmd {
	return func() tea.Msg {
		content, err := m.repo.FileDiff(m.filePath)

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

	// Safety check to prevent issues with very long content
	if len(content) > 500000 { // 500KB limit
		content = content[:500000] + "\n... (content truncated for display)"
	}

	lines := strings.Split(content, "\n")
	var formatted []string

	for _, line := range lines {
		if line == "" {
			formatted = append(formatted, "")
			continue
		}

		// Ensure the line is safe to render (no control characters except newlines)
		safeLine := strings.ReplaceAll(line, "\t", "    ") // Convert tabs to spaces

		var styledLine string
		switch {
		case strings.HasPrefix(safeLine, "+++") || strings.HasPrefix(safeLine, "---"):
			styledLine = m.headerStyle.Render(safeLine)
		case strings.HasPrefix(safeLine, "@@"):
			styledLine = m.headerStyle.Render(safeLine)
		case strings.HasPrefix(safeLine, "+"):
			styledLine = m.addedStyle.Render(safeLine)
		case strings.HasPrefix(safeLine, "-"):
			styledLine = m.removedStyle.Render(safeLine)
		case strings.HasPrefix(safeLine, "diff --git"):
			styledLine = m.headerStyle.Render(safeLine)
		case strings.HasPrefix(safeLine, "index "):
			styledLine = m.headerStyle.Render(safeLine)
		default:
			styledLine = m.contextStyle.Render(safeLine)
		}

		// Fallback: if styled line is empty or same as input, use plain text with prefix
		if styledLine == "" || styledLine == safeLine {
			switch {
			case strings.HasPrefix(safeLine, "+"):
				styledLine = "+ " + safeLine[1:]
			case strings.HasPrefix(safeLine, "-"):
				styledLine = "- " + safeLine[1:]
			case strings.HasPrefix(safeLine, "@@"):
				styledLine = ">> " + safeLine
			default:
				styledLine = "  " + safeLine
			}
		}

		formatted = append(formatted, styledLine)
	}

	result := strings.Join(formatted, "\n")

	// Final safety check - if result is empty, provide fallback
	if strings.TrimSpace(result) == "" {
		return m.contextStyle.Render("Diff content could not be formatted properly.\nRaw content length: " +
			fmt.Sprintf("%d", len(content)) + " characters")
	}

	return result
}

func ShowDiff(repo *git.GitRepo, filePath string) error {
	m := NewDiffViewerModel(repo, filePath)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}