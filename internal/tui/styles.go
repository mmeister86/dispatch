package tui

import "github.com/charmbracelet/lipgloss"

var palette = struct {
	bg       lipgloss.Color
	surface  lipgloss.Color
	line     lipgloss.Color
	text     lipgloss.Color
	dim      lipgloss.Color
	green    lipgloss.Color
	blue     lipgloss.Color
	amber    lipgloss.Color
	purple   lipgloss.Color
	warning  lipgloss.Color
	errorRed lipgloss.Color
}{
	bg:       lipgloss.Color("#070b09"),
	surface:  lipgloss.Color("#0a120a"),
	line:     lipgloss.Color("#142014"),
	text:     lipgloss.Color("#b0ccb0"),
	dim:      lipgloss.Color("#3a5a3a"),
	green:    lipgloss.Color("#68c868"),
	blue:     lipgloss.Color("#5899dd"),
	amber:    lipgloss.Color("#c49020"),
	purple:   lipgloss.Color("#9878c0"),
	warning:  lipgloss.Color("#c49020"),
	errorRed: lipgloss.Color("#cc5f5f"),
}

type styles struct {
	app       lipgloss.Style
	header    lipgloss.Style
	appName   lipgloss.Style
	status    lipgloss.Style
	body      lipgloss.Style
	footer    lipgloss.Style
	prompt    lipgloss.Style
	shortcuts lipgloss.Style
	label     lipgloss.Style
	user      lipgloss.Style
	agent     lipgloss.Style
	system    lipgloss.Style
	tool      lipgloss.Style
	accent    lipgloss.Style
}

func newStyles() styles {
	return styles{
		app: lipgloss.NewStyle().
			Background(palette.bg).
			Foreground(palette.text),
		header: lipgloss.NewStyle().
			Background(lipgloss.Color("#0c150c")).
			Foreground(palette.dim).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(palette.line).
			Padding(0, 2),
		appName: lipgloss.NewStyle().
			Foreground(palette.green).
			Bold(true),
		status: lipgloss.NewStyle().
			Foreground(palette.dim),
		body: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(palette.text).
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(palette.line).
			Padding(1, 2),
		footer: lipgloss.NewStyle().
			Background(lipgloss.Color("#0c150c")).
			Foreground(palette.dim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(palette.line).
			Padding(0, 2),
		prompt: lipgloss.NewStyle().
			Foreground(palette.green),
		shortcuts: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1e3a1e")),
		label: lipgloss.NewStyle().
			Faint(true),
		user: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#80b0bc")).
			PaddingLeft(2).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#1a3040")),
		agent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a8cca8")).
			PaddingLeft(2).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#1a3a1a")),
		system: lipgloss.NewStyle().
			Foreground(palette.purple).
			PaddingLeft(2).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#2a1840")),
		tool: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#b89040")).
			PaddingLeft(2).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#362a0c")),
		accent: lipgloss.NewStyle().
			Foreground(palette.amber),
	}
}
