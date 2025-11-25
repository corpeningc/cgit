package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type FilePickerModel struct {
	repo  *git.GitRepo
	files []string

	fileStatuses         []git.FileStatus
	stagedFileStatuses   []git.FileStatus
	unstagedFileStatuses []git.FileStatus

	selectedFiles      map[string]bool
	stagedSelections   map[string]bool
	unstagedSelections map[string]bool

	operationInProgress bool
	lastOperationStatus string
	showStatusMessage   bool

	currentIndex    int
	mode            Mode
	searchInput     textinput.Model
	searchQuery     string
	filteredIndices []int
	searchSelected  int
	quitting        bool
	confirmed       bool
	width           int
	height          int
	showStatusChars bool
	removing        bool

	// Staged files?
	staged bool
	// Scrolling support
	scrollOffset int
	visibleLines int

	// Diff viewer
	diffViewer DiffViewerModel

	// Styles
	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	checkedStyle    lipgloss.Style
	helpStyle       lipgloss.Style
	searchStyle     lipgloss.Style
}

func NewFilePicker(repo *git.GitRepo, stagedFileStatuses []git.FileStatus, unstagedFileStatuses []git.FileStatus, startInStaged bool) FilePickerModel {
	si := textinput.New()
	si.Placeholder = "Search files..."
	si.CharLimit = 100
	si.Width = 50

	var activeFileStatuses []git.FileStatus
	var files []string

	if startInStaged {
		activeFileStatuses = stagedFileStatuses
	} else {
		activeFileStatuses = unstagedFileStatuses
	}

	for _, status := range activeFileStatuses {
		files = append(files, status.Path)
	}

	return FilePickerModel{
		repo:                 repo,
		files:                files,
		fileStatuses:         activeFileStatuses,
		stagedFileStatuses:   stagedFileStatuses,
		unstagedFileStatuses: unstagedFileStatuses,
		selectedFiles:        make(map[string]bool),
		stagedSelections:     make(map[string]bool),
		unstagedSelections:   make(map[string]bool),
		searchInput:          si,
		showStatusChars:      true,
		staged:               startInStaged,

		// Initialize styles
		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),

		unselectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		checkedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		searchStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
	}
}

