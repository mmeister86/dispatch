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
	surface:  lipgloss.Color("#09110d"),
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
	headerBG  lipgloss.Color
	appName   lipgloss.Style
	status    lipgloss.Style
	body      lipgloss.Style
	footer    lipgloss.Style
	footerBG  lipgloss.Color
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
	headerBG := lipgloss.Color("#0b140f")
	footerBG := lipgloss.Color("#0b140f")
	return styles{
		app: lipgloss.NewStyle().
			Background(palette.bg).
			Foreground(palette.text),
		header: lipgloss.NewStyle().
			Background(headerBG).
			Foreground(palette.dim).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(palette.line).
			Padding(0, 2),
		headerBG: headerBG,
		appName: lipgloss.NewStyle().
			Background(headerBG).
			Foreground(palette.green).
			Bold(true),
		status: lipgloss.NewStyle().
			Background(headerBG).
			Foreground(palette.dim),
		body: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(palette.text).
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(palette.line).
			Padding(1, 2),
		footer: lipgloss.NewStyle().
			Background(footerBG).
			Foreground(palette.dim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(palette.line).
			Padding(0, 2),
		footerBG: footerBG,
		prompt: lipgloss.NewStyle().
			Background(footerBG).
			Foreground(palette.green),
		shortcuts: lipgloss.NewStyle().
			Background(footerBG).
			Foreground(lipgloss.Color("#5f7a63")),
		label: lipgloss.NewStyle().
			Background(palette.surface).
			Faint(true),
		user: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(palette.blue).
			PaddingLeft(1),
		agent: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(palette.text).
			PaddingLeft(1),
		system: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(palette.purple).
			PaddingLeft(1),
		tool: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(lipgloss.Color("#b9954f")).
			PaddingLeft(1).
			Faint(true),
		accent: lipgloss.NewStyle().
			Background(palette.surface).
			Foreground(palette.amber),
	}
}
