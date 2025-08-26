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
	BranchesPanel
	StashesPanel
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
	isLoading    bool
	loadingMsg   string
	showSearch      bool
	searchInput     textinput.Model
	searchQuery     string
	filteredIndices []int
	searchSelected  int
	
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
type loadingMsg string
type fileStatusUpdateMsg struct {
	filePath string
	staged   bool
}

func NewStatusModel(repo *git.GitRepo) StatusModel {
	vp := viewport.New(0, 0)
	
	ci := textinput.New()
	ci.Placeholder = "Enter commit message..."
	ci.CharLimit = 500
	ci.Width = 50
	
	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 100
	si.Width = 30
	
	return StatusModel{
		repo:         repo,
		viewport:     vp,
		commitInput:  ci,
		searchInput:  si,
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
		m.viewport.Height = msg.Height - 6  // Leave space for header, help, and padding
		
	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}
		
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		
		case "esc":
			if m.showSearch {
				m.showSearch = false
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.filteredIndices = nil
				m.searchSelected = 0
				return m, nil
			} else if m.showCommit {
				m.showCommit = false
				m.commitInput.SetValue("")
				return m, nil
			} else if m.showDiff {
				m.showDiff = false
				m.diffContent = ""
				return m, nil
			}
			
		case "r":
			if !m.showCommit && !m.showSearch {
				return m, tea.Batch(
					func() tea.Msg { return loadingMsg("Refreshing status...") },
					m.refreshStatus,
				)
			}
		
		case "enter":
			if m.showSearch {
				// Jump to selected search result and exit search
				if len(m.filteredIndices) > 0 && m.searchSelected < len(m.filteredIndices) {
					m.selectedIndex = m.filteredIndices[m.searchSelected]
				}
				m.showSearch = false
				m.searchInput.SetValue("")
				return m, nil
			} else if m.showCommit {
				// Commit with the entered message
				message := m.commitInput.Value()
				if message != "" {
					return m, m.performCommit(message)
				}
			} else if m.currentPanel == BranchesPanel && m.selectedIndex < len(m.repoStatus.Branches) {
				// Switch to selected branch
				branch := m.repoStatus.Branches[m.selectedIndex]
				if !branch.IsCurrent && !branch.IsRemote {
					return m, m.switchBranch(branch.Name)
				}
			} else {
				return m, m.showFileDiff
			}
			
		case "h", "left":
			if !m.showCommit && !m.showSearch {
				switch m.currentPanel {
				case StagedPanel:
					m.currentPanel = UnstagedPanel
				case BranchesPanel:
					m.currentPanel = StagedPanel
				case StashesPanel:
					m.currentPanel = BranchesPanel
				}
				m.selectedIndex = 0
			}
			
		case "l", "right":
			if !m.showCommit && !m.showSearch {
				switch m.currentPanel {
				case UnstagedPanel:
					m.currentPanel = StagedPanel
				case StagedPanel:
					m.currentPanel = BranchesPanel
				case BranchesPanel:
					m.currentPanel = StashesPanel
				}
				m.selectedIndex = 0
			}
			
		case "j", "down":
			if m.showDiff {
				m.viewport.LineDown(1)
			} else if m.showSearch {
				// Navigate down in search results
				if len(m.filteredIndices) > 0 {
					m.searchSelected = (m.searchSelected + 1) % len(m.filteredIndices)
				}
			} else if !m.showCommit {
				m.moveDown()
			}
			
		case "k", "up":
			if m.showDiff {
				m.viewport.LineUp(1)
			} else if m.showSearch {
				// Navigate up in search results
				if len(m.filteredIndices) > 0 {
					m.searchSelected = (m.searchSelected - 1 + len(m.filteredIndices)) % len(m.filteredIndices)
				}
			} else if !m.showCommit {
				m.moveUp()
			}
			
		case "g":
			if m.showDiff {
				m.viewport.GotoTop()
			} else if !m.showCommit && !m.showSearch {
				m.selectedIndex = 0
			}
			
		case "G":
			if m.showDiff {
				m.viewport.GotoBottom()
			} else if !m.showCommit && !m.showSearch {
				count := m.getCurrentFileCount()
				if count > 0 {
					m.selectedIndex = count - 1
				}
			}
			
		case "s", " ":
			if !m.showCommit && !m.showSearch {
				return m, m.stageFile
			}
			
		case "+":
			if !m.showCommit {
				if m.showSearch {
					return m, m.stageFileFromSearch
				} else {
					return m, m.stageFile
				}
			}
			
		case "-":
			if !m.showCommit {
				if m.showSearch {
					return m, m.unstageFileFromSearch
				} else {
					return m, m.unstageFile
				}
			}
			
		case "u":
			if !m.showCommit && !m.showSearch {
				return m, m.unstageFile
			}
			
		case "d":
			if !m.showCommit && !m.showSearch {
				if m.currentPanel == StashesPanel {
					return m, m.deleteStash
				} else {
					return m, m.discardChanges
				}
			}
			
		case "a":
			if !m.showCommit && !m.showSearch {
				return m, m.stageAllFiles
			}
			
		case "c":
			if !m.showCommit && !m.showSearch {
				// Check if there are staged files
				if m.repoStatus != nil && len(m.repoStatus.StagedFiles) > 0 {
					m.showCommit = true
					m.commitInput.Focus()
					m.commitInput.SetValue("")
					return m, nil
				}
			}
		
		case "p":
			if !m.showCommit && !m.showSearch {
				return m, m.pushChanges()
			}
			
		case "/":
			if !m.showCommit && !m.showDiff {
				m.showSearch = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				return m, nil
			}
		
		case "ctrl+d", "pgdn":
			if m.showDiff {
				m.viewport.HalfViewDown()
			}
			
		case "ctrl+u", "pgup":
			if m.showDiff {
				m.viewport.HalfViewUp()
			}
		}
		
	case statusMsg:
		m.repoStatus = msg
		m.isLoading = false
		if m.selectedIndex >= m.getCurrentFileCount() {
			m.selectedIndex = max(0, m.getCurrentFileCount()-1)
		}
		
	case diffMsg:
		m.diffContent = string(msg)
		m.showDiff = true
		// Ensure viewport is properly sized before setting content
		if m.width > 0 && m.height > 0 {
			m.viewport.Width = m.width - 4
			m.viewport.Height = m.height - 6
		}
		// Temporarily disable highlighting to debug the display issue
		// highlightedContent := m.highlightDiff(m.diffContent)
		// m.viewport.SetContent(highlightedContent)
		m.viewport.SetContent(m.diffContent)
		// Reset viewport position to top
		m.viewport.GotoTop()
		
	case refreshMsg:
		m.isLoading = true
		m.loadingMsg = "Refreshing status..."
		return m, m.refreshStatus
		
	case loadingMsg:
		m.isLoading = true
		m.loadingMsg = string(msg)
		return m, nil
	
	case fileStatusUpdateMsg:
		m.handleFileStatusUpdate(msg)
		return m, nil
	
	case error:
		m.showMessage(msg.Error())
		if m.showCommit {
			// Stay in commit mode on error
		}
		return m, m.refreshStatus
		
	case string:
		switch msg {
			case "commit_success":
				m.showCommit = false
				m.commitInput.SetValue("")
				m.commitInput.Blur()
				m.showMessage("Commit successful!")
				return m, m.refreshStatus
			case "push_success":
				m.showMessage("Push successful!")
				return m, m.refreshStatus
		}
	}
	
	// Update text input if in commit mode
	if m.showCommit {
		m.commitInput, cmd = m.commitInput.Update(msg)
		return m, cmd
	}
	
	// Update text input if in search mode
	if m.showSearch {
		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Perform real-time search if input changed
		if m.searchInput.Value() != oldValue {
			m.searchQuery = m.searchInput.Value()
			m.performSearch()
		}
		return m, cmd
	}
	
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m StatusModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	
	if m.repoStatus == nil || m.isLoading {
		if m.loadingMsg != "" {
			return m.loadingMsg + " ⏳"
		}
		return "Loading repository status... ⏳"
	}
	
	var sections []string
	
	// Header
	header := m.renderHeader()
	sections = append(sections, header)
	
	if m.showCommit {
		// Show commit input view
		sections = append(sections, m.renderCommitView())
	} else if m.showSearch {
		// Show search input view
		sections = append(sections, m.renderSearchView())
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
	branchesPanel := m.renderBranchesPanel()
	stashesPanel := m.renderStashesPanel()
	
	// Top row: unstaged and staged files
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, unstagedPanel, " ", stagedPanel)
	
	// Bottom row: branches and stashes
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, branchesPanel, " ", stashesPanel)
	
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
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
			} else if m.searchQuery != "" && m.containsIndex(m.filteredIndices, i) {
				style = m.diffAddedStyle  // Highlight search matches in green
			}
			
			statusChar := m.getStatusChar(file.Status)
			line := fmt.Sprintf("%s%s [%s]", prefix, file.Path, statusChar)
			content.WriteString(style.Render(line) + "\n")
		}
	}
	
	panelWidth := (m.width - 3) / 2
	panelHeight := (m.height - 15) / 2
	return m.panelStyle.Width(panelWidth).Height(panelHeight).Render(content.String())
}

