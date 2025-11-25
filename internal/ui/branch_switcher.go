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
	repo   *git.GitRepo
	remote bool
	mode   Mode

	// Scrolling support
	scrollOffset int
	visibleLines int
	currentIndex int

	width  int
	height int

	branches        []string
	searchInput     textinput.Model
	searchQuery     string
	filteredIndices []int
	searchSelected  int

	// Styles
	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
}

func (m BranchSwitcherModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m BranchSwitcherModel) renderBranches(i int) string {
	branch := m.branches[i]
	prefix := "  "
	style := m.unselectedStyle

	if i == m.currentIndex {
		prefix = "> "
		style = m.selectedStyle
	}

	line := fmt.Sprintf("%s %s", prefix, branch)
	return style.Render(line)
}

func (m BranchSwitcherModel) View() string {
	var sections []string

	if m.mode != SearchMode {
		title := m.titleStyle.Render("Select a branch")

		sections = append(sections, title)
		startIdx := m.scrollOffset
		endIdx := min(startIdx+m.visibleLines, len(m.branches))

		// Render branches
		for i := startIdx; i < endIdx; i++ {
			sections = append(sections, m.renderBranches(i))
		}

	} else {
		searchTitle := m.titleStyle.Render("Search branches:")
		sections = append(sections, searchTitle)
		sections = append(sections, m.searchInput.View())

		if m.searchQuery != "" {
			if len(m.filteredIndices) == 0 {
				sections = append(sections, m.unselectedStyle.Render("No matches found"))
			} else {
				resultsTitle := m.titleStyle.Render(fmt.Sprintf("Results (%d matches)", len(m.filteredIndices)))
				sections = append(sections, resultsTitle)

				for _, idx := range m.filteredIndices {
					if idx >= len(m.branches) {
						continue
					}

					// Render branches
					sections = append(sections, m.renderBranches(idx))
				}
			}
		} else {
			sections = append(sections, m.unselectedStyle.Render("Type to search..."))
		}
	}

	return strings.Join(sections, "\n")
}

func NewBranchBranchSwitcherModel(repo *git.GitRepo, remote bool) BranchSwitcherModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search branches..."
	searchInput.CharLimit = 100
	searchInput.Width = 50

	branches, err := repo.GetAllBranches(remote)

	if err != nil {
		fmt.Printf("Error initializing branch viewer %s", err)
	}

	return BranchSwitcherModel{
		repo:   repo,
		mode:   NormalMode,
		remote: remote,

		branches:    branches,
		searchInput: searchInput,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true),
	}
}

func (m BranchSwitcherModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == SearchMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.mode = NormalMode
				return m, nil
			case "enter":
				m.mode = SearchResultsMode
				return m, nil
			}
		}

		// Update search input if in search mode
		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Perform real-time search if input changed
		if m.searchInput.Value() != oldValue {
			m.searchQuery = m.searchInput.Value()
			m.performSearch()
		}
		return m, cmd
	}

	// TODO: handle SearchResult mode
	if m.mode == SearchResultsMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
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
				isClean, err := m.repo.IsClean()
				if err != nil {
					return m, nil
				}

				branch := m.branches[m.currentIndex]

				if !isClean {
					err = m.repo.Stash("Dirty working directory while switching to " + branch)

					if err != nil {
						return m, nil
					}
				}

				err = m.repo.SwitchBranch(branch)
				if err != nil {
					return m, nil
				}

				fmt.Printf("Successfully switched to branch '%s'.\n", branch)

				return m, tea.Quit
			}
		}
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 6

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
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
			isClean, err := m.repo.IsClean()
			if err != nil {
				return m, nil
			}

			branch := m.branches[m.currentIndex]

			if !isClean {
				err = m.repo.Stash("Dirty working directory while switching to " + branch)

				if err != nil {
					return m, nil
				}
			}

			err = m.repo.SwitchBranch(branch)
			if err != nil {
				return m, nil
			}

			fmt.Printf("Successfully switched to branch '%s'.\n", branch)

			return m, tea.Quit

		case "/":
			m.mode = SearchMode
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			return m, nil
		}
	}

	return m, cmd
}

func (m *BranchSwitcherModel) performSearch() {
	if m.searchQuery == "" {
		m.filteredIndices = nil
		m.searchSelected = 0
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filteredIndices = []int{}

	for i, branch := range m.branches {
		if m.fuzzyMatch(strings.ToLower(branch), query) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}

	// Reset search selection to first result
	m.searchSelected = 0
}

func SwitchBranches(repo *git.GitRepo, remote bool) ([]string, error) {
	m := NewBranchBranchSwitcherModel(repo, remote)

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

func (m BranchSwitcherModel) fuzzyMatch(text, query string) bool {
	if query == "" {
		return true
	}

	// Simple fuzzy matching - check if all characters in query appear in order
	textIdx := 0
	for _, queryChar := range query {
		found := false
		for textIdx < len(text) {
			if rune(text[textIdx]) == queryChar {
				found = true
				textIdx++
				break
			}
			textIdx++
		}
		if !found {
			return false
		}
	}
	return true
}
