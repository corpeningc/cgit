package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type BranchSwitcherModel struct {
	repo *git.GitRepo
	mode Mode

	// Scrolling support
	scrollOffset int
	visibleLines int
	currentIndex int

	width  int
	height int

	branches    []string
	searchInput textinput.Model
	searchQuery string

	// Styles
	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
}

func (m BranchSwitcherModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m BranchSwitcherModel) View() string {
	var sections []string
	title := m.titleStyle.Render("Select a branch")

	sections = append(sections, title)

	startIdx := m.scrollOffset
	endIndx := startIdx + m.visibleLines

	for i := startIdx; i < endIndx; i++ {
		branch := m.branches[i]
		prefix := "  "
		style := m.unselectedStyle

		if i == m.currentIndex {
			prefix = "> "
			style = m.selectedStyle
		}

		line := fmt.Sprintf("%s %s", prefix, branch)
		sections = append(sections, style.Render(line))
	}

	return strings.Join(sections, "\n")
}

func NewBranchBranchSwitcherModel(repo *git.GitRepo) BranchSwitcherModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search branches..."
	searchInput.CharLimit = 100
	searchInput.Width = 50

	branches, err := repo.GetAllBranches(true)

	if err != nil {
		fmt.Printf("Error initializing branch viewer %s", err)
	}

	return BranchSwitcherModel{
		repo: repo,
		mode: NormalMode,

		branches:    branches,
		searchInput: searchInput,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

func (m BranchSwitcherModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 6

	case tea.KeyMsg:
		if m.mode == SearchMode {
		} else {
			switch msg.String() {
			case "q":
				return m, tea.Quit

			case "j":
				if len(m.branches) > 0 {
					m.currentIndex = (m.currentIndex + 1) % len(m.branches)
					m.adjustScrolling()
				}

			case "k":
				if len(m.branches) > 0 {
					m.currentIndex = (m.currentIndex - 1 + len(m.branches)) % len(m.branches)
					m.adjustScrolling()
				}

			case "enter":
				branch := m.branches[m.currentIndex]
				err := m.repo.SwitchBranch(branch)
				if err != nil {
					return m, nil
				}
				return m, tea.Quit

			case "/":
				if m.mode == NormalMode {
					m.mode = SearchMode
					m.searchInput.Focus()
					m.searchInput.SetValue("")
					return m, nil
				}
			}
		}
	}
	return m, cmd
}

func SwitchBranches(repo *git.GitRepo) ([]string, error) {
	m := NewBranchBranchSwitcherModel(repo)

	program := tea.NewProgram(m, tea.WithAltScreen())

	_, err := program.Run()

	if err != nil {
		return nil, err
	}

	return []string{}, nil
}

func (m *BranchSwitcherModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}

	// If current item is below visible area, scroll down
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}

	// If current item is above visible area, scroll up
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}

	// Ensure we don't scroll past the end
	maxOffset := len(m.branches) - m.visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}

	// Ensure we don't scroll before the beginning
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}
