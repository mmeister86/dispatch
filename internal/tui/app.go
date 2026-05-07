package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/agent"
	"github.com/matthias/dispatch/internal/provider"
)

type messageKind int

const (
	kindSystem messageKind = iota
	kindUser
	kindAgent
	kindTool
)

type chatMessage struct {
	Kind messageKind
	Body string
}

type agentChunkMsg provider.Chunk
type agentDoneMsg struct{}

type Model struct {
	cfg       config.Config
	agent     *agent.Agent
	styles    styles
	viewport  viewport.Model
	textarea  textarea.Model
	spinner   spinner.Model
	messages  []chatMessage
	width     int
	height    int
	ready     bool
	streaming bool
	lastInput string
	showHelp  bool
	stream    <-chan provider.Chunk
	activeMsg int
}

func New(cfg config.Config, chatAgent *agent.Agent, warnings []string) Model {
	ta := textarea.New()
	ta.Placeholder = "Nachricht an dispatch"
	ta.ShowLineNumbers = false
	ta.Prompt = "› "
	ta.CharLimit = 4000
	ta.SetHeight(1)
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(palette.green)

	m := Model{
		cfg:       cfg,
		agent:     chatAgent,
		styles:    newStyles(),
		textarea:  ta,
		spinner:   sp,
		activeMsg: -1,
		messages: []chatMessage{
			{
				Kind: kindSystem,
				Body: "Willkommen bei dispatch.\nKeys werden lokal gelesen. Starte `dispatch --setup`, falls Integrationen fehlen.",
			},
			{
				Kind: kindAgent,
				Body: "Bereit. Frag mich nach Repo-Aktivitaet, Recherchethemen oder geplanten Posts.",
			},
		},
	}
	for _, warning := range warnings {
		m.messages = append(m.messages, chatMessage{Kind: kindSystem, Body: warning})
	}
	m.refreshViewport()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit
		case "ctrl+c":
			if m.streaming {
				m.streaming = false
				m.addMessage(kindSystem, "Laufender Agent-Call abgebrochen.")
				return m, nil
			}
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			m.refreshViewport()
			return m, nil
		case "ctrl+l":
			m.messages = nil
			m.addMessage(kindSystem, "Chat-Verlauf geleert.")
			return m, nil
		case "ctrl+p":
			return m.startAgent("Liste meine geplanten Posts.")
		case "ctrl+r":
			return m.startAgent("Zeige mir GitHub-Aktivitaet aus allen Repos der letzten 7 Tage.")
		case "ctrl+s":
			return m.startAgent("Welche Themen sind aktuell relevant fuer Developer Social Posts? Recherchiere mit Quellen.")
		case "up":
			if m.textarea.Value() == "" && m.lastInput != "" {
				m.textarea.SetValue(m.lastInput)
				return m, nil
			}
		case "enter":
			value := strings.TrimSpace(m.textarea.Value())
			if value == "" {
				return m, nil
			}
			m.lastInput = value
			m.textarea.Reset()
			return m.startAgent(value)
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case agentChunkMsg:
		chunk := provider.Chunk(msg)
		if chunk.Err != nil {
			m.addMessage(kindSystem, fmt.Sprintf("Agent-Fehler: %v", chunk.Err))
			m.streaming = false
			m.stream = nil
			m.activeMsg = -1
			return m, nil
		}
		if chunk.Text != "" && m.activeMsg >= 0 {
			m.appendToMessage(m.activeMsg, chunk.Text)
		} else if chunk.Text != "" {
			m.activeMsg = m.addMessage(kindAgent, chunk.Text)
		}
		if chunk.ToolCall != nil {
			m.addMessage(kindTool, formatToolCall(*chunk.ToolCall))
		}
		if chunk.ToolResult != nil {
			m.addMessage(kindTool, formatToolResult(*chunk.ToolResult))
		}
		return m, waitForAgentChunk(m.stream)
	case agentDoneMsg:
		m.streaming = false
		m.stream = nil
		m.activeMsg = -1
		m.refreshViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) startAgent(value string) (tea.Model, tea.Cmd) {
	m.addMessage(kindUser, value)
	if m.agent == nil {
		m.addMessage(kindAgent, "LLM-Streaming und Tools sind bereit, sobald ein LLM API Key konfiguriert ist. Starte `dispatch --setup` oder setze LLM_API_KEY.")
		return m, nil
	}
	stream, err := m.agent.Run(context.Background(), value)
	if err != nil {
		m.addMessage(kindSystem, fmt.Sprintf("Agent-Start fehlgeschlagen: %v", err))
		return m, nil
	}
	m.streaming = true
	m.stream = stream
	m.activeMsg = -1
	return m, waitForAgentChunk(stream)
}

func (m Model) View() string {
	if !m.ready {
		return "initialisiere dispatch..."
	}

	header := m.renderHeader()
	body := m.styles.body.
		Width(max(0, m.width-2)).
		Height(max(1, m.viewport.Height+2)).
		Render(m.viewport.View())
	footer := m.renderFooter()

	return m.styles.app.Width(m.width).Render(lipgloss.JoinVertical(lipgloss.Left, header, body, footer))
}

