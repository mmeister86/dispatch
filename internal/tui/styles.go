package tui

import "github.com/charmbracelet/lipgloss"

var palette = struct {
	bg       lipgloss.Color
	surface  lipgloss.Color
	line     lipgloss.Color
	text     lipgloss.Color
	dim      lipgloss.Color
	muted    lipgloss.Color
	green    lipgloss.Color
	teal     lipgloss.Color
	amber    lipgloss.Color
	purple   lipgloss.Color
	warning  lipgloss.Color
	errorRed lipgloss.Color
}{
	bg:       lipgloss.Color("#070b09"),
	surface:  lipgloss.Color("#0c120f"),
	line:     lipgloss.Color("#182620"),
	text:     lipgloss.Color("#c4dcc0"),
	dim:      lipgloss.Color("#5a7a62"),
	muted:    lipgloss.Color("#3a5242"),
	green:    lipgloss.Color("#7ed690"),
	teal:     lipgloss.Color("#6bb8c8"),
	amber:    lipgloss.Color("#c4a860"),
	purple:   lipgloss.Color("#9a8ec0"),
	warning:  lipgloss.Color("#c4a040"),
	errorRed: lipgloss.Color("#c86060"),
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
	divider   lipgloss.Style
}

func newStyles() styles {
	return styles{
		app: lipgloss.NewStyle().
			Foreground(palette.text),
		header: lipgloss.NewStyle().
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
			Foreground(palette.text).
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(palette.line).
			Padding(1, 2),
		footer: lipgloss.NewStyle().
			Foreground(palette.dim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(palette.line).
			Padding(0, 2),
		prompt: lipgloss.NewStyle().
			Foreground(palette.green),
		shortcuts: lipgloss.NewStyle().
			Foreground(palette.muted),
		label: lipgloss.NewStyle().
			Foreground(palette.muted),
		user: lipgloss.NewStyle().
			Foreground(palette.teal).
			PaddingLeft(1),
		agent: lipgloss.NewStyle().
			Foreground(palette.text).
			PaddingLeft(1),
		system: lipgloss.NewStyle().
			Foreground(palette.purple).
			PaddingLeft(1).
			Faint(true),
		tool: lipgloss.NewStyle().
			Foreground(palette.amber).
			PaddingLeft(1).
			Faint(true),
		accent: lipgloss.NewStyle().
			Foreground(palette.amber),
		divider: lipgloss.NewStyle().
			Foreground(palette.line),
	}
}
