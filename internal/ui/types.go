package ui

import (
	"github.com/corpeningc/cgit/internal/git"
)

type GitOperationCompleteMsg struct {
	success       bool
	error         error
	operation     string
	filesAffected []string
}

type StatusRefreshMsg struct {
	stagedFiles   []git.FileStatus
	unstagedFiles []git.FileStatus
	error         error
}

type ClearStatusMsg struct{}

type Mode int

const (
	NormalMode Mode = iota
	SearchMode
	SearchResultsMode
	DiffMode
)