func (m StatusModel) renderBranchesPanel() string {
	title := "Branches"
	if len(m.repoStatus.Branches) > 0 {
		localCount := 0
		remoteCount := 0
		for _, branch := range m.repoStatus.Branches {
			if branch.IsRemote {
				remoteCount++
			} else {
				localCount++
			}
		}
		title += fmt.Sprintf(" (L:%d R:%d)", localCount, remoteCount)
	}
	
	var content strings.Builder
	content.WriteString(m.titleStyle.Render(title) + "\n")
	
	if len(m.repoStatus.Branches) == 0 {
		content.WriteString(m.unselectedStyle.Render("  (no branches)"))
	} else {
		for i, branch := range m.repoStatus.Branches {
			if i >= 10 {
				content.WriteString(m.unselectedStyle.Render("  ..."))
				break
			}
			
			prefix := "  "
			style := m.unselectedStyle
			
			if m.currentPanel == BranchesPanel && i == m.selectedIndex {
				prefix = "> "
				style = m.selectedStyle
			} else if m.searchQuery != "" && m.containsIndex(m.filteredIndices, i) {
				style = m.diffAddedStyle  // Highlight search matches in green
			}
			
			branchType := ""
			if branch.IsRemote {
				branchType = "R"
			} else {
				branchType = "L"
			}
			
			if branch.IsCurrent {
				branchType = "*"
			}
			
			line := fmt.Sprintf("%s%s [%s]", prefix, branch.Name, branchType)
			content.WriteString(style.Render(line) + "\n")
		}
	}
	
	panelWidth := (m.width - 3) / 2
	panelHeight := (m.height - 15) / 2
	return m.panelStyle.Width(panelWidth).Height(panelHeight).Render(content.String())
}

