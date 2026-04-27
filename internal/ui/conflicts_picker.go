package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type conflictResolvedMsg struct {
	filePath string
	err      error
}

type conflictRefreshMsg struct {
	files []git.FileStatus
	err   error
}

type ConflictsPickerModel struct {
	repo         *git.GitRepo
	files        []git.FileStatus
	currentIndex int
	width        int
	height       int

	diffViewer DiffViewerModel

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	errorStyle      lipgloss.Style
	helpStyle       lipgloss.Style
	successStyle    lipgloss.Style
	separatorStyle  lipgloss.Style

	lastStatus     string
	showLastStatus bool
}

func NewConflictsPickerModel(repo *git.GitRepo, files []git.FileStatus) ConflictsPickerModel {
	m := ConflictsPickerModel{
		repo:  repo,
		files: files,

		titleStyle:      TitlePinkStyle,
		selectedStyle:   SelectedPeachStyle,
		unselectedStyle: UnselectedStyle,
		errorStyle:      ErrorStyle,
		helpStyle:       HelpStyle,
		successStyle:    SuccessStyle,
		separatorStyle:  SeparatorStyle,
	}

	if len(files) > 0 {
		m.diffViewer = NewDiffViewerModel(repo, files[0].Path)
	}

	return m
}

func (m ConflictsPickerModel) Init() tea.Cmd {
	if len(m.files) > 0 {
		return m.loadCurrentContent()
	}
	return nil
}

func (m ConflictsPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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

	case conflictResolvedMsg:
		if msg.err != nil {
			m.lastStatus = fmt.Sprintf("✗ %s: %v", msg.filePath, msg.err)
		} else {
			m.lastStatus = fmt.Sprintf("✓ Resolved %s", msg.filePath)
		}
		m.showLastStatus = true
		return m, m.refresh()

	case conflictRefreshMsg:
		if msg.err != nil {
			m.lastStatus = fmt.Sprintf("✗ Refresh failed: %v", msg.err)
			m.showLastStatus = true
			return m, nil
		}
		m.files = msg.files
		if len(m.files) == 0 {
			fmt.Println("\nAll conflicts resolved!")
			return m, tea.Quit
		}
		if m.currentIndex >= len(m.files) {
			m.currentIndex = len(m.files) - 1
		}
		return m, m.loadCurrentContent()

	case tea.KeyMsg:
		// Diff panel scroll keys
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

		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j", "down":
			if len(m.files) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.files)
				return m, m.loadCurrentContent()
			}

		case "k", "up":
			if len(m.files) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.files)) % len(m.files)
				return m, m.loadCurrentContent()
			}

		case "o":
			if len(m.files) > 0 {
				filePath := m.files[m.currentIndex].Path
				return m, m.resolveOurs(filePath)
			}

		case "t":
			if len(m.files) > 0 {
				filePath := m.files[m.currentIndex].Path
				return m, m.resolveTheirs(filePath)
			}

		case "e":
			if len(m.files) > 0 {
				filePath := m.files[m.currentIndex].Path
				editor := resolveEditor()
				editorCmd := exec.Command(editor, filePath)
				return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
					if err != nil {
						return conflictResolvedMsg{filePath: filePath, err: err}
					}
					addCmd := exec.Command("git", "add", filePath)
					addErr := addCmd.Run()
					return conflictResolvedMsg{filePath: filePath, err: addErr}
				})
			}
		}
	}

	return m, nil
}

func (m ConflictsPickerModel) View() string {
	leftWidth := m.width / 2
	if leftWidth < 10 {
		leftWidth = m.width
	}

	// ── Left panel ────────────────────────────────────────────────────────
	var left []string
	left = append(left, m.titleStyle.Render(fmt.Sprintf("Merge Conflicts (%d remaining)", len(m.files))))

	if m.showLastStatus {
		style := m.successStyle
		if strings.HasPrefix(m.lastStatus, "✗") {
			style = m.errorStyle
		}
		left = append(left, style.Render(m.lastStatus))
	}

	left = append(left, "")

	for i, f := range m.files {
		prefix := "  "
		style := m.unselectedStyle
		if i == m.currentIndex {
			prefix = "> "
			style = m.selectedStyle
		}
		left = append(left, style.Render(fmt.Sprintf("%s[%s] %s", prefix, f.Status, f.Path)))
	}

	leftPanel := lipgloss.NewStyle().Width(leftWidth).Render(strings.Join(left, "\n"))
	separator := m.separatorStyle.Render(strings.Repeat("│\n", m.height))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, m.diffViewer.View())
}

// loadCurrentContent loads the conflict content for the currently selected file.
func (m *ConflictsPickerModel) loadCurrentContent() tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	filePath := m.files[m.currentIndex].Path
	m.diffViewer = NewDiffViewerModel(m.repo, filePath)
	if m.width > 0 {
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth - 1
		sizeMsg := tea.WindowSizeMsg{Width: rightWidth, Height: m.height}
		updatedDiff, _ := m.diffViewer.Update(sizeMsg)
		if dv, ok := updatedDiff.(DiffViewerModel); ok {
			m.diffViewer = dv
		}
	}
	repo := m.repo
	return func() tea.Msg {
		content, err := repo.GetConflictContent(filePath)
		return diffLoadedMsg{content: content, err: err}
	}
}

func (m ConflictsPickerModel) resolveOurs(filePath string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.ResolveConflictOurs(filePath)
		return conflictResolvedMsg{filePath: filePath, err: err}
	}
}

func (m ConflictsPickerModel) resolveTheirs(filePath string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.ResolveConflictTheirs(filePath)
		return conflictResolvedMsg{filePath: filePath, err: err}
	}
}

func (m ConflictsPickerModel) refresh() tea.Cmd {
	return func() tea.Msg {
		files, err := m.repo.GetConflictedFiles()
		return conflictRefreshMsg{files: files, err: err}
	}
}

// resolveEditor returns the best available editor, preferring $EDITOR then nvim, vim, vi.
func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	for _, candidate := range []string{"nvim", "vim", "vi"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return "vi" // last resort — let the OS error surface naturally
}

func StartConflictsPicker(repo *git.GitRepo) error {
	files, err := repo.GetConflictedFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No conflicts.")
		return nil
	}

	m := NewConflictsPickerModel(repo, files)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