func (m *Model) layout() {
	headerHeight := 1
	footerHeight := 4
	bodyHeight := max(1, m.height-headerHeight-footerHeight-2)
	bodyWidth := max(20, m.width-6)

	if !m.ready {
		m.viewport = viewport.New(bodyWidth, bodyHeight)
		m.ready = true
	} else {
		m.viewport.Width = bodyWidth
		m.viewport.Height = bodyHeight
	}

	m.textarea.SetWidth(max(20, m.width-6))
	m.refreshViewport()
}

func (m *Model) addMessage(kind messageKind, body string) int {
	m.messages = append(m.messages, chatMessage{Kind: kind, Body: body})
	m.refreshViewport()
	return len(m.messages) - 1
}

func (m *Model) appendToMessage(index int, text string) {
	if index < 0 || index >= len(m.messages) {
		return
	}
	m.messages[index].Body += text
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	if !m.ready {
		return
	}
	var blocks []string
	for _, msg := range m.messages {
		blocks = append(blocks, m.renderMessage(msg))
	}
	if m.showHelp {
		blocks = append(blocks, m.renderMessage(chatMessage{
			Kind: kindSystem,
			Body: "Shortcuts\nctrl+p posts  ·  ctrl+r repos  ·  ctrl+s suche  ·  ctrl+l leeren  ·  ctrl+c abbrechen  ·  ctrl+q beenden",
		}))
	}
	if m.streaming && m.activeMsg < 0 {
		blocks = append(blocks, m.renderMessage(chatMessage{
			Kind: kindAgent,
			Body: m.spinner.View(),
		}))
	}
	m.viewport.SetContent(strings.Join(blocks, "\n\n"))
	m.viewport.GotoBottom()
}

func (m Model) renderHeader() string {
	status := strings.Join([]string{
		statusDot(m.cfg.LLM.APIKey != "") + " " + m.cfg.LLM.Provider + "/" + m.cfg.LLM.Model,
		statusDot(m.cfg.Postiz.APIKey != "") + " postiz",
		statusDot(m.cfg.GitHub.Token != "") + " github",
		statusDot(m.cfg.Search.APIKey != "") + " " + m.cfg.Search.Provider,
	}, "   ")

	left := m.styles.appName.Render("dispatch")
	right := m.styles.status.Render(status)
	spacer := strings.Repeat(" ", max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)-4))
	return m.styles.header.Width(m.width).Render(left + spacer + right)
}

func (m Model) renderFooter() string {
	input := m.textarea.View()
	shortcuts := m.styles.shortcuts.Render("ctrl+p posts   ctrl+r repos   ctrl+s suche   ctrl+c abbrechen   ? hilfe")
	return m.styles.footer.Width(m.width).Render(input + "\n" + shortcuts)
}

func (m Model) renderMessage(msg chatMessage) string {
	switch msg.Kind {
	case kindUser:
		return m.styles.label.Foreground(lipgloss.Color("#4a8a9a")).Render("DU") + "\n" + m.styles.user.Width(m.viewport.Width-2).Render(msg.Body)
	case kindAgent:
		label := "AGENT"
		if m.streaming {
			label += " · schreibt..."
		}
		body := msg.Body
		if body == "" && m.streaming {
			body = m.spinner.View()
		}
		return m.styles.label.Foreground(palette.green).Render(label) + "\n" + m.styles.agent.Width(m.viewport.Width-2).Render(body)
	case kindTool:
		return m.styles.label.Foreground(palette.amber).Render("▶ TOOL") + "\n" + m.styles.tool.Width(m.viewport.Width-2).Render(msg.Body)
	default:
		return m.styles.label.Foreground(palette.purple).Render("SYSTEM") + "\n" + m.styles.system.Width(m.viewport.Width-2).Render(msg.Body)
	}
}

func waitForAgentChunk(ch <-chan provider.Chunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return agentDoneMsg{}
		}
		return agentChunkMsg(chunk)
	}
}

func statusDot(ok bool) string {
	if ok {
		return lipgloss.NewStyle().Foreground(palette.green).Render("●")
	}
	return lipgloss.NewStyle().Foreground(palette.warning).Render("●")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatToolCall(call provider.ToolCall) string {
	args := strings.TrimSpace(call.Arguments)
	if args == "" {
		return call.Name
	}
	return fmt.Sprintf("%s(%s)", call.Name, truncateText(compactJSON(args), 160))
}

func formatToolResult(result provider.ToolResult) string {
	body := strings.TrimSpace(result.Result)
	if body == "" {
		return result.Name + " → fertig"
	}
	return fmt.Sprintf("%s →\n%s", result.Name, shortenLines(body, 6, 700))
}

func compactJSON(value string) string {
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return oneLine(value)
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return oneLine(value)
	}
	return string(encoded)
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func shortenLines(value string, maxLines, maxRunes int) string {
	lines := strings.Split(value, "\n")
	visible := lines
	omittedLines := 0
	if len(lines) > maxLines {
		visible = lines[:maxLines]
		omittedLines = len(lines) - maxLines
	}

	shortened := truncateText(strings.Join(visible, "\n"), maxRunes)
	if omittedLines > 0 {
		shortened += fmt.Sprintf("\n… +%d Zeilen", omittedLines)
	}
	return shortened
}

func truncateText(value string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes-1]) + "…"
}