func (m StatusModel) renderStashesPanel() string {
	title := "Stashes"
	if len(m.repoStatus.Stashes) > 0 {
		title += fmt.Sprintf(" (%d)", len(m.repoStatus.Stashes))
	}
	
	var content strings.Builder
	content.WriteString(m.titleStyle.Render(title) + "\n")
	
	if len(m.repoStatus.Stashes) == 0 {
		content.WriteString(m.unselectedStyle.Render("  (no stashes)"))
	} else {
		for i, stash := range m.repoStatus.Stashes {
			if i >= 8 {
				content.WriteString(m.unselectedStyle.Render("  ..."))
				break
			}
			
			prefix := "  "
			style := m.unselectedStyle
			
			if m.currentPanel == StashesPanel && i == m.selectedIndex {
				prefix = "> "
				style = m.selectedStyle
			} else if m.searchQuery != "" && m.containsIndex(m.filteredIndices, i) {
				style = m.diffAddedStyle  // Highlight search matches in green
			}
			
			message := stash.Message
			if len(message) > 25 {
				message = message[:22] + "..."
			}
			
			line := fmt.Sprintf("%s%s (%s)", prefix, message, stash.Date)
			content.WriteString(style.Render(line) + "\n")
		}
	}
	
	panelWidth := (m.width - 3) / 2
	panelHeight := (m.height - 15) / 2
	return m.panelStyle.Width(panelWidth).Height(panelHeight).Render(content.String())
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
			} else if m.searchQuery != "" && m.containsIndex(m.filteredIndices, i) {
				style = m.diffAddedStyle  // Highlight search matches in green
			}
			
			statusChar := m.getStatusChar(file.Status)
			line := fmt.Sprintf("%s%s [%s]", prefix, file.Path, statusChar)
			content.WriteString(style.Render(line) + "\n")
		}
	}
	
	panelWidth := (m.width - 3) / 2
	panelHeight := (m.height - 15) / 2
	return m.panelStyle.Width(panelWidth).Height(panelHeight).Render(content.String())
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

