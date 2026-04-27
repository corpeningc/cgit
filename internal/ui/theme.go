package ui

import "github.com/charmbracelet/lipgloss"

// Color palette shared across the UI package. New TUIs should pull
// from these names rather than reaching for raw color codes.
var (
	colorPink     = lipgloss.Color("205")
	colorPeach    = lipgloss.Color("#F1D3AB")
	colorGreen    = lipgloss.Color("46")
	colorRed      = lipgloss.Color("196")
	colorCyan     = lipgloss.Color("39")
	colorOrange   = lipgloss.Color("214")
	colorGray     = lipgloss.Color("245")
	colorDarkGray = lipgloss.Color("240")
)

// Common reusable styles. Two "title" variants exist because the
// codebase historically used pink in some TUIs and peach in others;
// unifying the palette is a future visual decision, not a dedup.
var (
	TitlePinkStyle  = lipgloss.NewStyle().Foreground(colorPink).Bold(true)
	TitlePeachStyle = lipgloss.NewStyle().Foreground(colorPeach).Bold(true)

	SelectedPinkStyle  = lipgloss.NewStyle().Foreground(colorPink).Bold(true)
	SelectedPeachStyle = lipgloss.NewStyle().Foreground(colorPeach).Bold(true)

	UnselectedStyle     = lipgloss.NewStyle().Foreground(colorGray)
	UnselectedBoldStyle = lipgloss.NewStyle().Foreground(colorGray).Bold(true)
	HelpStyle           = lipgloss.NewStyle().Foreground(colorGray)
	DimStyle            = lipgloss.NewStyle().Foreground(colorDarkGray)
	SeparatorStyle      = lipgloss.NewStyle().Foreground(colorDarkGray)

	SuccessStyle = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	SearchStyle  = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)

	StagedStyle   = lipgloss.NewStyle().Foreground(colorGreen)
	UnstagedStyle = lipgloss.NewStyle().Foreground(colorOrange)
)
