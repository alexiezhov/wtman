package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hibobio/wtman/core"
)

var (
	colorPrimary   lipgloss.Color
	colorGreen     lipgloss.Color
	colorDimmed    lipgloss.Color
	colorHighlight lipgloss.Color
	colorCyan      lipgloss.Color
	colorRed       lipgloss.Color

	styleTitle        lipgloss.Style
	styleHeader       lipgloss.Style
	styleSeparator    lipgloss.Style
	styleSelectedRow  lipgloss.Style
	styleNormalRow    lipgloss.Style
	styleCheckOn      lipgloss.Style
	styleCheckOff     lipgloss.Style
	styleHint         lipgloss.Style
	styleFilter       lipgloss.Style
	styleError        lipgloss.Style
	styleSpinner      lipgloss.Style
	styleAutocomplete lipgloss.Style
)

func init() {
	ApplyColors(core.DefaultColors())
}

func ApplyColors(c core.ColorsConfig) {
	colorPrimary = lipgloss.Color(c.Primary)
	colorGreen = lipgloss.Color(c.Green)
	colorDimmed = lipgloss.Color(c.Dimmed)
	colorHighlight = lipgloss.Color(c.Highlight)
	colorCyan = lipgloss.Color(c.Cyan)
	colorRed = lipgloss.Color(c.Red)

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(colorDimmed)
	styleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(c.Separator))
	styleSelectedRow = lipgloss.NewStyle().Background(colorHighlight).Foreground(lipgloss.Color(c.SelectedFg))
	styleNormalRow = lipgloss.NewStyle()
	styleCheckOn = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	styleCheckOff = lipgloss.NewStyle().Foreground(colorDimmed)
	styleHint = lipgloss.NewStyle().Foreground(colorDimmed)
	styleFilter = lipgloss.NewStyle().Foreground(colorCyan)
	styleError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	styleSpinner = lipgloss.NewStyle().Foreground(colorPrimary)
	styleAutocomplete = lipgloss.NewStyle().Foreground(colorDimmed)
}
