package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/agent"
	clip "github.com/matthias/dispatch/internal/clipboard"
	"github.com/matthias/dispatch/internal/provider"
	"github.com/matthias/dispatch/internal/session"
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

type sessionStore interface {
	Load() (session.Session, error)
	Save(session.Session) error
	Clear() error
	List() ([]session.Summary, error)
	StartNew() (session.Session, error)
	Open(id string) (session.Session, error)
	Delete(id string) error
}

type imageReader interface {
	ReadImages(context.Context) ([]clip.Attachment, error)
	AttachPastedImagePaths(context.Context, string) ([]clip.Attachment, error)
}

type Model struct {
	cfg         config.Config
	agent       *agent.Agent
	sessions    sessionStore
	session     session.Session
	styles      styles
	viewport    viewport.Model
	textarea    textarea.Model
	spinner     spinner.Model
	messages    []chatMessage
	attachments []clip.Attachment
	imageReader imageReader
	width       int
	height      int
	ready       bool
	streaming   bool
	showHelp    bool
	stream      <-chan provider.Chunk
	activeMsg   int
}

func New(cfg config.Config, chatAgent *agent.Agent, warnings []string) Model {
	return newModel(cfg, chatAgent, warnings, nil)
}

func NewWithSessionStore(cfg config.Config, chatAgent *agent.Agent, warnings []string, store sessionStore) Model {
	return newModel(cfg, chatAgent, warnings, store)
}

func newModel(cfg config.Config, chatAgent *agent.Agent, warnings []string, store sessionStore) Model {
	ta := textarea.New()
	ta.Placeholder = "Nachricht an dispatch"
	ta.ShowLineNumbers = false
	ta.Prompt = "› "
	ta.CharLimit = 4000
	ta.SetHeight(1)
	styleTextarea(&ta)
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(palette.green)

	m := Model{
		cfg:         cfg,
		agent:       chatAgent,
		sessions:    store,
		session:     session.Session{Version: 1},
		styles:      newStyles(),
		textarea:    ta,
		spinner:     sp,
		imageReader: clip.NewReader(),
		activeMsg:   -1,
		messages:    defaultMessages(),
	}
	if store != nil {
		sess, err := store.Load()
		if err != nil {
			m.messages = append(m.messages, chatMessage{Kind: kindSystem, Body: fmt.Sprintf("Session konnte nicht geladen werden: %v", err)})
		} else {
			m.session = sess
			if len(sess.Messages) > 0 {
				m.messages = chatMessagesFromSession(sess.Messages)
			}
			if chatAgent != nil && len(sess.AgentHistory) > 0 {
				chatAgent.SetHistory(sess.AgentHistory)
			}
		}
	}
	for _, warning := range warnings {
		m.messages = append(m.messages, chatMessage{Kind: kindSystem, Body: warning})
	}
	m.refreshViewport()
	return m
}