func (m StatusModel) renderSearchView() string {
	var content strings.Builder
	
	// Title
	panelName := map[PanelType]string{
		UnstagedPanel: "Unstaged Files",
		StagedPanel:   "Staged Files", 
		BranchesPanel: "Branches",
		StashesPanel:  "Stashes",
	}[m.currentPanel]
	
	title := m.titleStyle.Render(fmt.Sprintf("Search %s", panelName))
	content.WriteString(title + "\n\n")
	
	// Search input
	content.WriteString(m.headerStyle.Render("Search:") + "\n")
	content.WriteString(m.searchInput.View() + "\n\n")
	
	// Show search results
	if m.searchQuery != "" {
		if len(m.filteredIndices) == 0 {
			content.WriteString(m.unselectedStyle.Render("No matches found") + "\n")
		} else {
			content.WriteString(m.headerStyle.Render(fmt.Sprintf("Results (%d matches):", len(m.filteredIndices))) + "\n")
			
			// Show filtered items with navigation
			for i, idx := range m.filteredIndices {
				prefix := "  "
				style := m.unselectedStyle
				
				if i == m.searchSelected {
					prefix = "> "
					style = m.selectedStyle
				}
				
				var itemText string
				switch m.currentPanel {
				case UnstagedPanel:
					if idx < len(m.repoStatus.UnstagedFiles) {
						file := m.repoStatus.UnstagedFiles[idx]
						itemText = fmt.Sprintf("%s [%s]", file.Path, m.getStatusChar(file.Status))
					}
				case StagedPanel:
					if idx < len(m.repoStatus.StagedFiles) {
						file := m.repoStatus.StagedFiles[idx]
						itemText = fmt.Sprintf("%s [%s]", file.Path, m.getStatusChar(file.Status))
					}
				case BranchesPanel:
					if idx < len(m.repoStatus.Branches) {
						branch := m.repoStatus.Branches[idx]
						branchType := map[bool]string{true: "R", false: "L"}[branch.IsRemote]
						if branch.IsCurrent {
							branchType = "*"
						}
						itemText = fmt.Sprintf("%s [%s]", branch.Name, branchType)
					}
				case StashesPanel:
					if idx < len(m.repoStatus.Stashes) {
						stash := m.repoStatus.Stashes[idx]
						message := stash.Message
						if len(message) > 30 {
							message = message[:27] + "..."
						}
						itemText = fmt.Sprintf("%s (%s)", message, stash.Date)
					}
				}
				
				line := prefix + itemText
				content.WriteString(style.Render(line) + "\n")
			}
		}
	} else {
		content.WriteString(m.unselectedStyle.Render("Type to search...") + "\n")
	}
	
	return content.String()
}

func (m StatusModel) renderDiffView() string {
	// Just return the viewport view - content is set when diffMsg is received
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
	if m.showSearch {
		return m.helpStyle.Render("j/k: navigate results | +: stage | -: unstage | enter: select | esc: cancel | q: quit")
	} else if m.showCommit {
		return m.helpStyle.Render("enter: commit | esc: cancel | q: quit")
	} else if m.showDiff {
		return m.helpStyle.Render("j/k: scroll | g/G: top/bottom | ctrl+d/u: page | esc: back | q: quit")
	}
	
	help := "h/l: panels | j/k: navigate | /: search | s/+: stage | u/-: unstage | d: discard/delete | c: commit | p: push | enter: diff/switch | r: refresh | q: quit"
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
	
	switch m.currentPanel {
	case UnstagedPanel:
		return len(m.repoStatus.UnstagedFiles)
	case StagedPanel:
		return len(m.repoStatus.StagedFiles)
	case BranchesPanel:
		return len(m.repoStatus.Branches)
	case StashesPanel:
		return len(m.repoStatus.Stashes)
	default:
		return 0
	}
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
		return nil
	}
	
	file := m.repoStatus.UnstagedFiles[m.selectedIndex]
	err := m.repo.StageFile(file.Path)
	if err != nil {
		return fmt.Errorf("failed to stage file: %v", err)
	}
	
	// Update status locally without full refresh
	return m.updateFileStatus(file, true)
}

