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

type rebaseCompleteMsg struct {
	err error
}

type RebasePickerModel struct {
	repo         *git.GitRepo
	entries      []git.RebaseEntry
	currentIndex int
	scrollOffset int
	visibleLines int
	width        int
	height       int

	statusMsg  string
	showStatus bool

	titleStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	helpStyle       lipgloss.Style
	successStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	actionStyles    map[string]lipgloss.Style
}

var actionOrder = []string{"pick", "reword", "edit", "squash", "fixup", "drop"}

func nextAction(current string) string {
	for i, a := range actionOrder {
		if a == current {
			return actionOrder[(i+1)%len(actionOrder)]
		}
	}
	return "pick"
}

func NewRebasePickerModel(repo *git.GitRepo, entries []git.RebaseEntry) RebasePickerModel {
	return RebasePickerModel{
		repo:    repo,
		entries: entries,

		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F1D3AB")).Bold(true),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		helpStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		successStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		errorStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		actionStyles: map[string]lipgloss.Style{
			"pick":   lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
			"reword": lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
			"edit":   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
			"squash": lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
			"fixup":  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
			"drop":   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		},
	}
}

func (m RebasePickerModel) Init() tea.Cmd {
	return nil
}

func (m RebasePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleLines = msg.Height - 7

	case rebaseCompleteMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("✗ Rebase failed: %v", msg.err)
		} else {
			m.statusMsg = "✓ Rebase complete"
		}
		m.showStatus = true
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "j", "down":
			if len(m.entries) > 0 {
				m.currentIndex = (m.currentIndex + 1) % len(m.entries)
				m.adjustScrolling()
			}

		case "k", "up":
			if len(m.entries) > 0 {
				m.currentIndex = (m.currentIndex - 1 + len(m.entries)) % len(m.entries)
				m.adjustScrolling()
			}

		case "p":
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = "pick"
			}

		case "r":
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = "reword"
			}

		case "e":
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = "edit"
			}

		case "s":
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = "squash"
			}

		case "f":
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = "fixup"
			}

		case "d":
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = "drop"
			}

		case " ":
			// Cycle through actions
			if len(m.entries) > 0 {
				m.entries[m.currentIndex].Action = nextAction(m.entries[m.currentIndex].Action)
			}

		case "enter", "x":
			if len(m.entries) > 0 {
				return m, m.executeRebase()
			}
		}
	}

	return m, nil
}

func (m RebasePickerModel) View() string {
	var sections []string
	sections = append(sections, m.titleStyle.Render(fmt.Sprintf("Interactive Rebase (%d commits)", len(m.entries))))
	sections = append(sections, m.helpStyle.Render("oldest → newest (top = applied first)"))

	if m.showStatus {
		style := m.successStyle
		if strings.HasPrefix(m.statusMsg, "✗") {
			style = m.errorStyle
		}
		sections = append(sections, style.Render(m.statusMsg))
	}

	sections = append(sections, "")

	startIdx := m.scrollOffset
	endIdx := min(startIdx+m.visibleLines, len(m.entries))

	for i := startIdx; i < endIdx; i++ {
		e := m.entries[i]
		prefix := "  "
		subjectStyle := m.unselectedStyle
		if i == m.currentIndex {
			prefix = "> "
			subjectStyle = m.selectedStyle
		}
		actionStyle, ok := m.actionStyles[e.Action]
		if !ok {
			actionStyle = m.helpStyle
		}
		action := actionStyle.Render(fmt.Sprintf("%-6s", e.Action))
		line := fmt.Sprintf("%s%s  %s  %s", prefix, action, m.helpStyle.Render(e.Hash), subjectStyle.Render(e.Subject))
		sections = append(sections, line)
	}

	if len(m.entries) > m.visibleLines {
		sections = append(sections, "")
		sections = append(sections, m.helpStyle.Render(fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.entries))))
	}

	sections = append(sections, "")
	sections = append(sections, m.helpStyle.Render("p: pick  r: reword  e: edit  s: squash  f: fixup  d: drop  space: cycle"))
	sections = append(sections, m.helpStyle.Render("enter/x: execute  q: quit"))

	return strings.Join(sections, "\n")
}

func (m RebasePickerModel) executeRebase() tea.Cmd {
	return func() tea.Msg {
		// Build todo content (newest first from git log, but rebase needs oldest first)
		reversed := make([]git.RebaseEntry, len(m.entries))
		for i, e := range m.entries {
			reversed[len(m.entries)-1-i] = e
		}

		var lines []string
		for _, e := range reversed {
			lines = append(lines, fmt.Sprintf("%s %s %s", e.Action, e.Hash, e.Subject))
		}
		content := strings.Join(lines, "\n") + "\n"

		tmpFile, err := os.CreateTemp("", "cgit-rebase-*.txt")
		if err != nil {
			return rebaseCompleteMsg{err: err}
		}
		tmpPath := tmpFile.Name()
		if _, err := tmpFile.WriteString(content); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return rebaseCompleteMsg{err: err}
		}
		tmpFile.Close()

		// Normalise path separators for Git for Windows bash cp command
		tmpPathNorm := strings.ReplaceAll(tmpPath, `\`, `/`)
		seqEditor := fmt.Sprintf("cp '%s'", tmpPathNorm)

		count := len(m.entries)
		rebaseCmd := exec.Command("git",
			"-c", "sequence.editor="+seqEditor,
			"rebase", "-i", fmt.Sprintf("HEAD~%d", count),
		)
		rebaseCmd.Dir = m.repo.WorkDir
		rebaseCmd.Env = os.Environ()

		var errBuf strings.Builder
		rebaseCmd.Stderr = &errBuf

		runErr := rebaseCmd.Run()
		os.Remove(tmpPath)
		if runErr != nil {
			return rebaseCompleteMsg{err: fmt.Errorf("%w\n%s", runErr, errBuf.String())}
		}
		return rebaseCompleteMsg{}
	}
}

func (m *RebasePickerModel) adjustScrolling() {
	if m.visibleLines <= 0 {
		return
	}
	if m.currentIndex >= m.scrollOffset+m.visibleLines {
		m.scrollOffset = m.currentIndex - m.visibleLines + 1
	}
	if m.currentIndex < m.scrollOffset {
		m.scrollOffset = m.currentIndex
	}
	maxOffset := len(m.entries) - m.visibleLines
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

func StartRebasePicker(repo *git.GitRepo, limit int) error {
	entries, err := repo.GetRebaseCommits(limit)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No commits to rebase.")
		return nil
	}
	m := NewRebasePickerModel(repo, entries)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
