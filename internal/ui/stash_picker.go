package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type stashOpMsg struct {
	ref string
	op  string // "pop", "apply", "drop"
	err error
}

type stashRefreshMsg struct {
	stashes []git.StashEntry
	err     error
}

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

	diffViewer DiffViewerModel
	splitPane  bool

	lastStatus     string
	showLastStatus bool

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	successStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	helpStyle       lipgloss.Style
	separatorStyle  lipgloss.Style
}

func NewStashPickerModel(repo *git.GitRepo, stashes []git.StashEntry) StashPickerModel {
	si := textinput.New()
	si.Placeholder = "Search stashes..."
	si.CharLimit = 100
	si.Width = 50

	m := StashPickerModel{
		repo:      repo,
		mode:      NormalMode,
		stashes:   stashes,
		splitPane: true,

		searchInput: si,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		successStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		errorStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		helpStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		separatorStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}

	if len(stashes) > 0 {
		m.diffViewer = NewDiffViewerModel(repo, stashes[0].Ref)
	}

	return m
}

func (m StashPickerModel) Init() tea.Cmd {
	if len(m.stashes) > 0 {
		return tea.Batch(textinput.Blink, m.loadCurrentStashDiff())
	}
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
	return style.Render(fmt.Sprintf("%s[%s] %s", prefix, entry.Ref, entry.Description))
}

func (m StashPickerModel) View() string {
	leftWidth := m.width / 2
	if leftWidth < 10 {
		leftWidth = m.width
	}

	var sections []string

	if m.mode != SearchMode {
		sections = append(sections, m.titleStyle.Render("Stashes"))

		if m.showLastStatus {
			style := m.successStyle
			if strings.HasPrefix(m.lastStatus, "✗") {
				style = m.errorStyle
			}
			sections = append(sections, style.Render(m.lastStatus))
		}

		sections = append(sections, "")

		startIdx := m.scrollOffset
		endIdx := min(startIdx+m.visibleLines, len(m.stashes))
		for i := startIdx; i < endIdx; i++ {
			sections = append(sections, m.renderStash(i))
		}

		if len(m.stashes) > m.visibleLines {
			sections = append(sections, "")
			sections = append(sections, m.helpStyle.Render(fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.stashes))))
		}

		sections = append(sections, "")
		sections = append(sections, m.helpStyle.Render("enter: pop  a: apply  d: drop  s: toggle diff  /: search  q: quit"))
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
		sections = append(sections, m.helpStyle.Render("enter: lock results  esc: back"))
	}

	if m.splitPane && m.width > 20 {
		leftPanel := lipgloss.NewStyle().Width(leftWidth).Render(strings.Join(sections, "\n"))
		separator := m.separatorStyle.Render(strings.Repeat("│\n", m.height))
		return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, m.diffViewer.View())
	}

	return strings.Join(sections, "\n")
}

func (m StashPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 8
		leftWidth := msg.Width / 2
		rightWidth := msg.Width - leftWidth - 1
		rightMsg := tea.WindowSizeMsg{Width: rightWidth, Height: msg.Height}
		updatedDiff, diffCmd := m.diffViewer.Update(rightMsg)
		if dv, ok := updatedDiff.(DiffViewerModel); ok {
			m.diffViewer = dv
		}
		return m, diffCmd

	case diffLoadedMsg:
		updatedDiff, diffCmd := m.diffViewer.Update(msg)
		if dv, ok := updatedDiff.(DiffViewerModel); ok {
			m.diffViewer = dv
		}
		return m, diffCmd

	case stashOpMsg:
		if msg.err != nil {
			m.lastStatus = fmt.Sprintf("✗ %s %s: %v", msg.op, msg.ref, msg.err)
		} else {
			m.lastStatus = fmt.Sprintf("✓ %s %s", msg.op, msg.ref)
		}
		m.showLastStatus = true
		if msg.op == "drop" || msg.op == "pop" {
			return m, m.refreshStashes()
		}
		return m, nil

	case stashRefreshMsg:
		if msg.err != nil {
			m.lastStatus = fmt.Sprintf("✗ Refresh failed: %v", msg.err)
			m.showLastStatus = true
			return m, nil
		}
		m.stashes = msg.stashes
		if len(m.stashes) == 0 {
			return m, tea.Quit
		}
		if m.currentIndex >= len(m.stashes) {
			m.currentIndex = len(m.stashes) - 1
		}
		return m, m.loadCurrentStashDiff()
	}

	// Diff panel scroll keys (always active in normal mode)
	if m.mode == NormalMode {
		switch msg.(type) {
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "ctrl+j":
				m.diffViewer.viewport.ScrollDown(1)
				return m, nil
			case "ctrl+k":
				m.diffViewer.viewport.ScrollUp(1)
				return m, nil
			case "ctrl+d":
				m.diffViewer.viewport.HalfPageDown()
				return m, nil
			case "ctrl+u":
				m.diffViewer.viewport.HalfPageUp()
				return m, nil
			}
		}
	}

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
				return m, m.loadCurrentStashDiff()
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
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j":
			if len(m.stashes) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.stashes)
				m.adjustScrolling()
				return m, m.loadCurrentStashDiff()
			}

		case "k":
			if len(m.stashes) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.stashes)) % len(m.stashes)
				m.adjustScrolling()
				return m, m.loadCurrentStashDiff()
			}

		case "s":
			m.splitPane = !m.splitPane

		case "enter":
			if len(m.stashes) == 0 {
				return m, tea.Quit
			}
			ref := m.stashes[m.currentIndex].Ref
			return m, m.stashOp(ref, "pop")

		case "a":
			if len(m.stashes) > 0 {
				ref := m.stashes[m.currentIndex].Ref
				return m, m.stashOp(ref, "apply")
			}

		case "d":
			if len(m.stashes) > 0 {
				ref := m.stashes[m.currentIndex].Ref
				return m, m.stashOp(ref, "drop")
			}

		case "/":
			m.mode = SearchMode
			m.searchInput.Focus()
			m.searchInput.SetValue(m.searchQuery)
			return m, nil
		}
	}

	return m, cmd
}

func (m *StashPickerModel) loadCurrentStashDiff() tea.Cmd {
	if len(m.stashes) == 0 {
		return nil
	}
	ref := m.stashes[m.currentIndex].Ref
	m.diffViewer = NewDiffViewerModel(m.repo, ref)
	if m.width > 0 {
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth - 1
		sizeMsg := tea.WindowSizeMsg{Width: rightWidth, Height: m.height}
		updated, _ := m.diffViewer.Update(sizeMsg)
		if dv, ok := updated.(DiffViewerModel); ok {
			m.diffViewer = dv
		}
	}
	repo := m.repo
	return func() tea.Msg {
		content, err := repo.StashDiff(ref)
		return diffLoadedMsg{content: content, err: err}
	}
}

func (m StashPickerModel) stashOp(ref, op string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch op {
		case "pop":
			err = m.repo.StashPopRef(ref)
		case "apply":
			err = m.repo.StashApply(ref)
		case "drop":
			err = m.repo.StashDrop(ref)
		}
		return stashOpMsg{ref: ref, op: op, err: err}
	}
}

func (m StashPickerModel) refreshStashes() tea.Cmd {
	return func() tea.Msg {
		stashes, err := m.repo.StashList()
		return stashRefreshMsg{stashes: stashes, err: err}
	}
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
	if len(stashes) == 1 {
		if err := repo.StashPopRef(stashes[0].Ref); err != nil {
			return err
		}
		fmt.Printf("Popped stash '%s'.\n", stashes[0].Ref)
		return nil
	}
	m := NewStashPickerModel(repo, stashes)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
