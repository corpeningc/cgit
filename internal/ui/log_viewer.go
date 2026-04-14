package ui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

var hashRegex = regexp.MustCompile(`\b([0-9a-f]{7,12})\b`)

type LogViewerModel struct {
	repo         *git.GitRepo
	mode         Mode
	logLines     []string
	commitHashes []string // parallel to logLines; empty string for graph-only lines
	currentIndex int
	scrollOffset int
	visibleLines int
	width        int
	height       int

	diffViewer DiffViewerModel

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	helpStyle       lipgloss.Style
}

func NewLogViewerModel(repo *git.GitRepo, content string) LogViewerModel {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	hashes := make([]string, len(lines))
	for i, line := range lines {
		if m := hashRegex.FindString(line); m != "" {
			hashes[i] = m
		}
	}

	return LogViewerModel{
		repo:         repo,
		mode:         NormalMode,
		logLines:     lines,
		commitHashes: hashes,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		helpStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

func (m LogViewerModel) Init() tea.Cmd {
	return nil
}

func (m LogViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == DetailMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc":
				m.mode = NormalMode
				return m, nil
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			updatedViewer, sizeCmd := m.diffViewer.Update(msg)
			if dv, ok := updatedViewer.(DiffViewerModel); ok {
				m.diffViewer = dv
			}
			return m, sizeCmd
		case diffLoadedMsg:
			updatedViewer, loadCmd := m.diffViewer.Update(msg)
			if dv, ok := updatedViewer.(DiffViewerModel); ok {
				m.diffViewer = dv
			}
			return m, loadCmd
		}
		// Forward all other keys to the diff viewer
		updatedViewer, viewCmd := m.diffViewer.Update(msg)
		if dv, ok := updatedViewer.(DiffViewerModel); ok {
			m.diffViewer = dv
		}
		return m, viewCmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j", "down":
			if len(m.logLines) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.logLines)
				m.adjustScrolling()
			}

		case "k", "up":
			if len(m.logLines) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.logLines)) % len(m.logLines)
				m.adjustScrolling()
			}

		case "g", "home":
			m.currentIndex = 0
			m.scrollOffset = 0

		case "G", "end":
			if len(m.logLines) > 0 {
				m.currentIndex = len(m.logLines) - 1
				m.adjustScrolling()
			}

		case "enter":
			if m.currentIndex < len(m.commitHashes) && m.commitHashes[m.currentIndex] != "" {
				hash := m.commitHashes[m.currentIndex]
				m.diffViewer = NewDiffViewerModel(m.repo, hash)
				m.mode = DetailMode
				var cmds []tea.Cmd
				cmds = append(cmds, m.loadCommitDetail(hash))
				if m.width > 0 && m.height > 0 {
					sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
					updatedViewer, sizeCmd := m.diffViewer.Update(sizeMsg)
					if dv, ok := updatedViewer.(DiffViewerModel); ok {
						m.diffViewer = dv
					}
					if sizeCmd != nil {
						cmds = append(cmds, sizeCmd)
					}
				}
				return m, tea.Batch(cmds...)
			}
		}
	}

	return m, cmd
}

func (m LogViewerModel) loadCommitDetail(hash string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.repo.ShowCommit(hash)
		return diffLoadedMsg{content: content, err: err}
	}
}

func (m LogViewerModel) View() string {
	if m.mode == DetailMode {
		return m.diffViewer.View()
	}

	var sections []string
	sections = append(sections, m.titleStyle.Render("Git Log"))
	sections = append(sections, "")

	startIdx := m.scrollOffset
	endIdx := min(startIdx+m.visibleLines, len(m.logLines))

	for i := startIdx; i < endIdx; i++ {
		line := m.logLines[i]
		if i == m.currentIndex {
			sections = append(sections, m.selectedStyle.Render("> "+line))
		} else {
			sections = append(sections, m.unselectedStyle.Render("  "+line))
		}
	}

	sections = append(sections, "")
	help := "j/k: navigate | enter: view commit | g/G: top/bottom | q: quit"
	sections = append(sections, m.helpStyle.Render(help))

	return strings.Join(sections, "\n")
}

func (m *LogViewerModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}
	maxOffset := len(m.logLines) - m.visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func StartLogViewer(repo *git.GitRepo, content string) error {
	m := NewLogViewerModel(repo, content)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
