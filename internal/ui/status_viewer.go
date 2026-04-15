package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type statusFilesLoadedMsg struct {
	staged   []git.FileStatus
	unstaged []git.FileStatus
	err      error
}

type StatusViewerModel struct {
	repo          *git.GitRepo
	stagedFiles   []git.FileStatus
	unstagedFiles []git.FileStatus
	statusBar     StatusBar
	currentTab    int // 0=staged, 1=unstaged
	currentIndex  int
	scrollOffset  int
	visibleLines  int
	width         int
	height        int
	launchManage  bool
	manageStaged  bool

	titleStyle       lipgloss.Style
	selectedStyle    lipgloss.Style
	unselectedStyle  lipgloss.Style
	activeTabStyle   lipgloss.Style
	inactiveTabStyle lipgloss.Style
	helpStyle        lipgloss.Style
	stagedStyle      lipgloss.Style
	unstagedStyle    lipgloss.Style
}

func NewStatusViewerModel(repo *git.GitRepo) StatusViewerModel {
	return StatusViewerModel{
		repo: repo,

		titleStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		selectedStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		activeTabStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Underline(true),
		inactiveTabStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		helpStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		stagedStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
		unstagedStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	}
}

func (m StatusViewerModel) Init() tea.Cmd {
	return tea.Batch(FetchStatusBar(m.repo), m.fetchFiles())
}

func (m StatusViewerModel) fetchFiles() tea.Cmd {
	return func() tea.Msg {
		staged, unstaged, err := m.repo.GetFileStatuses()
		return statusFilesLoadedMsg{staged: staged, unstaged: unstaged, err: err}
	}
}

func (m StatusViewerModel) currentFiles() []git.FileStatus {
	if m.currentTab == 0 {
		return m.stagedFiles
	}
	return m.unstagedFiles
}

func (m StatusViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 8

	case StatusBarMsg:
		m.statusBar = msg.Bar

	case statusFilesLoadedMsg:
		if msg.err == nil {
			m.stagedFiles = msg.staged
			m.unstagedFiles = msg.unstaged
		}
		m.currentIndex = 0
		m.scrollOffset = 0

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "tab":
			m.currentTab = 1 - m.currentTab
			m.currentIndex = 0
			m.scrollOffset = 0

		case "j", "down":
			files := m.currentFiles()
			if len(files) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(files)
				m.adjustScrolling()
			}

		case "k", "up":
			files := m.currentFiles()
			if len(files) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(files)) % len(files)
				m.adjustScrolling()
			}

		case "m":
			m.launchManage = true
			m.manageStaged = m.currentTab == 0
			return m, tea.Quit

		case "r":
			return m, m.fetchFiles()
		}
	}

	return m, nil
}

func (m StatusViewerModel) View() string {
	var sections []string

	if bar := m.statusBar.Render(m.helpStyle); bar != "" {
		sections = append(sections, bar)
	}

	sections = append(sections, "")

	stagedLabel := fmt.Sprintf("  Staged (%d)  ", len(m.stagedFiles))
	unstagedLabel := fmt.Sprintf("  Unstaged (%d)  ", len(m.unstagedFiles))
	if m.currentTab == 0 {
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
			m.activeTabStyle.Render(stagedLabel),
			m.inactiveTabStyle.Render(unstagedLabel)))
	} else {
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top,
			m.inactiveTabStyle.Render(stagedLabel),
			m.activeTabStyle.Render(unstagedLabel)))
	}
	sections = append(sections, "")

	files := m.currentFiles()
	if len(files) == 0 {
		sections = append(sections, m.unselectedStyle.Render("  No files"))
	} else {
		startIdx := m.scrollOffset
		endIdx := min(startIdx+m.visibleLines, len(files))
		for i := startIdx; i < endIdx; i++ {
			f := files[i]
			prefix := "  "
			style := m.unselectedStyle
			if i == m.currentIndex {
				prefix = "> "
				style = m.selectedStyle
			}
			statusStyle := m.stagedStyle
			if m.currentTab == 1 {
				statusStyle = m.unstagedStyle
			}
			line := fmt.Sprintf("%s%s  %s", prefix, statusStyle.Render(f.Status), f.Path)
			sections = append(sections, style.Render(line))
		}
		if len(files) > m.visibleLines {
			sections = append(sections, "")
			sections = append(sections, m.helpStyle.Render(fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(files))))
		}
	}

	sections = append(sections, "")
	sections = append(sections, m.helpStyle.Render("Tab: switch  j/k: navigate  m: manage  r: refresh  q: quit"))

	return strings.Join(sections, "\n")
}

func (m *StatusViewerModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}
	files := m.currentFiles()
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}
	maxOffset := len(files) - m.visibleLines
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

// StartStatusViewer runs the status TUI, looping back after manage sessions.
func StartStatusViewer(repo *git.GitRepo) error {
	for {
		m := NewStatusViewerModel(repo)
		p := tea.NewProgram(m, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		sv, ok := finalModel.(StatusViewerModel)
		if !ok || !sv.launchManage {
			return nil
		}
		repoStatus, err := repo.GetRepositoryStatus()
		if err != nil {
			return err
		}
		_, _, err = SelectFiles(repo, repoStatus.StagedFiles, repoStatus.UnstagedFiles, sv.manageStaged)
		if err != nil {
			return err
		}
	}
}
