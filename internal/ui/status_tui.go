package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type PanelType int

const (
	UnstagedPanel PanelType = iota
	StagedPanel
	DiffPanel
	CommitPanel
)

type StatusModel struct {
	repo         *git.GitRepo
	repoStatus   *git.RepoStatus
	currentPanel PanelType
	selectedIndex int
	viewport     viewport.Model
	showDiff     bool
	diffContent  string
	commitInput  textinput.Model
	showCommit   bool
	width        int
	height       int
	quitting     bool
	message      string
	messageTime  time.Time
	
	// Styles
	titleStyle        lipgloss.Style
	panelStyle        lipgloss.Style
	selectedStyle     lipgloss.Style
	unselectedStyle   lipgloss.Style
	headerStyle       lipgloss.Style
	helpStyle         lipgloss.Style
	messageStyle      lipgloss.Style
	
	// Diff styles
	diffAddedStyle    lipgloss.Style
	diffRemovedStyle  lipgloss.Style
	diffHeaderStyle   lipgloss.Style
	diffHunkStyle     lipgloss.Style
}

type refreshMsg struct{}
type statusMsg *git.RepoStatus
type diffMsg string
type messageTimeoutMsg struct{}

func NewStatusModel(repo *git.GitRepo) StatusModel {
	vp := viewport.New(0, 0)
	
	ci := textinput.New()
	ci.Placeholder = "Enter commit message..."
	ci.CharLimit = 500
	ci.Width = 50
	
	return StatusModel{
		repo:         repo,
		viewport:     vp,
		commitInput:  ci,
		currentPanel: UnstagedPanel,
		
		// Initialize styles
		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		
		panelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
		
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		
		unselectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		
		messageStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		
		// Diff syntax highlighting styles
		diffAddedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")), // Green for additions
		
		diffRemovedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red for deletions
		
		diffHeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true), // Blue for headers
		
		diffHunkStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")), // Orange for hunk headers
	}
}

func (m StatusModel) Init() tea.Cmd {
	return tea.Batch(m.refreshStatus, textinput.Blink)
}

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 12
		
	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}
		
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		
		case "esc":
			if m.showCommit {
				m.showCommit = false
				m.commitInput.SetValue("")
				return m, nil
			} else if m.showDiff {
				m.showDiff = false
				m.diffContent = ""
				return m, nil
			}
			
		case "r":
			if !m.showCommit {
				m.showMessage("Refreshing...")
				return m, tea.Batch(m.refreshStatus, m.scheduleMessageTimeout())
			}
		
		case "enter":
			if m.showCommit {
				// Commit with the entered message
				message := m.commitInput.Value()
				if message != "" {
					return m, m.performCommit(message)
				}
			} else {
				return m, m.showFileDiff
			}
			
		case "h", "left":
			if !m.showCommit && m.currentPanel == StagedPanel {
				m.currentPanel = UnstagedPanel
				m.selectedIndex = 0
			}
			
		case "l", "right":
			if !m.showCommit && m.currentPanel == UnstagedPanel {
				m.currentPanel = StagedPanel  
				m.selectedIndex = 0
			}
			
		case "j", "down":
			if !m.showCommit {
				m.moveDown()
			}
			
		case "k", "up":
			if !m.showCommit {
				m.moveUp()
			}
			
		case "g":
			if !m.showCommit {
				m.selectedIndex = 0
			}
			
		case "G":
			if !m.showCommit {
				if m.currentPanel == UnstagedPanel && m.repoStatus != nil {
					m.selectedIndex = len(m.repoStatus.UnstagedFiles) - 1
				} else if m.currentPanel == StagedPanel && m.repoStatus != nil {
					m.selectedIndex = len(m.repoStatus.StagedFiles) - 1
				}
			}
			
		case "s", " ":
			if !m.showCommit {
				return m, m.stageFile
			}
			
		case "u":
			if !m.showCommit {
				return m, m.unstageFile
			}
			
		case "d":
			if !m.showCommit {
				return m, m.discardChanges
			}
			
		case "a":
			if !m.showCommit {
				return m, m.stageAllFiles
			}
			
		case "c":
			if !m.showCommit {
				// Check if there are staged files
				if m.repoStatus != nil && len(m.repoStatus.StagedFiles) > 0 {
					m.showCommit = true
					m.commitInput.Focus()
					m.commitInput.SetValue("")
					return m, nil
				}
			}
		
		case "p":
			if !m.showCommit {
				return m, m.pushChanges()
			}
		}
		
	case statusMsg:
		m.repoStatus = msg
		if m.selectedIndex >= m.getCurrentFileCount() {
			m.selectedIndex = max(0, m.getCurrentFileCount()-1)
		}
		
	case diffMsg:
		m.diffContent = string(msg)
		m.showDiff = true
		m.viewport.SetContent(m.diffContent)
		
	case refreshMsg:
		return m, m.refreshStatus
		
	case messageTimeoutMsg:
		if time.Since(m.messageTime) >= 3*time.Second {
			m.message = ""
		}
		return m, nil
	
	case error:
		m.showMessage(msg.Error())
		if m.showCommit {
			// Stay in commit mode on error
		}
		return m, tea.Batch(m.refreshStatus, m.scheduleMessageTimeout())
		
	case string:
		switch msg {
			case "commit_success":
				m.showCommit = false
				m.commitInput.SetValue("")
				m.commitInput.Blur()
				m.showMessage("Commit successful!")
				return m, tea.Batch(m.refreshStatus, m.scheduleMessageTimeout())
			case "push_success":
				m.showMessage("Push successful!")
				return m, tea.Batch(m.refreshStatus, m.scheduleMessageTimeout())
		}
	}
	
	// Only update text input/viewport for actual user input messages
	switch msg.(type) {
	case tea.KeyMsg, tea.WindowSizeMsg:
		// Update text input if in commit mode
		if m.showCommit {
			m.commitInput, cmd = m.commitInput.Update(msg)
			return m, cmd
		}
		
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	
	return m, nil
}