func (m FilePickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle diff mode separately
	if m.mode == DiffMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "q":
				m.mode = NormalMode
				return m, nil
			}
		}

		updatedModel, cmd := m.diffViewer.Update(msg)
		if diffModel, ok := updatedModel.(DiffViewerModel); ok {
			m.diffViewer = diffModel
		}
		return m, cmd
	}

	if m.mode == SearchMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.mode = NormalMode
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 6 // Account for title, help, etc.

	case GitOperationCompleteMsg:
		m.operationInProgress = false
		if msg.success {
			action := "staged"
			if msg.operation == "restore" {
				if m.staged {
					action = "restored from staging"
				} else {
					action = "discarded"
				}
			}

			m.lastOperationStatus = fmt.Sprintf("✓ %s %d file(s)", action, len(msg.filesAffected))
			m.showStatusMessage = true
			return m, tea.Batch(
				m.refreshRepositoryStatus(),
				m.clearStatusAfterDelay(),
			)
		} else {
			m.lastOperationStatus = fmt.Sprintf("✗ Error: %v", msg.error)
			m.showStatusMessage = true
			return m, m.clearStatusAfterDelay()
		}

	case StatusRefreshMsg:
		if msg.error != nil {
			m.lastOperationStatus = fmt.Sprintf("✗ Failed to refresh: %v", msg.error)
			m.showStatusMessage = true
			return m, m.clearStatusAfterDelay()
		}

		m.stagedFileStatuses = msg.stagedFiles
		m.unstagedFileStatuses = msg.unstagedFiles

		if m.staged {
			m.fileStatuses = m.stagedFileStatuses
			m.selectedFiles = m.stagedSelections
		} else {
			m.fileStatuses = m.unstagedFileStatuses
			m.selectedFiles = m.unstagedSelections
		}

		m.files = []string{}
		for _, status := range m.fileStatuses {
			m.files = append(m.files, status.Path)
		}

		if m.currentIndex >= len(m.files) {
			if len(m.files) > 0 {
				m.currentIndex = len(m.files) - 1
			} else {
				m.currentIndex = 0
			}
		}
		m.adjustScrolling()

		return m, nil

	case ClearStatusMsg:
		m.showStatusMessage = false
		return m, nil

	case tea.KeyMsg:
		if m.mode == SearchMode {
			switch msg.String() {
			case "esc":
				m.mode = NormalMode
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.filteredIndices = nil
				m.searchSelected = 0
				return m, nil
			}
		} else if m.mode == DiffMode {

		} else { // This is normal mode

			if m.quitting {
				return m, tea.Quit
			}

			switch msg.String() {
			case "q", "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit

			case "enter":
				if len(m.files) > 0 {
					file := m.files[m.currentIndex]
					m.selectedFiles[file] = !m.selectedFiles[file]
				}

			case "c", "ctrl+enter":
				if m.operationInProgress || len(m.getSelectedFiles()) == 0 {
					return m, nil
				}

				if m.staged {
					m.lastOperationStatus = "Cannot stage already staged files. Use 'r' to restore."
					m.showStatusMessage = true
					return m, tea.Batch(m.clearStatusAfterDelay())
				}

				selectedFiles := m.getSelectedFiles()
				m.operationInProgress = true
				m.selectedFiles = make(map[string]bool)

				return m, m.performGitOperation(selectedFiles, false)

			case "r":
				if m.operationInProgress || len(m.getSelectedFiles()) == 0 {
					return m, nil
				}

				selectedFiles := m.getSelectedFiles()
				m.operationInProgress = true
				m.selectedFiles = make(map[string]bool)

				return m, m.performGitOperation(selectedFiles, true)
			case "/":
				m.mode = SearchMode
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				return m, nil

			case "j", "down":
				if len(m.files) > 0 {
					m.currentIndex = (m.currentIndex + 1) % len(m.files)
					m.adjustScrolling()
				}

			case "k", "up":
				// Navigate up in file list with scrolling
				if len(m.files) > 0 {
					m.currentIndex = (m.currentIndex - 1 + len(m.files)) % len(m.files)
					m.adjustScrolling()
				}

			case "g":
				m.currentIndex = 0
				m.scrollOffset = 0

			case "G":
				if len(m.files) > 0 {
					m.currentIndex = len(m.files) - 1
					m.adjustScrolling()
				}

			case " ":
				if len(m.files) > 0 {
					filePath := m.files[m.currentIndex]
					m.diffViewer = NewDiffViewerModel(m.repo, filePath)
					m.diffViewer.staged = m.staged
					m.mode = DiffMode
					var cmds []tea.Cmd
					cmds = append(cmds, m.diffViewer.Init())
					if m.width > 0 && m.height > 0 {
						sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
						updatedModel, sizeCmd := m.diffViewer.Update(sizeMsg)
						if diffModel, ok := updatedModel.(DiffViewerModel); ok {
							m.diffViewer = diffModel
						}
						if sizeCmd != nil {
							cmds = append(cmds, sizeCmd)
						}
					}
					return m, tea.Batch(cmds...)
				}

			case "a":
				// Select all files
				for _, file := range m.files {
					m.selectedFiles[file] = true
				}

			case "tab":
				if !m.operationInProgress {
					if m.staged {
						m.stagedSelections = m.selectedFiles
					} else {
						m.unstagedSelections = m.selectedFiles
					}

					m.showStatusMessage = false

					m.staged = !m.staged
					if m.staged {
						m.fileStatuses = m.stagedFileStatuses
						m.selectedFiles = m.stagedSelections
					} else {
						m.fileStatuses = m.unstagedFileStatuses
						m.selectedFiles = m.unstagedSelections
					}

					m.files = []string{}
					for _, status := range m.fileStatuses {
						m.files = append(m.files, status.Path)
					}

					m.currentIndex = 0
					m.scrollOffset = 0
				}

			case "A":
				// Deselect all files
				m.selectedFiles = make(map[string]bool)
			}
		}
	}

	return m, cmd
}

