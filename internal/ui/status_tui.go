package ui

import (
	"fmt"
	"strings"
	"time"

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
)

type StatusModel struct {
	repo         *git.GitRepo
	repoStatus   *git.RepoStatus
	currentPanel PanelType
	selectedIndex int
	viewport     viewport.Model
	showDiff     bool
	diffContent  string
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
}

type refreshMsg struct{}
type statusMsg *git.RepoStatus
type diffMsg string
type messageTimeoutMsg struct{}

func NewStatusModel(repo *git.GitRepo) StatusModel {
	vp := viewport.New(0, 0)
	
	return StatusModel{
		repo:         repo,
		viewport:     vp,
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
	}
}

func (m StatusModel) Init() tea.Cmd {
	return m.refreshStatus
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
			if m.showDiff {
				m.showDiff = false
				m.diffContent = ""
				return m, nil
			}
			
		case "r":
			m.showMessage("Refreshing...")
			return m, m.refreshStatus
			
		case "h", "left":
			if m.currentPanel == StagedPanel {
				m.currentPanel = UnstagedPanel
				m.selectedIndex = 0
			}
			
		case "l", "right":
			if m.currentPanel == UnstagedPanel {
				m.currentPanel = StagedPanel  
				m.selectedIndex = 0
			}
			
		case "j", "down":
			m.moveDown()
			
		case "k", "up":
			m.moveUp()
			
		case "g":
			m.selectedIndex = 0
			
		case "G":
			if m.currentPanel == UnstagedPanel && m.repoStatus != nil {
				m.selectedIndex = len(m.repoStatus.UnstagedFiles) - 1
			} else if m.currentPanel == StagedPanel && m.repoStatus != nil {
				m.selectedIndex = len(m.repoStatus.StagedFiles) - 1
			}
			
		case "s", " ":
			return m, m.stageFile
			
		case "u":
			return m, m.unstageFile
			
		case "d":
			return m, m.discardChanges
			
		case "a":
			return m, m.stageAllFiles
			
		case "c":
			return m, m.commitPrompt
			
		case "enter":
			return m, m.showFileDiff
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
	}
	
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
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
	
	if m.showDiff {
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

func (m StatusModel) renderDiffView() string {
	return m.viewport.View()
}

func (m StatusModel) renderHelp() string {
	if m.showDiff {
		return m.helpStyle.Render("esc: back | q: quit")
	}
	
	help := "j/k: nav | h/l: panels | s: stage | u: unstage | d: discard | a: stage all | c: commit | enter: diff | r: refresh | q: quit"
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

func (m StatusModel) moveDown() {
	count := m.getCurrentFileCount()
	if count > 0 {
		m.selectedIndex = (m.selectedIndex + 1) % count
	}
}

func (m StatusModel) moveUp() {
	count := m.getCurrentFileCount()
	if count > 0 {
		m.selectedIndex = (m.selectedIndex - 1 + count) % count
	}
}

func (m *StatusModel) showMessage(msg string) {
	m.message = msg
	m.messageTime = time.Now()
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
	err := m.repo.UnstageFile(file.Path)
	if err != nil {
		// Handle error
		return refreshMsg{}
	}
	
	return refreshMsg{}
}

func (m StatusModel) discardChanges() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != UnstagedPanel || 
	   m.selectedIndex >= len(m.repoStatus.UnstagedFiles) {
		return refreshMsg{}
	}
	
	file := m.repoStatus.UnstagedFiles[m.selectedIndex]
	err := m.repo.DiscardChanges(file.Path)
	if err != nil {
		// Handle error
		return refreshMsg{}
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

func (m StatusModel) commitPrompt() tea.Msg {
	// Check if there are staged files
	if m.repoStatus == nil || len(m.repoStatus.StagedFiles) == 0 {
		return refreshMsg{}
	}
	
	// Launch commit input in a goroutine
	go func() {
		err := StartCommitInput(m.repo)
		if err == nil {
			// Commit successful, could send a message or refresh
		}
	}()
	
	return refreshMsg{}
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