package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hibobio/wtman/core"
)

var (
	colorTitle    lipgloss.Color
	colorSuccess  lipgloss.Color
	colorMuted    lipgloss.Color
	colorSelBg    lipgloss.Color
	colorAccent   lipgloss.Color
	colorError    lipgloss.Color

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
	colorTitle = lipgloss.Color(c.Title)
	colorSuccess = lipgloss.Color(c.Success)
	colorMuted = lipgloss.Color(c.Muted)
	colorSelBg = lipgloss.Color(c.SelectedBg)
	colorAccent = lipgloss.Color(c.Accent)
	colorError = lipgloss.Color(c.Error)

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
	styleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(c.Separator))
	styleSelectedRow = lipgloss.NewStyle().Background(colorSelBg).Foreground(lipgloss.Color(c.SelectedFg))
	styleNormalRow = lipgloss.NewStyle()
	styleCheckOn = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	styleCheckOff = lipgloss.NewStyle().Foreground(colorMuted)
	styleHint = lipgloss.NewStyle().Foreground(colorMuted)
	styleFilter = lipgloss.NewStyle().Foreground(colorAccent)
	styleError = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	styleSpinner = lipgloss.NewStyle().Foreground(colorTitle)
	styleAutocomplete = lipgloss.NewStyle().Foreground(colorMuted)
}