func (m StatusModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	
	if m.repoStatus == nil {
		return "Loading repository status..."
	}
	
	var sections []string
	
	// Header
	header := m.renderHeader()
	sections = append(sections, header)
	
	if m.showCommit {
		// Show commit input view
		sections = append(sections, m.renderCommitView())
	} else if m.showDiff {
		// Show diff view
		sections = append(sections, m.renderDiffView())
	} else {
		// Show main status view
		sections = append(sections, m.renderMainView())
	}
	
	// Help and message
	help := m.renderHelp()
	sections = append(sections, help)
	
	if m.message != "" {
		msg := m.messageStyle.Render(m.message)
		sections = append(sections, msg)
	}
	
	return strings.Join(sections, "\n")
}

func (m StatusModel) renderHeader() string {
	if m.repoStatus == nil {
		return ""
	}
	
	// Branch info with tracking
	branchInfo := fmt.Sprintf("Branch: %s", m.repoStatus.CurrentBranch)
	if m.repoStatus.Ahead > 0 || m.repoStatus.Behind > 0 {
		branchInfo += fmt.Sprintf(" (↑%d ↓%d)", m.repoStatus.Ahead, m.repoStatus.Behind)
	}
	
	// Last commit info
	commitInfo := ""
	if m.repoStatus.LastCommit.Hash != "" {
		commitInfo = fmt.Sprintf("Last: %s %s", 
			m.repoStatus.LastCommit.Hash, 
			m.repoStatus.LastCommit.Message)
	}
	
	left := m.headerStyle.Render(branchInfo)
	right := m.headerStyle.Render(commitInfo)
	
	// Center the content
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right)
}

func (m StatusModel) renderMainView() string {
	unstagedPanel := m.renderUnstagedPanel()
	stagedPanel := m.renderStagedPanel()
	
	return lipgloss.JoinHorizontal(lipgloss.Top, unstagedPanel, " ", stagedPanel)
}

func (m StatusModel) renderUnstagedPanel() string {
	title := "Unstaged Changes"
	if len(m.repoStatus.UnstagedFiles) > 0 {
		title += fmt.Sprintf(" (%d)", len(m.repoStatus.UnstagedFiles))
	}
	
	var content strings.Builder
	content.WriteString(m.titleStyle.Render(title) + "\n")
	
	if len(m.repoStatus.UnstagedFiles) == 0 {
		content.WriteString(m.unselectedStyle.Render("  (no unstaged changes)"))
	} else {
		for i, file := range m.repoStatus.UnstagedFiles {
			prefix := "  "
			style := m.unselectedStyle
			
			if m.currentPanel == UnstagedPanel && i == m.selectedIndex {
				prefix = "> "
				style = m.selectedStyle
			}
			
			statusChar := m.getStatusChar(file.Status)
			line := fmt.Sprintf("%s%s [%s]", prefix, file.Path, statusChar)
			content.WriteString(style.Render(line) + "\n")
		}
	}
	
	panelWidth := (m.width - 3) / 2
	return m.panelStyle.Width(panelWidth).Render(content.String())
}