func (m StatusModel) unstageFile() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != StagedPanel || 
		m.selectedIndex >= len(m.repoStatus.StagedFiles) {
		return nil
	}
	
	file := m.repoStatus.StagedFiles[m.selectedIndex]
	err := m.repo.UnstageFile(file.Path, file.Status)
	if err != nil {
		return fmt.Errorf("failed to unstage file: %v", err)
	}
	
	// Update status locally without full refresh
	return m.updateFileStatus(file, false)
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
	
	switch m.currentPanel {
	case UnstagedPanel:
		if m.selectedIndex < len(m.repoStatus.UnstagedFiles) {
			filePath := m.repoStatus.UnstagedFiles[m.selectedIndex].Path
			diff, err := m.repo.GetFileDiff(filePath, false)
			if err != nil {
				return diffMsg("Error getting diff: " + err.Error())
			}
			return diffMsg(diff)
		}
		
	case StagedPanel:
		if m.selectedIndex < len(m.repoStatus.StagedFiles) {
			filePath := m.repoStatus.StagedFiles[m.selectedIndex].Path
			diff, err := m.repo.GetFileDiff(filePath, true)
			if err != nil {
				return diffMsg("Error getting diff: " + err.Error())
			}
			return diffMsg(diff)
		}
		
	case BranchesPanel:
		if m.selectedIndex < len(m.repoStatus.Branches) {
			branch := m.repoStatus.Branches[m.selectedIndex]
			info := fmt.Sprintf("Branch: %s\n", branch.Name)
			info += fmt.Sprintf("Type: %s\n", map[bool]string{true: "Remote", false: "Local"}[branch.IsRemote])
			info += fmt.Sprintf("Current: %s\n", map[bool]string{true: "Yes", false: "No"}[branch.IsCurrent])
			if branch.Tracking != "" {
				info += fmt.Sprintf("Tracking: %s\n", branch.Tracking)
			}
			return diffMsg(info)
		}
		
	case StashesPanel:
		if m.selectedIndex < len(m.repoStatus.Stashes) {
			stash := m.repoStatus.Stashes[m.selectedIndex]
			info := fmt.Sprintf("Stash: %s\n", stash.Message)
			info += fmt.Sprintf("Branch: %s\n", stash.Branch)
			info += fmt.Sprintf("Date: %s\n", stash.Date)
			info += fmt.Sprintf("Index: %d\n", stash.Index)
			return diffMsg(info)
		}
	}
	
	return refreshMsg{}
}