func styleTextarea(ta *textarea.Model) {
	base := lipgloss.NewStyle().
		Foreground(palette.text)
	prompt := lipgloss.NewStyle().
		Foreground(palette.green)
	placeholder := lipgloss.NewStyle().
		Foreground(palette.dim)
	cursorLine := lipgloss.NewStyle().
		Foreground(palette.text)

	ta.FocusedStyle.Base = base
	ta.FocusedStyle.CursorLine = cursorLine
	ta.FocusedStyle.Placeholder = placeholder
	ta.FocusedStyle.Prompt = prompt
	ta.FocusedStyle.Text = base

	ta.BlurredStyle.Base = base
	ta.BlurredStyle.CursorLine = cursorLine
	ta.BlurredStyle.Placeholder = placeholder
	ta.BlurredStyle.Prompt = prompt
	ta.BlurredStyle.Text = base
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
		if msg.Paste && m.pasteImagePaths(string(msg.Runes)) {
			return m, nil
		}
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
		case "ctrl+v":
			if m.pasteClipboardImages() {
				return m, nil
			}
		case "ctrl+x":
			if len(m.attachments) > 0 {
				m.attachments = nil
				m.addMessageNoPersist(kindSystem, "Bildanhaenge entfernt.")
				return m, nil
			}
		case "ctrl+l":
			m.attachments = nil
			m.messages = nil
			if m.agent != nil {
				m.agent.ClearHistory()
			}
			if m.sessions != nil {
				if err := m.sessions.Clear(); err != nil {
					m.addMessage(kindSystem, fmt.Sprintf("Session konnte nicht geloescht werden: %v", err))
					return m, nil
				}
			}
			m.addMessageNoPersist(kindSystem, "Chat-Verlauf geleert.")
			return m, nil
		case "ctrl+p":
			return m.startAgent("Liste meine Postiz-Posts der letzten 30 Tage und der naechsten 30 Tage.")
		case "ctrl+o":
			return m.startAgent("Liste meine verbundenen Postiz-Kanaele und Provider.")
		case "ctrl+r":
			return m.startAgent("Zeige mir GitHub-Aktivitaet aus allen Repos der letzten 7 Tage.")
		case "ctrl+s":
			return m.startAgent("Welche Themen sind aktuell relevant fuer Developer Social Posts? Recherchiere mit Quellen.")
		case "enter":
			value := strings.TrimSpace(m.textarea.Value())
			attachments := append([]clip.Attachment(nil), m.attachments...)
			if value == "" && len(attachments) == 0 {
				return m, nil
			}
			m.textarea.Reset()
			m.attachments = nil
			if len(attachments) == 0 && isSessionCommand(value) {
				return m.handleSessionCommand(value)
			}
			visibleValue := value
			if visibleValue == "" {
				visibleValue = attachmentCountText(len(attachments)) + " angehaengt"
			}
			return m.startAgentWithPrompt(visibleValue, agentPromptWithAttachments(value, attachments))
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
			m.removeActiveAgentMessage()
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
		m.persistSession()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func isSessionCommand(value string) bool {
	return value == "/session" || strings.HasPrefix(value, "/session ")
}

func (m *Model) pasteClipboardImages() bool {
	if m.imageReader == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	attachments, err := m.imageReader.ReadImages(ctx)
	if err != nil {
		if errors.Is(err, clip.ErrNoImages) {
			return false
		}
		if errors.Is(err, clip.ErrUnavailable) {
			m.addMessageNoPersist(kindSystem, fmt.Sprintf("Bild-Paste nicht verfuegbar: %v", err))
			return true
		}
		m.addMessageNoPersist(kindSystem, fmt.Sprintf("Bild-Paste fehlgeschlagen: %v", err))
		return true
	}
	if len(attachments) == 0 {
		return false
	}
	m.attachments = append(m.attachments, attachments...)
	m.addMessageNoPersist(kindSystem, attachmentCountText(len(attachments))+" angehaengt. Mit Enter mitsenden, mit ctrl+x entfernen.")
	return true
}

func (m *Model) pasteImagePaths(value string) bool {
	if m.imageReader == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	attachments, err := m.imageReader.AttachPastedImagePaths(ctx, value)
	if err != nil {
		if errors.Is(err, clip.ErrNoImages) {
			return false
		}
		m.addMessageNoPersist(kindSystem, fmt.Sprintf("Bild-Paste fehlgeschlagen: %v", err))
		return true
	}
	if len(attachments) == 0 {
		return false
	}
	m.attachments = append(m.attachments, attachments...)
	m.addMessageNoPersist(kindSystem, attachmentCountText(len(attachments))+" angehaengt. Mit Enter mitsenden, mit ctrl+x entfernen.")
	return true
}

func (m Model) startAgent(value string) (tea.Model, tea.Cmd) {
	return m.startAgentWithPrompt(value, value)
}

func (m Model) startAgentWithPrompt(visibleValue, prompt string) (tea.Model, tea.Cmd) {
	m.addMessage(kindUser, visibleValue)
	if m.agent == nil {
		m.addMessage(kindAgent, "LLM-Streaming und Tools sind bereit, sobald ein LLM API Key konfiguriert ist. Starte `dispatch --setup` oder setze LLM_API_KEY.")
		return m, nil
	}
	stream, err := m.agent.Run(context.Background(), prompt)
	if err != nil {
		m.addMessage(kindSystem, fmt.Sprintf("Agent-Start fehlgeschlagen: %v", err))
		return m, nil
	}
	m.streaming = true
	m.stream = stream
	m.activeMsg = -1
	m.persistSession()
	return m, waitForAgentChunk(stream)
}

func (m Model) handleSessionCommand(value string) (tea.Model, tea.Cmd) {
	if m.sessions == nil {
		m.addMessageNoPersist(kindSystem, "Sessions sind in dieser Ansicht nicht verfuegbar.")
		return m, nil
	}

	fields := strings.Fields(value)
	if len(fields) == 1 {
		return m.showSessions()
	}

	switch fields[1] {
	case "new":
		return m.startNewSession()
	case "open":
		if len(fields) < 3 {
			m.addMessageNoPersist(kindSystem, "Nutze `/session open <id>`.")
			return m, nil
		}
		return m.openSession(fields[2])
	case "delete":
		if len(fields) < 3 {
			m.addMessageNoPersist(kindSystem, "Nutze `/session delete <id>`.")
			return m, nil
		}
		return m.deleteSession(fields[2])
	default:
		m.addMessageNoPersist(kindSystem, "Sessions\n/session\n/session new\n/session open <id>\n/session delete <id>")
		return m, nil
	}
}

func (m Model) showSessions() (tea.Model, tea.Cmd) {
	summaries, err := m.sessions.List()
	if err != nil {
		m.addMessageNoPersist(kindSystem, fmt.Sprintf("Sessions konnten nicht geladen werden: %v", err))
		return m, nil
	}
	m.addMessageNoPersist(kindSystem, formatSessionOverview(summaries))
	return m, nil
}

func (m Model) startNewSession() (tea.Model, tea.Cmd) {
	sess, err := m.sessions.StartNew()
	if err != nil {
		m.addMessageNoPersist(kindSystem, fmt.Sprintf("Neue Session konnte nicht gestartet werden: %v", err))
		return m, nil
	}
	m.session = sess
	m.attachments = nil
	m.messages = defaultMessages()
	if m.agent != nil {
		m.agent.ClearHistory()
	}
	m.addMessageNoPersist(kindSystem, fmt.Sprintf("Neue Session gestartet: %s", sess.ID))
	return m, nil
}

func (m Model) openSession(id string) (tea.Model, tea.Cmd) {
	sess, err := m.sessions.Open(id)
	if err != nil {
		m.addMessageNoPersist(kindSystem, fmt.Sprintf("Session konnte nicht geoeffnet werden: %v", err))
		return m, nil
	}
	m.session = sess
	m.attachments = nil
	if len(sess.Messages) > 0 {
		m.messages = chatMessagesFromSession(sess.Messages)
	} else {
		m.messages = defaultMessages()
	}
	if m.agent != nil {
		m.agent.SetHistory(sess.AgentHistory)
	}
	m.addMessageNoPersist(kindSystem, fmt.Sprintf("Session geoeffnet: %s", sess.ID))
	return m, nil
}

func (m Model) deleteSession(id string) (tea.Model, tea.Cmd) {
	if err := m.sessions.Delete(id); err != nil {
		m.addMessageNoPersist(kindSystem, fmt.Sprintf("Session konnte nicht geloescht werden: %v", err))
		return m, nil
	}
	m.addMessageNoPersist(kindSystem, fmt.Sprintf("Session geloescht: %s", id))
	return m, nil
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

	return m.paintCanvas(lipgloss.JoinVertical(lipgloss.Left, header, body, footer))
}

func (m *Model) layout() {
	const (
		headerHeight = 2
		bodyChrome   = 4
		footerHeight = 3
	)
	bodyHeight := max(1, m.height-headerHeight-bodyChrome-footerHeight)
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

func (m Model) paintCanvas(view string) string {
	if m.width <= 0 || m.height <= 0 {
		return m.styles.app.Render(view)
	}

	lines := strings.Split(view, "\n")
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	for len(lines) < m.height {
		lines = append(lines, "")
	}

	painted := make([]string, len(lines))
	for i, line := range lines {
		painted[i] = m.styles.app.Width(m.width).MaxWidth(m.width).Render(line)
	}
	return strings.Join(painted, "\n")
}

func (m *Model) addMessage(kind messageKind, body string) int {
	m.messages = append(m.messages, chatMessage{Kind: kind, Body: body})
	m.refreshViewport()
	m.persistSession()
	return len(m.messages) - 1
}

func (m *Model) addMessageNoPersist(kind messageKind, body string) int {
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
	m.persistSession()
}

func (m *Model) removeActiveAgentMessage() {
	if m.activeMsg < 0 || m.activeMsg >= len(m.messages) || m.messages[m.activeMsg].Kind != kindAgent {
		m.activeMsg = -1
		return
	}
	m.messages = append(m.messages[:m.activeMsg], m.messages[m.activeMsg+1:]...)
	m.activeMsg = -1
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
			Body: "Shortcuts\nctrl+p posts  ·  ctrl+o konten  ·  ctrl+r repos  ·  ctrl+s suche  ·  ctrl+l leeren  ·  ctrl+c abbrechen  ·  ctrl+q beenden",
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
		statusDot(m.cfg.Postiz.APIKey != "" || strings.Contains(m.cfg.Postiz.BaseURL, "/mcp/")) + " postiz",
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
	shortcutText := "ctrl+p posts   ctrl+o konten   ctrl+r repos   ctrl+s suche   ctrl+v bild   ctrl+c abbrechen   ? hilfe"
	if len(m.attachments) > 0 {
		shortcutText += "   ctrl+x bilder entfernen"
	}
	shortcuts := m.styles.shortcuts.Render(shortcutText)
	attachmentStatus := ""
	if len(m.attachments) > 0 {
		attachmentStatus = "\n" + m.styles.accent.Render(attachmentCountText(len(m.attachments))+" angehaengt")
	}
	return m.styles.footer.Width(m.width).Render(input + attachmentStatus + "\n" + shortcuts)
}

func (m Model) renderMessage(msg chatMessage) string {
	width := m.messageWidth()
	switch msg.Kind {
	case kindUser:
		return m.styles.label.Render("Du") + "\n" + m.styles.user.Width(width).Render(cleanInlineMarkdown(msg.Body))
	case kindAgent:
		label := "Antwort"
		if m.streaming {
			label += " · schreibt"
		}
		body := msg.Body
		if body == "" && m.streaming {
			body = m.spinner.View()
		}
		return m.styles.label.Render(label) + "\n" + m.styles.agent.Width(width).Render(cleanMessageMarkdown(body, width))
	case kindTool:
		name, detail := splitToolMessage(msg.Body)
		label := "Tool"
		if name != "" {
			label += " · " + name
		}
		return m.styles.label.Render(label) + "\n" + m.styles.tool.Width(width).Render(shortenLines(detail, 4, 420))
	default:
		return m.styles.label.Render("System") + "\n" + m.styles.system.Width(width).Render(cleanInlineMarkdown(msg.Body))
	}
}

func (m Model) messageWidth() int {
	width := max(20, m.viewport.Width-2)
	if width > 112 {
		return 112
	}
	return width
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

func agentPromptWithAttachments(value string, attachments []clip.Attachment) string {
	if len(attachments) == 0 {
		return value
	}
	var b strings.Builder
	value = strings.TrimSpace(value)
	if value != "" {
		b.WriteString(value)
	} else {
		b.WriteString("Der User hat Bilder ohne zusaetzlichen Text angehaengt.")
	}
	b.WriteString("\n\nAngehaengte Bilder fuer Postiz:\n")
	for i, attachment := range attachments {
		fmt.Fprintf(&b, "%d. %s", i+1, attachment.Path)
		if attachment.MIMEType != "" || attachment.OriginalName != "" {
			b.WriteString(" (")
			var parts []string
			if attachment.OriginalName != "" {
				parts = append(parts, "Name: "+attachment.OriginalName)
			}
			if attachment.MIMEType != "" {
				parts = append(parts, "Typ: "+attachment.MIMEType)
			}
			b.WriteString(strings.Join(parts, ", "))
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	b.WriteString("\nLade jedes lokale Bild zuerst mit dem Postiz-Tool upload_media hoch. Verwende die zurueckgegebenen Medien-URLs oder Pfade anschliessend in create_post.media_urls. Erstelle keinen Post ohne Vorschau und explizite Bestaetigung des Users.")
	return b.String()
}

func attachmentCountText(count int) string {
	if count == 1 {
		return "1 Bild"
	}
	return fmt.Sprintf("%d Bilder", count)
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

func min(a, b int) int {
	if a < b {
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

func splitToolMessage(body string) (string, string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ""
	}

	first, rest, found := strings.Cut(body, "\n")
	first = strings.TrimSpace(first)
	name := first
	if before, _, ok := strings.Cut(name, "("); ok {
		name = strings.TrimSpace(before)
	}
	if before, _, ok := strings.Cut(name, "→"); ok {
		name = strings.TrimSpace(before)
	}
	if !found {
		if detail, ok := strings.CutPrefix(first, name+"("); ok {
			return name, "(" + strings.TrimSpace(detail)
		}
		if detail, ok := strings.CutPrefix(first, name+" →"); ok {
			return name, strings.TrimSpace(detail)
		}
		return name, body
	}
	return name, strings.TrimSpace(rest)
}

func cleanMessageMarkdown(body string, width int) string {
	lines := strings.Split(body, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		cleaned = append(cleaned, cleanMarkdownLine(line, width))
	}
	return strings.Join(cleaned, "\n")
}

func cleanMarkdownLine(line string, width int) string {
	trimmed := strings.TrimSpace(line)
	if isMarkdownRule(trimmed) {
		return strings.Repeat("─", max(12, min(width, 56)))
	}
	if strings.HasPrefix(trimmed, "#") {
		return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
	}
	return cleanInlineMarkdown(line)
}

func cleanInlineMarkdown(value string) string {
	value = strings.ReplaceAll(value, "**", "")
	value = strings.ReplaceAll(value, "__", "")
	return value
}

func isMarkdownRule(value string) bool {
	if len(value) < 3 {
		return false
	}
	for _, ch := range value {
		if ch != '-' && ch != '_' && ch != '*' {
			return false
		}
	}
	return true
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

func defaultMessages() []chatMessage {
	return []chatMessage{
		{
			Kind: kindSystem,
			Body: "Willkommen bei dispatch.\nKeys werden lokal gelesen. Starte `dispatch --setup`, falls Integrationen fehlen.",
		},
		{
			Kind: kindAgent,
			Body: "Bereit. Frag mich nach Repo-Aktivitaet, Recherchethemen oder geplanten Posts.",
		},
	}
}

func chatMessagesFromSession(messages []session.Message) []chatMessage {
	restored := make([]chatMessage, 0, len(messages))
	for _, msg := range messages {
		restored = append(restored, chatMessage{
			Kind: messageKindFromSession(msg.Kind),
			Body: msg.Body,
		})
	}
	return restored
}

func messageKindFromSession(kind session.MessageKind) messageKind {
	switch kind {
	case session.KindUser:
		return kindUser
	case session.KindAgent:
		return kindAgent
	case session.KindTool:
		return kindTool
	default:
		return kindSystem
	}
}

func sessionKindFromMessage(kind messageKind) session.MessageKind {
	switch kind {
	case kindUser:
		return session.KindUser
	case kindAgent:
		return session.KindAgent
	case kindTool:
		return session.KindTool
	default:
		return session.KindSystem
	}
}

func (m Model) sessionMessages() []session.Message {
	messages := make([]session.Message, 0, len(m.messages))
	for _, msg := range m.messages {
		messages = append(messages, session.Message{
			Kind: sessionKindFromMessage(msg.Kind),
			Body: msg.Body,
		})
	}
	return messages
}

func (m Model) agentHistory() []provider.Message {
	if m.agent == nil {
		return nil
	}
	return m.agent.History()
}

func (m *Model) persistSession() {
	if m.sessions == nil {
		return
	}
	sess := m.session
	sess.Messages = m.sessionMessages()
	sess.AgentHistory = m.agentHistory()
	_ = m.sessions.Save(sess)
}

func formatSessionOverview(summaries []session.Summary) string {
	if len(summaries) == 0 {
		return "Sessions\nKeine gespeicherten Sessions."
	}
	var lines []string
	lines = append(lines, "Sessions")
	for _, summary := range summaries {
		marker := " "
		if summary.Current {
			marker = "*"
		}
		title := strings.TrimSpace(summary.Title)
		if title == "" {
			title = "Neue Session"
		}
		lines = append(lines, fmt.Sprintf("%s %s  %s  %d Nachrichten", marker, summary.ID, title, summary.MessageCount))
	}
	return strings.Join(lines, "\n")
}