func (m FilePickerModel) View() string {
	if m.quitting {
		return ""
	}

	// Handle diff mode
	if m.mode == DiffMode {
		return m.diffViewer.View()
	}

	var sections []string
	var managing string

	if m.staged {
		managing = "Staged changes"
	} else {
		managing = "Unstaged changes"
	}
	// Title
	title := m.titleStyle.Render("Select files to manage --- ", managing)
	sections = append(sections, title)

	if m.showStatusMessage && m.lastOperationStatus != "" {
		statusStyle := m.checkedStyle
		if strings.HasPrefix(m.lastOperationStatus, "✗") {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		}
		sections = append(sections, statusStyle.Render(m.lastOperationStatus))
	}

	if m.operationInProgress {
		sections = append(sections, m.searchStyle.Render("⏳ Operation in progress..."))
	}

	if m.mode == SearchMode {
		// Show search input
		searchTitle := m.searchStyle.Render("Search files:")
		sections = append(sections, searchTitle)
		sections = append(sections, m.searchInput.View())

		// Show search results
		if m.searchQuery != "" {
			if len(m.filteredIndices) == 0 {
				sections = append(sections, m.unselectedStyle.Render("No matches found"))
			} else {
				resultsTitle := m.searchStyle.Render(fmt.Sprintf("Results (%d matches):", len(m.filteredIndices)))
				sections = append(sections, resultsTitle)

				// Show filtered files with navigation
				for i, idx := range m.filteredIndices {
					if idx >= len(m.files) {
						continue
					}

					file := m.files[idx]
					prefix := "  "
					style := m.unselectedStyle

					if i == m.searchSelected {
						prefix = "> "
						style = m.selectedStyle
					}

					checkbox := "[ ]"
					if m.selectedFiles[file] {
						checkbox = m.checkedStyle.Render("[x]")
					}

					// Add status character if available
					statusChar := ""
					if m.showStatusChars && idx < len(m.fileStatuses) {
						statusChar = fmt.Sprintf("[%s] ", m.fileStatuses[idx].Status)
					}

					line := fmt.Sprintf("%s%s %s%s", prefix, checkbox, statusChar, file)
					sections = append(sections, style.Render(line))
				}
			}
		} else {
			sections = append(sections, m.unselectedStyle.Render("Type to search..."))
		}
	} else {
		// Show file list with scrolling
		selectedCount := len(m.getSelectedFiles())
		subtitle := fmt.Sprintf("(%d selected)", selectedCount)
		sections = append(sections, m.unselectedStyle.Render(subtitle))
		sections = append(sections, "")

		// Calculate visible range
		startIdx := m.scrollOffset
		endIdx := min(startIdx+m.visibleLines, len(m.files))

		// Show visible files
		for i := startIdx; i < endIdx; i++ {
			file := m.files[i]
			prefix := "  "
			style := m.unselectedStyle

			if i == m.currentIndex {
				prefix = "> "
				style = m.selectedStyle
			}

			checkbox := "[ ]"
			if m.selectedFiles[file] {
				checkbox = m.checkedStyle.Render("[x]")
			}

			// Add status character if available
			statusChar := ""
			if m.showStatusChars && i < len(m.fileStatuses) {
				statusChar = fmt.Sprintf("[%s] ", m.fileStatuses[i].Status)
			}

			line := fmt.Sprintf("%s%s %s%s", prefix, checkbox, statusChar, file)
			sections = append(sections, style.Render(line))
		}

		// Show scroll indicator if needed
		if len(m.files) > m.visibleLines {
			scrollInfo := fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.files))
			sections = append(sections, "")
			sections = append(sections, m.helpStyle.Render(scrollInfo))
		}
	}

	// Help
	help := ""
	if m.mode == SearchMode {
		help = "space: diff | enter: select | esc: back "
	} else if !m.staged {
		help = "Tab: toggle /: search | space: diff | enter: select | c: stage | r: remove | a: select all | A: deselect all | q: quit"
	} else {
		help = "Tab: toggle /: search | space: diff | enter: select | r: restore | a: select all | A: deselect all | q: quit"
	}

	sections = append(sections, "")
	sections = append(sections, m.helpStyle.Render(help))

	return strings.Join(sections, "\n")
}

func (m *FilePickerModel) adjustScrolling() {
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
	maxOffset := len(m.files) - m.visibleLines
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

func (m *FilePickerModel) performSearch() {
	if m.searchQuery == "" {
		m.filteredIndices = nil
		m.searchSelected = 0
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filteredIndices = []int{}

	for i, file := range m.files {
		if m.fuzzyMatch(strings.ToLower(file), query) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}

	// Reset search selection to first result
	m.searchSelected = 0
}

func (m FilePickerModel) fuzzyMatch(text, query string) bool {
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

func (m FilePickerModel) getSelectedFiles() []string {
	var selected []string
	for file, isSelected := range m.selectedFiles {
		if isSelected {
			selected = append(selected, file)
		}
	}
	return selected
}

func (m FilePickerModel) performGitOperation(files []string, restore bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		var operation string

		if restore {
			operation = "restore"
			err = m.repo.RemoveFiles(files, m.staged)
		} else {
			operation = "stage"
			err = m.repo.AddFiles(files)
		}

		return GitOperationCompleteMsg{
			success:       err == nil,
			error:         err,
			operation:     operation,
			filesAffected: files,
		}
	}
}

func (m FilePickerModel) refreshRepositoryStatus() tea.Cmd {
	return func() tea.Msg {
		stagedFiles, unstagedFiles, err := m.repo.GetFileStatuses()
		return StatusRefreshMsg{
			stagedFiles:   stagedFiles,
			unstagedFiles: unstagedFiles,
			error:         err,
		}
	}
}

func (m FilePickerModel) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// SelectFiles provides an enhanced file picker specifically for unstaged files with status display
func SelectFiles(repo *git.GitRepo, stagedFileStatuses []git.FileStatus, unstagedFileStatuses []git.FileStatus, staged bool) ([]string, bool, error) {
	if len(stagedFileStatuses) == 0 && len(unstagedFileStatuses) == 0 {
		return []string{}, false, nil
	}

	m := NewFilePicker(repo, stagedFileStatuses, unstagedFileStatuses, staged)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, false, err
	}

	// Type assert to get our model back
	if model, ok := finalModel.(FilePickerModel); ok {
		if model.confirmed {
			return model.getSelectedFiles(), model.removing, nil
		}
	}

	return []string{}, false, nil
}