func (m StatusModel) renderStagedPanel() string {
	title := "Staged Changes"
	if len(m.repoStatus.StagedFiles) > 0 {
		title += fmt.Sprintf(" (%d)", len(m.repoStatus.StagedFiles))
	}
	
	var content strings.Builder
	content.WriteString(m.titleStyle.Render(title) + "\n")
	
	if len(m.repoStatus.StagedFiles) == 0 {
		content.WriteString(m.unselectedStyle.Render("  (no staged changes)"))
	} else {
		for i, file := range m.repoStatus.StagedFiles {
			prefix := "  "
			style := m.unselectedStyle
			
			if m.currentPanel == StagedPanel && i == m.selectedIndex {
				prefix = "> "
				style = m.selectedStyle
			}
			
			statusChar := m.getStatusChar(file.Status)
			line := fmt.Sprintf("%s%s [%s]", prefix, file.Path, statusChar)
			content.WriteString(style.Render(line) + "\n")
		}
	}
	
	panelWidth := (m.width - 3) / 2
	return m.panelStyle.Width(panelWidth).Render(content.String())
}

func (m StatusModel) renderCommitView() string {
	var content strings.Builder
	
	// Title
	title := m.titleStyle.Render("Commit Changes")
	content.WriteString(title + "\n\n")
	
	// Show staged files
	if m.repoStatus != nil && len(m.repoStatus.StagedFiles) > 0 {
		content.WriteString(m.headerStyle.Render("Files to be committed:") + "\n")
		for _, file := range m.repoStatus.StagedFiles {
			statusChar := m.getStatusChar(file.Status)
			line := fmt.Sprintf("  %s [%s]", file.Path, statusChar)
			content.WriteString(m.unselectedStyle.Render(line) + "\n")
		}
		content.WriteString("\n")
	}
	
	// Commit message input
	content.WriteString(m.headerStyle.Render("Commit message:") + "\n")
	content.WriteString(m.commitInput.View() + "\n")
	
	return content.String()
}

func (m StatusModel) renderDiffView() string {
	if m.diffContent == "" {
		return m.viewport.View()
	}
	
	// Apply syntax highlighting to diff content
	highlightedContent := m.highlightDiff(m.diffContent)
	m.viewport.SetContent(highlightedContent)
	
	return m.viewport.View()
}

func (m StatusModel) highlightDiff(content string) string {
	lines := strings.Split(content, "\n")
	var highlightedLines []string
	
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			// Added lines (green)
			highlightedLines = append(highlightedLines, m.diffAddedStyle.Render(line))
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			// Removed lines (red)
			highlightedLines = append(highlightedLines, m.diffRemovedStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			// Hunk headers (orange)
			highlightedLines = append(highlightedLines, m.diffHunkStyle.Render(line))
		case strings.HasPrefix(line, "diff --git") || 
		     strings.HasPrefix(line, "index ") ||
		     strings.HasPrefix(line, "---") ||
		     strings.HasPrefix(line, "+++"):
			// Diff headers (blue)
			highlightedLines = append(highlightedLines, m.diffHeaderStyle.Render(line))
		default:
			// Context lines (default color)
			highlightedLines = append(highlightedLines, line)
		}
	}
	
	return strings.Join(highlightedLines, "\n")
}

func (m StatusModel) renderHelp() string {
	if m.showCommit {
		return m.helpStyle.Render("enter: commit | esc: cancel | q: quit")
	} else if m.showDiff {
		return m.helpStyle.Render("esc: back | q: quit")
	}
	
	help := "j/k: nav | h/l: panels | s: stage | u: unstage | d: discard | a: stage all | c: commit | p: push | enter: diff | r: refresh | q: quit"
	return m.helpStyle.Render(help)
}

