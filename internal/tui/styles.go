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
	surface:  lipgloss.Color("#070b09"),
	line:     lipgloss.Color("#203026"),
	text:     lipgloss.Color("#d6ead2"),
	dim:      lipgloss.Color("#78927c"),
	green:    lipgloss.Color("#98e69f"),
	blue:     lipgloss.Color("#83bed1"),
	amber:    lipgloss.Color("#e0b35a"),
	purple:   lipgloss.Color("#baa0df"),
	warning:  lipgloss.Color("#e0b35a"),
	errorRed: lipgloss.Color("#e27a7a"),
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
			Foreground(lipgloss.Color("#5f7a63")),
		label: lipgloss.NewStyle().
			Faint(true),
		user: lipgloss.NewStyle().
			Foreground(palette.blue).
			PaddingLeft(1),
		agent: lipgloss.NewStyle().
			Foreground(palette.text).
			PaddingLeft(1),
		system: lipgloss.NewStyle().
			Foreground(palette.purple).
			PaddingLeft(1),
		tool: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#b9954f")).
			PaddingLeft(1).
			Faint(true),
		accent: lipgloss.NewStyle().
			Foreground(palette.amber),
	}
}
