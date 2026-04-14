package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type StashPickerModel struct {
	repo    *git.GitRepo
	mode    Mode
	stashes []git.StashEntry

	currentIndex    int
	scrollOffset    int
	visibleLines    int
	width           int
	height          int
	searchInput     textinput.Model
	searchQuery     string
	filteredIndices []int
	searchSelected  int

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
}

func NewStashPickerModel(repo *git.GitRepo, stashes []git.StashEntry) StashPickerModel {
	si := textinput.New()
	si.Placeholder = "Search stashes..."
	si.CharLimit = 100
	si.Width = 50

	return StashPickerModel{
		repo:    repo,
		mode:    NormalMode,
		stashes: stashes,

		searchInput: si,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true),
	}
}

func (m StashPickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m StashPickerModel) renderStash(i int) string {
	entry := m.stashes[i]
	prefix := "  "
	style := m.unselectedStyle

	if i == m.currentIndex {
		prefix = "> "
		style = m.selectedStyle
	}

	line := fmt.Sprintf("%s [%s] %s", prefix, entry.Ref, entry.Description)
	return style.Render(line)
}

func (m StashPickerModel) View() string {
	var sections []string

	if m.mode != SearchMode {
		sections = append(sections, m.titleStyle.Render("Select a stash to pop"))

		startIdx := m.scrollOffset
		endIdx := min(startIdx+m.visibleLines, len(m.stashes))
		for i := startIdx; i < endIdx; i++ {
			sections = append(sections, m.renderStash(i))
		}

		if len(m.stashes) > m.visibleLines {
			info := fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.stashes))
			sections = append(sections, "")
			sections = append(sections, m.unselectedStyle.Render(info))
		}

		sections = append(sections, "")
		sections = append(sections, m.unselectedStyle.Render("j/k: navigate | enter: pop | /: search | q: quit"))
	} else {
		sections = append(sections, m.titleStyle.Render("Search stashes:"))
		sections = append(sections, m.searchInput.View())

		if m.searchQuery != "" {
			if len(m.filteredIndices) == 0 {
				sections = append(sections, m.unselectedStyle.Render("No matches found"))
			} else {
				sections = append(sections, m.titleStyle.Render(fmt.Sprintf("Results (%d matches)", len(m.filteredIndices))))
				for _, idx := range m.filteredIndices {
					if idx >= len(m.stashes) {
						continue
					}
					sections = append(sections, m.renderStash(idx))
				}
			}
		} else {
			sections = append(sections, m.unselectedStyle.Render("Type to search..."))
		}

		sections = append(sections, "")
		sections = append(sections, m.unselectedStyle.Render("enter: lock results | esc: back"))
	}

	return strings.Join(sections, "\n")
}

func (m StashPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == SearchMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.mode = NormalMode
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.filteredIndices = nil
				return m, nil
			case "enter":
				if len(m.filteredIndices) > 0 {
					filtered := make([]git.StashEntry, len(m.filteredIndices))
					for i, idx := range m.filteredIndices {
						filtered[i] = m.stashes[idx]
					}
					m.stashes = filtered
				}
				m.mode = NormalMode
				m.currentIndex = 0
				return m, nil
			}
		}

		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		if m.searchInput.Value() != oldValue {
			m.searchQuery = m.searchInput.Value()
			m.performSearch()
		}
		return m, cmd
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
			if len(m.stashes) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.stashes)
				m.adjustScrolling()
			}

		case "k":
			if len(m.stashes) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.stashes)) % len(m.stashes)
				m.adjustScrolling()
			}

		case "enter":
			if len(m.stashes) == 0 {
				return m, tea.Quit
			}
			ref := m.stashes[m.currentIndex].Ref
			err := m.repo.StashPopRef(ref)
			if err != nil {
				fmt.Printf("Error popping stash: %v\n", err)
			} else {
				fmt.Printf("Successfully popped stash '%s'.\n", ref)
			}
			return m, tea.Quit

		case "/":
			m.mode = SearchMode
			m.searchInput.Focus()
			m.searchInput.SetValue(m.searchQuery)
			return m, nil
		}
	}

	return m, cmd
}

func (m *StashPickerModel) performSearch() {
	if m.searchQuery == "" {
		m.filteredIndices = nil
		m.searchSelected = 0
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filteredIndices = []int{}
	for i, entry := range m.stashes {
		text := strings.ToLower(entry.Ref + " " + entry.Description)
		if fuzzyMatchStr(text, query) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}
	m.searchSelected = 0
}

func (m *StashPickerModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}
	maxOffset := len(m.stashes) - m.visibleLines
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

// fuzzyMatchStr is the shared fuzzy match logic used across pickers.
func fuzzyMatchStr(text, query string) bool {
	if query == "" {
		return true
	}
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

func StartStashPicker(repo *git.GitRepo) error {
	stashes, err := repo.StashList()
	if err != nil {
		return err
	}

	if len(stashes) == 0 {
		fmt.Println("No stashes found.")
		return nil
	}

	// Skip picker when there's only one stash
	if len(stashes) == 1 {
		err = repo.StashPopRef(stashes[0].Ref)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully popped stash '%s'.\n", stashes[0].Ref)
		return nil
	}

	m := NewStashPickerModel(repo, stashes)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
