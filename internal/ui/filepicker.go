package ui

import (
	"fmt"
	"os/exec"
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
	searchLocked    bool
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

	// Diff viewer (visible on the right in split-pane mode)
	diffViewer DiffViewerModel
	splitPane  bool

	// Styles
	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	checkedStyle    lipgloss.Style
	helpStyle       lipgloss.Style
	searchStyle     lipgloss.Style
	separatorStyle  lipgloss.Style
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

	m := FilePickerModel{
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

		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}

	m.splitPane = true

	if len(files) > 0 {
		m.diffViewer = NewDiffViewerModel(repo, files[0])
		m.diffViewer.staged = startInStaged
	}

	return m
}

func (m FilePickerModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textinput.Blink)
	if len(m.files) > 0 {
		cmds = append(cmds, m.diffViewer.Init())
	}
	return tea.Batch(cmds...)
}

func (m FilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 8

		var diffMsg tea.WindowSizeMsg
		if m.mode == DiffMode {
			diffMsg = tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height}
		} else {
			leftWidth := msg.Width / 2
			diffMsg = tea.WindowSizeMsg{Width: msg.Width - leftWidth - 1, Height: msg.Height}
		}
		updatedDiff, diffCmd := m.diffViewer.Update(diffMsg)
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
			return m, tea.Batch(m.refreshRepositoryStatus(), m.clearStatusAfterDelay())
		}
		m.lastOperationStatus = fmt.Sprintf("✗ Error: %v", msg.error)
		m.showStatusMessage = true
		return m, m.clearStatusAfterDelay()

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
		return m, m.loadCurrentDiff()

	case ClearStatusMsg:
		m.showStatusMessage = false
		return m, nil

	case tea.KeyMsg:
		// Split-pane diff scroll keys (active in Normal and locked Search mode)
		if m.mode != DiffMode && m.mode != SearchMode || (m.mode == SearchMode && m.searchLocked) {
			switch msg.String() {
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

		switch msg.String() {
		case "esc":
			switch m.mode {
			case DiffMode:
				m.mode = NormalMode
				// Resize diff viewer back to right-pane width
				if m.width > 0 {
					leftWidth := m.width / 2
					rightWidth := m.width - leftWidth - 1
					sizeMsg := tea.WindowSizeMsg{Width: rightWidth, Height: m.height}
					updatedDiff, _ := m.diffViewer.Update(sizeMsg)
					if dv, ok := updatedDiff.(DiffViewerModel); ok {
						m.diffViewer = dv
					}
				}
			case SearchMode:
				m.mode = NormalMode
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.filteredIndices = nil
				m.searchSelected = 0
				m.searchLocked = false
			case NormalMode:
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case "q":
			switch m.mode {
			case DiffMode:
				m.mode = NormalMode
				if m.width > 0 {
					leftWidth := m.width / 2
					rightWidth := m.width - leftWidth - 1
					sizeMsg := tea.WindowSizeMsg{Width: rightWidth, Height: m.height}
					updatedDiff, _ := m.diffViewer.Update(sizeMsg)
					if dv, ok := updatedDiff.(DiffViewerModel); ok {
						m.diffViewer = dv
					}
				}
				return m, nil
			case NormalMode:
				m.quitting = true
				return m, tea.Quit
			}
			// SearchMode: fall through to text input

		case "enter":
			switch m.mode {
			case SearchMode:
				if !m.searchLocked && len(m.filteredIndices) > 0 {
					m.searchLocked = true
					m.searchInput.Blur()
				} else if m.searchLocked && len(m.filteredIndices) > 0 {
					file := m.files[m.filteredIndices[m.searchSelected]]
					m.selectedFiles[file] = !m.selectedFiles[file]
				}
				return m, nil
			case NormalMode:
				if len(m.files) > 0 {
					file := m.files[m.currentIndex]
					m.selectedFiles[file] = !m.selectedFiles[file]
				}
				return m, nil
			}

		case "j", "down":
			switch m.mode {
			case SearchMode:
				if m.searchLocked && len(m.filteredIndices) > 0 {
					m.searchSelected = (m.searchSelected + 1) % len(m.filteredIndices)
					return m, m.loadCurrentDiff()
				}
				// Unlocked: fall through to text input
			case NormalMode:
				if len(m.files) > 0 {
					m.currentIndex = (m.currentIndex + 1) % len(m.files)
					m.adjustScrolling()
					return m, m.loadCurrentDiff()
				}
				return m, nil
			}

		case "k", "up":
			switch m.mode {
			case SearchMode:
				if m.searchLocked && len(m.filteredIndices) > 0 {
					m.searchSelected = (m.searchSelected - 1 + len(m.filteredIndices)) % len(m.filteredIndices)
					return m, m.loadCurrentDiff()
				}
				// Unlocked: fall through to text input
			case NormalMode:
				if len(m.files) > 0 {
					m.currentIndex = (m.currentIndex - 1 + len(m.files)) % len(m.files)
					m.adjustScrolling()
					return m, m.loadCurrentDiff()
				}
				return m, nil
			}
		}

		// DiffMode: forward remaining keys to the diff viewer
		if m.mode == DiffMode {
			updatedDiff, diffCmd := m.diffViewer.Update(msg)
			if dv, ok := updatedDiff.(DiffViewerModel); ok {
				m.diffViewer = dv
			}
			return m, diffCmd
		}

		// SearchMode unlocked: forward remaining keys to the text input
		if m.mode == SearchMode && !m.searchLocked {
			oldValue := m.searchInput.Value()
			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.searchInput.Value() != oldValue {
				m.searchQuery = m.searchInput.Value()
				m.performSearch()
			}
			return m, cmd
		}

		// NormalMode and locked SearchMode share file action keys
		inLockedSearch := m.mode == SearchMode && m.searchLocked
		if m.mode == NormalMode || inLockedSearch {
			if m.mode == NormalMode && m.quitting {
				return m, tea.Quit
			}
			switch msg.String() {
			case "ctrl+c":
				if m.mode == NormalMode {
					m.quitting = true
					return m, tea.Quit
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

			case "p":
				if m.operationInProgress || m.staged || len(m.files) == 0 {
					return m, nil
				}
				filePath := m.files[m.currentFileIdx()]
				patchCmd := exec.Command("git", "add", "-p", filePath)
				return m, tea.ExecProcess(patchCmd, func(err error) tea.Msg {
					return GitOperationCompleteMsg{
						success:       err == nil,
						error:         err,
						operation:     "patch",
						filesAffected: []string{filePath},
					}
				})

			case " ":
				if len(m.files) > 0 {
					m.mode = DiffMode
					// Expand diff viewer to full screen
					if m.width > 0 {
						sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
						updatedDiff, _ := m.diffViewer.Update(sizeMsg)
						if dv, ok := updatedDiff.(DiffViewerModel); ok {
							m.diffViewer = dv
						}
					}
				}
				return m, nil

			case "/":
				if m.mode == NormalMode {
					m.mode = SearchMode
					m.searchInput.Focus()
					m.searchInput.SetValue("")
				} else if inLockedSearch {
					m.searchLocked = false
					m.searchInput.Focus()
				}
				return m, nil

			case "g":
				if m.mode == NormalMode {
					m.currentIndex = 0
					m.scrollOffset = 0
					return m, m.loadCurrentDiff()
				}

			case "G":
				if m.mode == NormalMode && len(m.files) > 0 {
					m.currentIndex = len(m.files) - 1
					m.adjustScrolling()
					return m, m.loadCurrentDiff()
				}

			case "a":
				if inLockedSearch {
					for _, idx := range m.filteredIndices {
						m.selectedFiles[m.files[idx]] = true
					}
				} else {
					for _, file := range m.files {
						m.selectedFiles[file] = true
					}
				}

			case "A":
				m.selectedFiles = make(map[string]bool)

			case "s":
				m.splitPane = !m.splitPane

			case "tab":
				if m.mode == NormalMode && !m.operationInProgress {
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
					return m, m.loadCurrentDiff()
				}
			}
		}
	}

	return m, cmd
}

// currentFileIdx returns the index into m.files for the currently highlighted item.
func (m FilePickerModel) currentFileIdx() int {
	if m.mode == SearchMode && m.searchLocked && len(m.filteredIndices) > 0 {
		return m.filteredIndices[m.searchSelected]
	}
	return m.currentIndex
}

// loadCurrentDiff creates a new diff viewer for the currently highlighted file.
func (m *FilePickerModel) loadCurrentDiff() tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	filePath := m.files[m.currentFileIdx()]
	m.diffViewer = NewDiffViewerModel(m.repo, filePath)
	m.diffViewer.staged = m.staged
	// Re-apply the current pane size
	if m.width > 0 && m.height > 0 {
		rightWidth := m.width - m.width/2 - 1
		sizeMsg := tea.WindowSizeMsg{Width: rightWidth, Height: m.height}
		updatedDiff, _ := m.diffViewer.Update(sizeMsg)
		if dv, ok := updatedDiff.(DiffViewerModel); ok {
			m.diffViewer = dv
		}
	}
	return m.diffViewer.Init()
}

func (m FilePickerModel) View() string {
	if m.quitting {
		return ""
	}

	// Full-screen diff mode
	if m.mode == DiffMode {
		return m.diffViewer.View()
	}

	leftWidth := m.width / 2
	if leftWidth < 10 {
		leftWidth = m.width // fallback for very narrow terminals
	}

	// ── Left panel: file list ──────────────────────────────────────────────
	var leftSections []string
	var managing string
	if m.staged {
		managing = "Staged changes"
	} else {
		managing = "Unstaged changes"
	}
	leftSections = append(leftSections, m.titleStyle.Render("Files — "+managing))

	if m.showStatusMessage && m.lastOperationStatus != "" {
		statusStyle := m.checkedStyle
		if strings.HasPrefix(m.lastOperationStatus, "✗") {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		}
		leftSections = append(leftSections, statusStyle.Render(m.lastOperationStatus))
	}

	if m.operationInProgress {
		leftSections = append(leftSections, m.searchStyle.Render("⏳ Operation in progress..."))
	}

	if m.mode == SearchMode {
		if m.searchLocked {
			leftSections = append(leftSections, m.searchStyle.Render(fmt.Sprintf("Results for \"%s\":", m.searchQuery)))
		} else {
			leftSections = append(leftSections, m.searchStyle.Render("Search files:"))
			leftSections = append(leftSections, m.searchInput.View())
		}

		if m.searchQuery != "" {
			if len(m.filteredIndices) == 0 {
				leftSections = append(leftSections, m.unselectedStyle.Render("No matches found"))
			} else {
				leftSections = append(leftSections, m.searchStyle.Render(fmt.Sprintf("Results (%d):", len(m.filteredIndices))))
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
					statusChar := ""
					if m.showStatusChars && idx < len(m.fileStatuses) {
						statusChar = fmt.Sprintf("[%s] ", m.fileStatuses[idx].Status)
					}
					line := fmt.Sprintf("%s%s %s%s", prefix, checkbox, statusChar, file)
					leftSections = append(leftSections, style.Render(line))
				}
			}
		} else {
			leftSections = append(leftSections, m.unselectedStyle.Render("Type to search..."))
		}
	} else {
		selectedCount := len(m.getSelectedFiles())
		leftSections = append(leftSections, m.unselectedStyle.Render(fmt.Sprintf("(%d selected)", selectedCount)))
		leftSections = append(leftSections, "")

		startIdx := m.scrollOffset
		endIdx := min(startIdx+m.visibleLines, len(m.files))
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
			statusChar := ""
			if m.showStatusChars && i < len(m.fileStatuses) {
				statusChar = fmt.Sprintf("[%s] ", m.fileStatuses[i].Status)
			}
			line := fmt.Sprintf("%s%s %s%s", prefix, checkbox, statusChar, file)
			leftSections = append(leftSections, style.Render(line))
		}

		if len(m.files) > m.visibleLines {
			leftSections = append(leftSections, "")
			leftSections = append(leftSections, m.helpStyle.Render(fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.files))))
		}
	}

	if m.splitPane {
		leftPanel := lipgloss.NewStyle().Width(leftWidth).Render(strings.Join(leftSections, "\n"))
		separator := m.separatorStyle.Render(strings.Repeat("│\n", m.height))
		return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, m.diffViewer.View())
	}

	return lipgloss.NewStyle().Width(m.width).Render(strings.Join(leftSections, "\n"))
}

func (m *FilePickerModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}
	maxOffset := len(m.files) - m.visibleLines
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
	m.searchSelected = 0
}

func (m FilePickerModel) fuzzyMatch(text, query string) bool {
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

// SelectFiles provides an enhanced file picker with split-pane diff preview.
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

	if model, ok := finalModel.(FilePickerModel); ok {
		if model.confirmed {
			return model.getSelectedFiles(), model.removing, nil
		}
	}

	return []string{}, false, nil
}
