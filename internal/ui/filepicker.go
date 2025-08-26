package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type FilePickerMode int

const (
	NormalMode FilePickerMode = iota
	SearchMode
)

type FilePickerModel struct {
	files           []string
	fileStatuses    []git.FileStatus
	selectedFiles   map[string]bool
	currentIndex    int
	mode           FilePickerMode
	searchInput    textinput.Model
	searchQuery    string
	filteredIndices []int
	searchSelected  int
	quitting       bool
	confirmed      bool
	width          int
	height         int
	showStatusChars bool
	
	// Styles
	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	checkedStyle    lipgloss.Style
	helpStyle       lipgloss.Style
	searchStyle     lipgloss.Style
}

func NewFilePickerModel(files []string) FilePickerModel {
	si := textinput.New()
	si.Placeholder = "Search files..."
	si.CharLimit = 100
	si.Width = 50
	
	return FilePickerModel{
		files:         files,
		selectedFiles: make(map[string]bool),
		searchInput:   si,
		
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

func NewFilePickerModelWithStatus(fileStatuses []git.FileStatus) FilePickerModel {
	si := textinput.New()
	si.Placeholder = "Search files..."
	si.CharLimit = 100
	si.Width = 50
	
	var files []string
	for _, status := range fileStatuses {
		files = append(files, status.Path)
	}
	
	return FilePickerModel{
		files:           files,
		fileStatuses:    fileStatuses,
		selectedFiles:   make(map[string]bool),
		searchInput:     si,
		showStatusChars: true,
		
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
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}
		
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
			
		case "esc":
			if m.mode == SearchMode {
				m.mode = NormalMode
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.filteredIndices = nil
				m.searchSelected = 0
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
			
		case "enter":
			if m.mode == SearchMode {
				// Select search result and exit search
				if len(m.filteredIndices) > 0 && m.searchSelected < len(m.filteredIndices) {
					m.currentIndex = m.filteredIndices[m.searchSelected]
				}
				m.mode = NormalMode
				m.searchInput.SetValue("")
				return m, nil
			} else {
				// Confirm selection
				m.confirmed = true
				m.quitting = true
				return m, tea.Quit
			}
			
		case "/":
			if m.mode == NormalMode {
				m.mode = SearchMode
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				return m, nil
			}
			
		case "j", "down":
			if m.mode == SearchMode {
				// Navigate down in search results
				if len(m.filteredIndices) > 0 {
					m.searchSelected = (m.searchSelected + 1) % len(m.filteredIndices)
				}
			} else {
				// Navigate down in file list
				if len(m.files) > 0 {
					m.currentIndex = (m.currentIndex + 1) % len(m.files)
				}
			}
			
		case "k", "up":
			if m.mode == SearchMode {
				// Navigate up in search results
				if len(m.filteredIndices) > 0 {
					m.searchSelected = (m.searchSelected - 1 + len(m.filteredIndices)) % len(m.filteredIndices)
				}
			} else {
				// Navigate up in file list
				if len(m.files) > 0 {
					m.currentIndex = (m.currentIndex - 1 + len(m.files)) % len(m.files)
				}
			}
			
		case "g":
			if m.mode == NormalMode {
				m.currentIndex = 0
			}
			
		case "G":
			if m.mode == NormalMode && len(m.files) > 0 {
				m.currentIndex = len(m.files) - 1
			}
			
		case " ", "s":
			if m.mode == NormalMode && len(m.files) > 0 {
				// Toggle selection
				file := m.files[m.currentIndex]
				m.selectedFiles[file] = !m.selectedFiles[file]
			}
			
		case "a":
			if m.mode == NormalMode {
				// Select all files
				for _, file := range m.files {
					m.selectedFiles[file] = true
				}
			}
			
		case "A":
			if m.mode == NormalMode {
				// Deselect all files
				m.selectedFiles = make(map[string]bool)
			}
		}
	}
	
	// Update search input if in search mode
	if m.mode == SearchMode {
		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Perform real-time search if input changed
		if m.searchInput.Value() != oldValue {
			m.searchQuery = m.searchInput.Value()
			m.performSearch()
		}
		return m, cmd
	}
	
	return m, cmd
}

func (m FilePickerModel) View() string {
	if m.quitting {
		return ""
	}
	
	var sections []string
	
	// Title
	title := m.titleStyle.Render("Select files to add")
	sections = append(sections, title)
	
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
		// Show file list
		selectedCount := len(m.getSelectedFiles())
		subtitle := fmt.Sprintf("(%d selected)", selectedCount)
		sections = append(sections, m.unselectedStyle.Render(subtitle))
		sections = append(sections, "")
		
		// Show files
		for i, file := range m.files {
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
	}
	
	// Help
	help := ""
	if m.mode == SearchMode {
		help = "j/k: navigate | space: select | enter: go to file | esc: back | q: quit"
	} else {
		help = "j/k: navigate | /: search | space: select | a: select all | A: deselect all | enter: confirm | q: quit"
	}
	sections = append(sections, "")
	sections = append(sections, m.helpStyle.Render(help))
	
	return strings.Join(sections, "\n")
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

func SelectFiles(files []string) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}
	
	m := NewFilePickerModel(files)
	p := tea.NewProgram(m)
	
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	
	// Type assert to get our model back
	if model, ok := finalModel.(FilePickerModel); ok {
		if model.confirmed {
			return model.getSelectedFiles(), nil
		}
	}
	
	// User quit without confirming
	return []string{}, nil
}

// SelectFilesWithSearch provides an enhanced file picker with fuzzy search capabilities
func SelectFilesWithSearch(files []string) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}
	
	m := NewFilePickerModel(files)
	p := tea.NewProgram(m, tea.WithAltScreen())
	
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	
	// Type assert to get our model back
	if model, ok := finalModel.(FilePickerModel); ok {
		if model.confirmed {
			return model.getSelectedFiles(), nil
		}
	}
	
	// User quit without confirming
	return []string{}, nil
}

// SelectUnstagedFilesWithSearch provides an enhanced file picker specifically for unstaged files with status display
func SelectUnstagedFilesWithSearch(fileStatuses []git.FileStatus) ([]string, error) {
	if len(fileStatuses) == 0 {
		return []string{}, nil
	}
	
	m := NewFilePickerModelWithStatus(fileStatuses)
	p := tea.NewProgram(m, tea.WithAltScreen())
	
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	
	// Type assert to get our model back
	if model, ok := finalModel.(FilePickerModel); ok {
		if model.confirmed {
			return model.getSelectedFiles(), nil
		}
	}
	
	// User quit without confirming
	return []string{}, nil
}