func (m *StatusModel) performSearch() {
	if m.searchQuery == "" {
		m.filteredIndices = nil
		m.searchSelected = 0
		return
	}
	
	query := strings.ToLower(m.searchQuery)
	m.filteredIndices = []int{}
	
	switch m.currentPanel {
	case UnstagedPanel:
		for i, file := range m.repoStatus.UnstagedFiles {
			if m.fuzzyMatch(strings.ToLower(file.Path), query) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	case StagedPanel:
		for i, file := range m.repoStatus.StagedFiles {
			if m.fuzzyMatch(strings.ToLower(file.Path), query) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	case BranchesPanel:
		for i, branch := range m.repoStatus.Branches {
			if m.fuzzyMatch(strings.ToLower(branch.Name), query) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	case StashesPanel:
		for i, stash := range m.repoStatus.Stashes {
			if m.fuzzyMatch(strings.ToLower(stash.Message), query) || 
				m.fuzzyMatch(strings.ToLower(stash.Branch), query) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	}
	
	// Reset search selection to first result
	m.searchSelected = 0
}

func (m StatusModel) fuzzyMatch(text, query string) bool {
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

func (m StatusModel) switchBranch(branchName string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.SwitchBranch(branchName)
		if err != nil {
			return fmt.Errorf("failed to switch branch: %v", err)
		}
		return refreshMsg{}
	}
}

func (m StatusModel) containsIndex(indices []int, target int) bool {
	for _, idx := range indices {
		if idx == target {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// updateFileStatus moves a file between staged/unstaged without full refresh
func (m StatusModel) updateFileStatus(file git.FileStatus, toStaged bool) tea.Msg {
	return fileStatusUpdateMsg{
		filePath: file.Path,
		staged:   toStaged,
	}
}

// handleFileStatusUpdate moves a file between staged/unstaged lists locally
func (m *StatusModel) handleFileStatusUpdate(msg fileStatusUpdateMsg) {
	if m.repoStatus == nil {
		return
	}

	if msg.staged {
		// Move from unstaged to staged
		for i, file := range m.repoStatus.UnstagedFiles {
			if file.Path == msg.filePath {
				// Remove from unstaged
				m.repoStatus.UnstagedFiles = append(m.repoStatus.UnstagedFiles[:i], m.repoStatus.UnstagedFiles[i+1:]...)
				// Add to staged
				file.Staged = true
				file.WorkTree = false
				m.repoStatus.StagedFiles = append(m.repoStatus.StagedFiles, file)
				// Adjust selection
				if m.selectedIndex >= len(m.repoStatus.UnstagedFiles) {
					m.selectedIndex = max(0, len(m.repoStatus.UnstagedFiles)-1)
				}
				break
			}
		}
	} else {
		// Move from staged to unstaged
		for i, file := range m.repoStatus.StagedFiles {
			if file.Path == msg.filePath {
				// Remove from staged
				m.repoStatus.StagedFiles = append(m.repoStatus.StagedFiles[:i], m.repoStatus.StagedFiles[i+1:]...)
				// Add to unstaged
				file.Staged = false
				file.WorkTree = true
				m.repoStatus.UnstagedFiles = append(m.repoStatus.UnstagedFiles, file)
				// Adjust selection
				if m.selectedIndex >= len(m.repoStatus.StagedFiles) {
					m.selectedIndex = max(0, len(m.repoStatus.StagedFiles)-1)
				}
				break
			}
		}
	}
}

// stageFileFromSearch stages a file from search results
func (m StatusModel) stageFileFromSearch() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != UnstagedPanel || 
		len(m.filteredIndices) == 0 || m.searchSelected >= len(m.filteredIndices) {
		return nil
	}
	
	actualIndex := m.filteredIndices[m.searchSelected]
	if actualIndex >= len(m.repoStatus.UnstagedFiles) {
		return nil
	}
	
	file := m.repoStatus.UnstagedFiles[actualIndex]
	err := m.repo.StageFile(file.Path)
	if err != nil {
		return fmt.Errorf("failed to stage file: %v", err)
	}
	
	return m.updateFileStatus(file, true)
}

// unstageFileFromSearch unstages a file from search results
func (m StatusModel) unstageFileFromSearch() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != StagedPanel || 
		len(m.filteredIndices) == 0 || m.searchSelected >= len(m.filteredIndices) {
		return nil
	}
	
	actualIndex := m.filteredIndices[m.searchSelected]
	if actualIndex >= len(m.repoStatus.StagedFiles) {
		return nil
	}
	
	file := m.repoStatus.StagedFiles[actualIndex]
	err := m.repo.UnstageFile(file.Path, file.Status)
	if err != nil {
		return fmt.Errorf("failed to unstage file: %v", err)
	}
	
	return m.updateFileStatus(file, false)
}

// deleteStash deletes the selected stash
func (m StatusModel) deleteStash() tea.Msg {
	if m.repoStatus == nil || m.currentPanel != StashesPanel || 
		m.selectedIndex >= len(m.repoStatus.Stashes) {
		return nil
	}
	
	stash := m.repoStatus.Stashes[m.selectedIndex]
	err := m.repo.DeleteStash(stash.Index)
	if err != nil {
		return fmt.Errorf("failed to delete stash: %v", err)
	}
	
	return refreshMsg{}
}

func StartStatusTUI(repo *git.GitRepo) error {
	m := NewStatusModel(repo)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}