func (m StatusModel) getStatusChar(status string) string {
	switch status {
	case "M":
		return "M"
	case "A":
		return "A"
	case "D":
		return "D"
	case "R":
		return "R"
	case "?":
		return "?"
	default:
		return status
	}
}

func (m StatusModel) getCurrentFileCount() int {
	if m.repoStatus == nil {
		return 0
	}
	
	if m.currentPanel == UnstagedPanel {
		return len(m.repoStatus.UnstagedFiles)
	}
	return len(m.repoStatus.StagedFiles)
}

func (m *StatusModel) moveDown() {
	count := m.getCurrentFileCount()
	if count > 0 {
		m.selectedIndex = (m.selectedIndex + 1) % count
	}
}

func (m *StatusModel) moveUp() {
	count := m.getCurrentFileCount()
	if count > 0 {
		m.selectedIndex = (m.selectedIndex - 1 + count) % count
	}
}

func (m *StatusModel) showMessage(msg string) {
	m.message = msg
	m.messageTime = time.Now()
}

func (m StatusModel) scheduleMessageTimeout() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return messageTimeoutMsg{}
	})
}

func (m StatusModel) refreshStatus() tea.Msg {
	status, err := m.repo.GetRepositoryStatus()
	if err != nil {
		return statusMsg(nil)
	}
	return statusMsg(status)
}

func (m StatusModel) stageFile() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != UnstagedPanel || 
	   m.selectedIndex >= len(m.repoStatus.UnstagedFiles) {
		return refreshMsg{}
	}
	
	file := m.repoStatus.UnstagedFiles[m.selectedIndex]
	err := m.repo.StageFile(file.Path)
	if err != nil {
		// Handle error
		return refreshMsg{}
	}
	
	return refreshMsg{}
}

func (m StatusModel) unstageFile() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != StagedPanel || 
	   m.selectedIndex >= len(m.repoStatus.StagedFiles) {
		return refreshMsg{}
	}
	
	file := m.repoStatus.StagedFiles[m.selectedIndex]
	err := m.repo.UnstageFile(file.Path, file.Status)
	if err != nil {
		// Handle error - show error message
		return fmt.Errorf("failed to unstage file: %v", err)
	}
	
	return refreshMsg{}
}

func (m StatusModel) discardChanges() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != UnstagedPanel || 
	   m.selectedIndex >= len(m.repoStatus.UnstagedFiles) {
		return refreshMsg{}
	}
	
	file := m.repoStatus.UnstagedFiles[m.selectedIndex]
	err := m.repo.DiscardChanges(file.Path, file.Status)
	if err != nil {
		// Handle error - show error message
		return fmt.Errorf("failed to discard changes: %v", err)
	}
	
	return refreshMsg{}
}

func (m StatusModel) stageAllFiles() tea.Msg {
	err := m.repo.StageAllFiles()
	if err != nil {
		// Handle error
		return refreshMsg{}
	}
	
	return refreshMsg{}
}

func (m StatusModel) pushChanges() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Push()
		if err != nil {
			return fmt.Errorf("push failed: %v", err)
		}
		return "push_success"
	}
}


func (m StatusModel) performCommit(message string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Commit(message)
		if err != nil {
			return fmt.Errorf("commit failed: %v", err)
		}
		return "commit_success"
	}
}

func (m StatusModel) showFileDiff() tea.Msg {
	if m.repoStatus == nil {
		return refreshMsg{}
	}
	
	var filePath string
	var staged bool
	
	if m.currentPanel == UnstagedPanel && m.selectedIndex < len(m.repoStatus.UnstagedFiles) {
		filePath = m.repoStatus.UnstagedFiles[m.selectedIndex].Path
		staged = false
	} else if m.currentPanel == StagedPanel && m.selectedIndex < len(m.repoStatus.StagedFiles) {
		filePath = m.repoStatus.StagedFiles[m.selectedIndex].Path
		staged = true
	}
	
	if filePath == "" {
		return refreshMsg{}
	}
	
	diff, err := m.repo.GetFileDiff(filePath, staged)
	if err != nil {
		return diffMsg("Error getting diff: " + err.Error())
	}
	
	return diffMsg(diff)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func StartStatusTUI(repo *git.GitRepo) error {
	m := NewStatusModel(repo)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}