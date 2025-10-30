package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corpeningc/cgit/internal/git"
)

type ConflictResolverModel struct {
	repo                 *git.GitRepo
	conflictFiles        []git.ConflictFile
	currentFileIndex     int
	currentConflictIndex int
	resolution           git.ResolutionChoice
	content              string
	viewport             viewport.Model
}
