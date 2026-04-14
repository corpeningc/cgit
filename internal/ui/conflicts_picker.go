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

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	errorStyle      lipgloss.Style
	helpStyle       lipgloss.Style
	successStyle    lipgloss.Style

	lastStatus     string
	showLastStatus bool
}

func NewConflictsPickerModel(repo *git.GitRepo, files []git.FileStatus) ConflictsPickerModel {
	return ConflictsPickerModel{
		repo:  repo,
		files: files,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		errorStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		helpStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		successStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
	}
}

func (m ConflictsPickerModel) Init() tea.Cmd {
	return nil
}

func (m ConflictsPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j", "down":
			if len(m.files) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.files)
			}

		case "k", "up":
			if len(m.files) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.files)) % len(m.files)
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
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi"
				}
				editorCmd := exec.Command(editor, filePath)
				return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
					if err != nil {
						return conflictResolvedMsg{filePath: filePath, err: err}
					}
					// Stage the file after editing
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
	var sections []string
	sections = append(sections, m.titleStyle.Render(fmt.Sprintf("Merge Conflicts (%d remaining)", len(m.files))))

	if m.showLastStatus {
		style := m.successStyle
		if strings.HasPrefix(m.lastStatus, "✗") {
			style = m.errorStyle
		}
		sections = append(sections, style.Render(m.lastStatus))
	}

	sections = append(sections, "")

	for i, f := range m.files {
		prefix := "  "
		style := m.unselectedStyle
		if i == m.currentIndex {
			prefix = "> "
			style = m.selectedStyle
		}
		line := fmt.Sprintf("%s[%s] %s", prefix, f.Status, f.Path)
		sections = append(sections, style.Render(line))
	}

	sections = append(sections, "")
	sections = append(sections, m.helpStyle.Render("j/k: navigate | o: keep ours | t: keep theirs | e: open in $EDITOR | q: quit"))

	return strings.Join(sections, "\n")
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
