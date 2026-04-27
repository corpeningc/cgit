package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type branchOpMsg struct {
	op  string
	err error
}

type branchRefreshMsg struct {
	branches []git.BranchDetail
	err      error
}

type BranchManagerModel struct {
	repo         *git.GitRepo
	branches     []git.BranchDetail
	currentIndex int
	scrollOffset int
	visibleLines int
	width        int
	height       int

	lastStatus     string
	showLastStatus bool
	switched       bool // signals the caller to re-exec to pick up new branch

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	currentStyle    lipgloss.Style
	successStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	helpStyle       lipgloss.Style
	dimStyle        lipgloss.Style
}

func NewBranchManagerModel(repo *git.GitRepo, branches []git.BranchDetail) BranchManagerModel {
	return BranchManagerModel{
		repo:     repo,
		branches: branches,

		titleStyle:      TitlePinkStyle,
		selectedStyle:   SelectedPeachStyle,
		unselectedStyle: UnselectedStyle,
		currentStyle:    SuccessStyle,
		successStyle:    SuccessStyle,
		errorStyle:      ErrorStyle,
		helpStyle:       HelpStyle,
		dimStyle:        DimStyle,
	}
}

func (m BranchManagerModel) Init() tea.Cmd {
	return nil
}

func (m BranchManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 7

	case branchOpMsg:
		if msg.err != nil {
			m.lastStatus = fmt.Sprintf("✗ %s: %v", msg.op, msg.err)
		} else {
			m.lastStatus = fmt.Sprintf("✓ %s", msg.op)
		}
		m.showLastStatus = true
		return m, m.refresh()

	case branchRefreshMsg:
		if msg.err != nil {
			m.lastStatus = fmt.Sprintf("✗ Refresh failed: %v", msg.err)
			m.showLastStatus = true
			return m, nil
		}
		m.branches = msg.branches
		if m.currentIndex >= len(m.branches) {
			m.currentIndex = max(0, len(m.branches)-1)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j", "down":
			if len(m.branches) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.branches)
				m.adjustScrolling()
			}

		case "k", "up":
			if len(m.branches) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.branches)) % len(m.branches)
				m.adjustScrolling()
			}

		case "enter":
			if len(m.branches) > 0 {
				b := m.branches[m.currentIndex]
				if b.Current {
					return m, nil
				}
				return m, m.switchBranch(b.Name)
			}

		case "d":
			if len(m.branches) > 0 {
				b := m.branches[m.currentIndex]
				if b.Current {
					m.lastStatus = "✗ Cannot delete the current branch"
					m.showLastStatus = true
					return m, nil
				}
				return m, m.deleteBranch(b.Name, false)
			}

		case "D":
			if len(m.branches) > 0 {
				b := m.branches[m.currentIndex]
				if b.Current {
					m.lastStatus = "✗ Cannot delete the current branch"
					m.showLastStatus = true
					return m, nil
				}
				return m, m.deleteBranch(b.Name, true)
			}
		}
	}

	return m, nil
}

func (m BranchManagerModel) View() string {
	var sections []string
	sections = append(sections, m.titleStyle.Render(fmt.Sprintf("Branches (%d)", len(m.branches))))

	if m.showLastStatus {
		style := m.successStyle
		if strings.HasPrefix(m.lastStatus, "✗") {
			style = m.errorStyle
		}
		sections = append(sections, style.Render(m.lastStatus))
	}

	sections = append(sections, "")

	startIdx := m.scrollOffset
	endIdx := min(startIdx+m.visibleLines, len(m.branches))

	for i := startIdx; i < endIdx; i++ {
		b := m.branches[i]
		prefix := "  "
		nameStyle := m.unselectedStyle
		if i == m.currentIndex {
			prefix = "> "
			nameStyle = m.selectedStyle
		}
		marker := " "
		if b.Current {
			marker = m.currentStyle.Render("*")
		}
		meta := m.dimStyle.Render(fmt.Sprintf("  %s  %s  %s", b.Hash, b.Date, b.Subject))
		line := fmt.Sprintf("%s%s %s%s", prefix, marker, nameStyle.Render(b.Name), meta)
		sections = append(sections, line)
	}

	if len(m.branches) > m.visibleLines {
		sections = append(sections, "")
		sections = append(sections, m.helpStyle.Render(fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.branches))))
	}

	sections = append(sections, "")
	sections = append(sections, m.helpStyle.Render("enter: switch  d: delete  D: force delete  j/k: navigate  q: quit"))

	return strings.Join(sections, "\n")
}

func (m BranchManagerModel) switchBranch(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.SwitchBranch(name)
		return branchOpMsg{op: "Switched to " + name, err: err}
	}
}

func (m BranchManagerModel) deleteBranch(name string, force bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if force {
			err = m.repo.ForceDeleteBranch(name)
		} else {
			err = m.repo.DeleteBranch(name)
		}
		return branchOpMsg{op: "Deleted " + name, err: err}
	}
}

func (m BranchManagerModel) refresh() tea.Cmd {
	return func() tea.Msg {
		branches, err := m.repo.GetBranchDetails()
		return branchRefreshMsg{branches: branches, err: err}
	}
}

func (m *BranchManagerModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}
	maxOffset := len(m.branches) - m.visibleLines
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

func StartBranchManager(repo *git.GitRepo) error {
	branches, err := repo.GetBranchDetails()
	if err != nil {
		return err
	}
	if len(branches) == 0 {
		fmt.Println("No branches found.")
		return nil
	}
	m := NewBranchManagerModel(repo, branches)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
