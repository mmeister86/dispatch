package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/agent"
	"github.com/matthias/dispatch/internal/clipboard"
	"github.com/matthias/dispatch/internal/provider"
	"github.com/matthias/dispatch/internal/session"
	"github.com/muesli/termenv"
)

func TestToolMessagesRenderBeforeDelayedAgentReply(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.layout()
	m.streaming = true
	m.activeMsg = -1

	model, _ := m.Update(agentChunkMsg(provider.Chunk{
		ToolCall: &provider.ToolCall{Name: "get_activity", Arguments: `{"repos":["all"]}`},
	}))
	m = model.(Model)
	model, _ = m.Update(agentChunkMsg(provider.Chunk{
		ToolResult: &provider.ToolResult{Name: "get_activity", Result: "Repo: dispatch\n- feat: tools"},
	}))
	m = model.(Model)
	model, _ = m.Update(agentChunkMsg(provider.Chunk{Text: "Hier ist die Übersicht."}))
	m = model.(Model)

	var toolIndex, agentIndex = -1, -1
	for i, msg := range m.messages {
		switch {
		case msg.Kind == kindTool && strings.Contains(msg.Body, "get_activity"):
			if toolIndex == -1 {
				toolIndex = i
			}
		case msg.Kind == kindAgent && strings.Contains(msg.Body, "Hier ist die Übersicht."):
			agentIndex = i
		}
	}

	if toolIndex == -1 || agentIndex == -1 {
		t.Fatalf("missing tool or agent message: %#v", m.messages)
	}
	if toolIndex > agentIndex {
		t.Fatalf("tool message index %d should be before agent message index %d", toolIndex, agentIndex)
	}
}

func TestToolCallRemovesPreToolAgentText(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.layout()
	m.streaming = true
	m.activeMsg = -1
	baseline := len(m.messages)

	model, _ := m.Update(agentChunkMsg(provider.Chunk{Text: "Ich schaue kurz nach."}))
	m = model.(Model)
	model, _ = m.Update(agentChunkMsg(provider.Chunk{
		ToolCall: &provider.ToolCall{Name: "get_activity", Arguments: `{"repos":["all"]}`},
	}))
	m = model.(Model)
	model, _ = m.Update(agentChunkMsg(provider.Chunk{
		ToolResult: &provider.ToolResult{Name: "get_activity", Result: "Repo: dispatch\n- feat: tools"},
	}))
	m = model.(Model)
	model, _ = m.Update(agentChunkMsg(provider.Chunk{Text: "Hier ist die Übersicht."}))
	m = model.(Model)

	var firstToolIndex, firstAgentIndex = -1, -1
	for i, msg := range m.messages[baseline:] {
		if firstToolIndex == -1 && msg.Kind == kindTool {
			firstToolIndex = baseline + i
		}
		if firstAgentIndex == -1 && msg.Kind == kindAgent {
			firstAgentIndex = baseline + i
		}
		if strings.Contains(msg.Body, "Ich schaue kurz nach.") {
			t.Fatalf("pre-tool agent text should not stay in the transcript once a tool is called: %#v", m.messages)
		}
	}

	if firstToolIndex == -1 || firstAgentIndex == -1 {
		t.Fatalf("missing tool or agent message: %#v", m.messages)
	}
	if firstAgentIndex < firstToolIndex {
		t.Fatalf("first agent message index %d should not be before first tool index %d", firstAgentIndex, firstToolIndex)
	}
}

func TestFormatToolCallCompactsAndTruncatesArguments(t *testing.T) {
	body := formatToolCall(provider.ToolCall{
		Name: "search",
		Arguments: `{
			"query": "dispatch terminal user interface transcript rendering",
			"include_domains": ["example.com", "docs.example.com"]
		}`,
	})

	if strings.Contains(body, "\n") {
		t.Fatalf("tool call should be one line: %q", body)
	}
	if !strings.HasPrefix(body, `search({"include_domains":["example.com","docs.example.com"],"query":"dispatch terminal user interface transcript rendering"}`) {
		t.Fatalf("tool call was not compacted: %q", body)
	}
}

func TestFormatToolResultShortensVerboseOutput(t *testing.T) {
	body := formatToolResult(provider.ToolResult{
		Name:   "get_activity",
		Result: strings.Join([]string{"one", "two", "three", "four", "five", "six", "seven", "eight"}, "\n"),
	})

	if strings.Contains(body, "seven") || strings.Contains(body, "eight") {
		t.Fatalf("tool result should omit lines after the limit: %q", body)
	}
	if !strings.Contains(body, "… +2 Zeilen") {
		t.Fatalf("tool result should mention omitted lines: %q", body)
	}
}

func TestAgentMarkdownIsCleanedForTerminalReading(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.width = 100
	m.height = 24
	m.layout()

	rendered := m.renderMessage(chatMessage{
		Kind: kindAgent,
		Body: "### Inaktive Repos\n---\n**Fazit:** Bitte als Post vorbereiten.",
	})

	if strings.Contains(rendered, "###") || strings.Contains(rendered, "**") {
		t.Fatalf("agent markdown markers should be cleaned from terminal output: %q", rendered)
	}
	if strings.Contains(rendered, "---") {
		t.Fatalf("horizontal markdown rules should render as terminal separators: %q", rendered)
	}
	if !strings.Contains(rendered, "Inaktive Repos") || !strings.Contains(rendered, "Fazit:") {
		t.Fatalf("rendered message lost important content: %q", rendered)
	}
}

func TestAgentMessageUsesReadableLineWidth(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.width = 180
	m.height = 24
	m.layout()

	rendered := m.renderMessage(chatMessage{
		Kind: kindAgent,
		Body: "Fazit: " + strings.Repeat("dieser Abschnitt bleibt auch in sehr breiten Terminals gut lesbar ", 4),
	})

	for _, line := range strings.Split(rendered, "\n") {
		if width := visibleWidth(line); width > 116 {
			t.Fatalf("rendered line width = %d, want <= 116\nline: %q\nrendered:\n%s", width, line, rendered)
		}
	}
}

func TestToolResultRendersAsCompactStatusBlock(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.width = 100
	m.height = 24
	m.layout()

	rendered := m.renderMessage(chatMessage{
		Kind: kindTool,
		Body: formatToolResult(provider.ToolResult{
			Name:   "get_activity",
			Result: strings.Join([]string{"one", "two", "three", "four", "five", "six"}, "\n"),
		}),
	})

	if strings.Contains(rendered, "▶ TOOL") {
		t.Fatalf("tool label should be calmer than the old loud marker: %q", rendered)
	}
	if !strings.Contains(rendered, "Tool · get_activity") {
		t.Fatalf("tool result should expose the tool name in a compact label: %q", rendered)
	}
	if strings.Contains(rendered, "six") {
		t.Fatalf("tool result preview should be shorter than full verbose output: %q", rendered)
	}
}

func TestViewRendersExactTerminalSizeWithoutBackgroundPane(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
	})

	m := New(config.Config{}, nil, nil)
	m.width = 96
	m.height = 24
	m.layout()

	rendered := m.View()
	lines := strings.Split(rendered, "\n")

	if len(lines) != m.height {
		t.Fatalf("rendered line count = %d, want %d\n%s", len(lines), m.height, rendered)
	}
	for i, line := range lines {
		if width := visibleWidth(line); width != m.width {
			t.Fatalf("line %d width = %d, want %d\nline: %q\nrendered:\n%s", i+1, width, m.width, line, rendered)
		}
		if strings.Contains(line, "48;2;") {
			t.Fatalf("line %d should not paint a full-width background pane: %q", i+1, line)
		}
	}
}

func TestMessageRenderingDoesNotPaintInlineBackgrounds(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
	})

	m := New(config.Config{}, nil, nil)
	m.width = 96
	m.height = 24
	m.layout()

	rendered := m.renderMessage(chatMessage{
		Kind: kindAgent,
		Body: "Hier ist Text ohne Terminal-Default-Hintergrund.",
	})

	for i, line := range strings.Split(rendered, "\n") {
		if strings.TrimSpace(stripANSI(line)) == "" {
			continue
		}
		if strings.Contains(line, "48;2;") {
			t.Fatalf("message line %d should not paint inline background blocks: %q", i+1, line)
		}
	}
}

type memorySessionStore struct {
	loaded  session.Session
	listed  []session.Summary
	opened  map[string]session.Session
	deleted string
	saved   []session.Session
	cleared bool
	started session.Session
}

type tuiFakeProvider struct{}

func (p tuiFakeProvider) Complete(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk)
	close(ch)
	return ch, nil
}

func (p tuiFakeProvider) Name() string    { return "fake" }
func (p tuiFakeProvider) ModelID() string { return "fake-model" }

func (s *memorySessionStore) Load() (session.Session, error) {
	return s.loaded, nil
}

func (s *memorySessionStore) Save(sess session.Session) error {
	s.saved = append(s.saved, sess)
	return nil
}

func (s *memorySessionStore) Clear() error {
	s.cleared = true
	return nil
}

func (s *memorySessionStore) List() ([]session.Summary, error) {
	return s.listed, nil
}

func (s *memorySessionStore) StartNew() (session.Session, error) {
	if s.started.ID == "" {
		s.started = session.Session{ID: "new-session"}
	}
	s.loaded = s.started
	return s.started, nil
}

func (s *memorySessionStore) Open(id string) (session.Session, error) {
	sess := s.opened[id]
	s.loaded = sess
	return sess, nil
}

func (s *memorySessionStore) Delete(id string) error {
	s.deleted = id
	return nil
}

func TestNewWithSessionStoreRestoresTranscript(t *testing.T) {
	chatAgent := agent.New(tuiFakeProvider{}, "", 20)
	store := &memorySessionStore{
		loaded: session.Session{
			Messages: []session.Message{
				{Kind: session.KindUser, Body: "Plane drei Posts."},
				{Kind: session.KindAgent, Body: "Ich habe den Plan begonnen."},
			},
			AgentHistory: []provider.Message{
				{Role: provider.RoleUser, Content: "Plane drei Posts."},
				{Role: provider.RoleAssistant, Content: "Ich habe den Plan begonnen."},
			},
		},
	}

	m := NewWithSessionStore(config.Config{}, chatAgent, nil, store)
	m.width = 80
	m.height = 20
	m.layout()

	if len(m.messages) != 2 {
		t.Fatalf("messages length = %d, want restored transcript only", len(m.messages))
	}
	if m.messages[0].Kind != kindUser || m.messages[0].Body != "Plane drei Posts." {
		t.Fatalf("first restored message mismatch: %#v", m.messages[0])
	}
	if !strings.Contains(m.viewport.View(), "Ich habe den Plan begonnen.") {
		t.Fatalf("viewport should render restored session: %q", m.viewport.View())
	}
	if len(chatAgent.History()) != 2 {
		t.Fatalf("agent history should be restored: %#v", chatAgent.History())
	}
}

func TestCtrlLClearsTranscriptAgentHistoryAndPersistedSession(t *testing.T) {
	chatAgent := agent.New(tuiFakeProvider{}, "", 20)
	chatAgent.SetHistory([]provider.Message{{Role: provider.RoleUser, Content: "Alter Kontext"}})
	store := &memorySessionStore{
		loaded: session.Session{
			Messages: []session.Message{{Kind: session.KindUser, Body: "Alter Plan"}},
		},
	}
	m := NewWithSessionStore(config.Config{}, chatAgent, nil, store)
	m.layout()

	model, _ := m.Update(teaKey("ctrl+l"))
	m = model.(Model)

	if !store.cleared {
		t.Fatal("ctrl+l should clear persisted session")
	}
	if len(m.messages) != 1 || !strings.Contains(m.messages[0].Body, "Chat-Verlauf geleert.") {
		t.Fatalf("messages after clear = %#v", m.messages)
	}
	if len(chatAgent.History()) != 0 {
		t.Fatalf("agent history should be cleared: %#v", chatAgent.History())
	}
}

func TestPlainUpDoesNotRestoreSubmittedInput(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.layout()
	m.textarea.SetValue("Welche Themen sind aktuell relevant?")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if got := m.textarea.Value(); got != "" {
		t.Fatalf("textarea after submit = %q, want empty", got)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(Model)

	if got := m.textarea.Value(); got != "" {
		t.Fatalf("plain up should leave answered input cleared, got %q", got)
	}
}

func TestAgentChunkPersistsVisibleAssistantText(t *testing.T) {
	store := &memorySessionStore{}
	m := NewWithSessionStore(config.Config{}, nil, nil, store)
	m.layout()
	m.streaming = true
	m.activeMsg = -1

	model, _ := m.Update(agentChunkMsg(provider.Chunk{Text: "Erster Satz."}))
	m = model.(Model)
	model, _ = m.Update(agentChunkMsg(provider.Chunk{Text: " Zweiter Satz."}))
	m = model.(Model)

	if len(store.saved) < 2 {
		t.Fatalf("expected save after each text chunk, got %d saves", len(store.saved))
	}
	last := store.saved[len(store.saved)-1]
	if len(last.Messages) == 0 || last.Messages[len(last.Messages)-1].Body != "Erster Satz. Zweiter Satz." {
		t.Fatalf("last saved transcript mismatch: %#v", last.Messages)
	}
	if len(last.AgentHistory) != 0 {
		t.Fatalf("partial assistant text should not enter agent history until turn completes: %#v", last.AgentHistory)
	}
}

func TestSessionCommandListsSessionsWithoutCallingAgent(t *testing.T) {
	fp := &countingProvider{}
	chatAgent := agent.New(fp, "", 20)
	store := &memorySessionStore{
		listed: []session.Summary{
			{ID: "abc123", Title: "Launch plan", MessageCount: 4, Current: true},
			{ID: "def456", Title: "Follow-up", MessageCount: 2},
		},
	}
	m := NewWithSessionStore(config.Config{}, chatAgent, nil, store)
	m.layout()
	m.textarea.SetValue("/session")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if fp.calls != 0 {
		t.Fatalf("/session should not call the agent, calls = %d", fp.calls)
	}
	if !containsMessage(m.messages, "abc123") || !containsMessage(m.messages, "Launch plan") {
		t.Fatalf("session overview not rendered: %#v", m.messages)
	}
}

func TestSessionNewStartsFreshLocalSession(t *testing.T) {
	chatAgent := agent.New(tuiFakeProvider{}, "", 20)
	chatAgent.SetHistory([]provider.Message{{Role: provider.RoleUser, Content: "Alter Kontext"}})
	store := &memorySessionStore{started: session.Session{ID: "fresh"}}
	m := NewWithSessionStore(config.Config{}, chatAgent, nil, store)
	m.layout()
	m.textarea.SetValue("/session new")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if len(chatAgent.History()) != 0 {
		t.Fatalf("new session should clear agent history: %#v", chatAgent.History())
	}
	if !containsMessage(m.messages, "Neue Session gestartet: fresh") {
		t.Fatalf("new session confirmation missing: %#v", m.messages)
	}
}

func TestSessionOpenRestoresSelectedSession(t *testing.T) {
	chatAgent := agent.New(tuiFakeProvider{}, "", 20)
	store := &memorySessionStore{
		opened: map[string]session.Session{
			"abc123": {
				ID: "abc123",
				Messages: []session.Message{
					{Kind: session.KindUser, Body: "Alter Plan"},
					{Kind: session.KindAgent, Body: "Alter Entwurf"},
				},
				AgentHistory: []provider.Message{
					{Role: provider.RoleUser, Content: "Alter Plan"},
					{Role: provider.RoleAssistant, Content: "Alter Entwurf"},
				},
			},
		},
	}
	m := NewWithSessionStore(config.Config{}, chatAgent, nil, store)
	m.layout()
	m.textarea.SetValue("/session open abc123")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if !containsMessage(m.messages, "Alter Entwurf") {
		t.Fatalf("opened transcript missing: %#v", m.messages)
	}
	if len(chatAgent.History()) != 2 {
		t.Fatalf("opened agent history missing: %#v", chatAgent.History())
	}
}

func TestSessionDeleteIsLocalCommand(t *testing.T) {
	fp := &countingProvider{}
	chatAgent := agent.New(fp, "", 20)
	store := &memorySessionStore{}
	m := NewWithSessionStore(config.Config{}, chatAgent, nil, store)
	m.layout()
	m.textarea.SetValue("/session delete old123")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if fp.calls != 0 {
		t.Fatalf("/session delete should not call the agent, calls = %d", fp.calls)
	}
	if store.deleted != "old123" {
		t.Fatalf("deleted id = %q, want old123", store.deleted)
	}
	if !containsMessage(m.messages, "Session geloescht: old123") {
		t.Fatalf("delete confirmation missing: %#v", m.messages)
	}
}

func TestCtrlVAppendsClipboardImageAttachments(t *testing.T) {
	reader := &fakeImageReader{attachments: []clipboard.Attachment{
		{Path: "/tmp/one.png", OriginalName: "one.png", MIMEType: "image/png", Source: "clipboard"},
		{Path: "/tmp/two.jpg", OriginalName: "two.jpg", MIMEType: "image/jpeg", Source: "file"},
	}}
	m := New(config.Config{}, nil, nil)
	m.imageReader = reader
	m.width = 96
	m.height = 24
	m.layout()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m = model.(Model)

	if reader.calls != 1 {
		t.Fatalf("ReadImages calls = %d, want 1", reader.calls)
	}
	if len(m.attachments) != 2 {
		t.Fatalf("attachments = %#v", m.attachments)
	}
	if !strings.Contains(m.renderFooter(), "2 Bilder angehaengt") {
		t.Fatalf("footer should show attachment count: %q", m.renderFooter())
	}
	if !containsMessage(m.messages, "2 Bilder angehaengt") {
		t.Fatalf("pasting attachments should add a visible confirmation: %#v", m.messages)
	}
	if containsMessage(m.messages, "/tmp/one.png") {
		t.Fatalf("pasting attachments should not write paths to transcript: %#v", m.messages)
	}
}

func TestFooterExplainsReliableImagePasteShortcut(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.width = 110
	m.height = 24
	m.layout()

	footer := strings.Join(strings.Fields(m.renderFooter()), " ")

	if !strings.Contains(footer, "ctrl+v bild einfuegen") {
		t.Fatalf("footer should explain reliable image paste shortcut: %q", footer)
	}
	if !strings.Contains(footer, "ctrl+x anhaenge loeschen") {
		t.Fatalf("footer should explain attachment removal shortcut: %q", footer)
	}
	if !strings.Contains(footer, "cmd+v muss vom Terminal weitergereicht werden") {
		t.Fatalf("footer should explain cmd+v terminal dependency: %q", footer)
	}
}

func TestKittySuperVPastesClipboardImageAttachments(t *testing.T) {
	reader := &fakeImageReader{attachments: []clipboard.Attachment{
		{Path: "/tmp/screenshot.png", OriginalName: "screenshot.png", MIMEType: "image/png", Source: "clipboard"},
	}}
	m := New(config.Config{}, nil, nil)
	m.imageReader = reader
	m.width = 96
	m.height = 24
	m.layout()

	model, _ := m.Update(fakeStringerMsg("?CSI[49 49 56 59 57 117]?"))
	m = model.(Model)

	if reader.calls != 1 {
		t.Fatalf("ReadImages calls = %d, want 1", reader.calls)
	}
	if len(m.attachments) != 1 {
		t.Fatalf("attachments = %#v", m.attachments)
	}
	if !containsMessage(m.messages, "1 Bild angehaengt") {
		t.Fatalf("pasting attachments should add a visible confirmation: %#v", m.messages)
	}
}

func TestDebugKeysCommandLogsUnknownCSIEvents(t *testing.T) {
	reader := &fakeImageReader{err: clipboard.ErrNoImages}
	m := New(config.Config{}, nil, nil)
	m.imageReader = reader
	m.layout()
	m.textarea.SetValue("/debug keys")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if !m.keyDebug {
		t.Fatal("key debug should be enabled")
	}

	model, _ = m.Update(fakeStringerMsg("?CSI[49 49 56 59 57 117]?"))
	m = model.(Model)

	if !containsMessage(m.messages, `Debug key: unknown-csi="?CSI[49 49 56 59 57 117]?" decoded="118;9u" kitty_super_v=true`) {
		t.Fatalf("debug log missing unknown CSI details: %#v", m.messages)
	}
}

func TestBracketedPasteImagePathAppendsAttachmentInsteadOfText(t *testing.T) {
	reader := &fakeImageReader{pastedAttachments: []clipboard.Attachment{
		{Path: "/tmp/cached-photo.png", OriginalName: "photo.png", MIMEType: "image/png", Source: "file"},
	}}
	m := New(config.Config{}, nil, nil)
	m.imageReader = reader
	m.width = 96
	m.height = 24
	m.layout()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/Users/me/photo.png"), Paste: true})
	m = model.(Model)

	if reader.pasteCalls != 1 || reader.lastPasted != "/Users/me/photo.png" {
		t.Fatalf("AttachPastedImagePaths calls=%d value=%q", reader.pasteCalls, reader.lastPasted)
	}
	if len(m.attachments) != 1 {
		t.Fatalf("attachments = %#v", m.attachments)
	}
	if got := m.textarea.Value(); got != "" {
		t.Fatalf("pasted image path should not remain in textarea, got %q", got)
	}
	if !containsMessage(m.messages, "1 Bild angehaengt") {
		t.Fatalf("confirmation missing: %#v", m.messages)
	}
}

func TestBracketedPastePlainTextStillUpdatesTextarea(t *testing.T) {
	reader := &fakeImageReader{pasteErr: clipboard.ErrNoImages}
	m := New(config.Config{}, nil, nil)
	m.imageReader = reader
	m.layout()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("normal pasted text"), Paste: true})
	m = model.(Model)

	if reader.pasteCalls != 1 {
		t.Fatalf("AttachPastedImagePaths calls = %d, want 1", reader.pasteCalls)
	}
	if got := m.textarea.Value(); got != "normal pasted text" {
		t.Fatalf("textarea = %q, want pasted text", got)
	}
	if len(m.attachments) != 0 {
		t.Fatalf("attachments = %#v", m.attachments)
	}
}

func TestEnterSendsAttachmentInstructionsToAgentAndClearsPendingAttachments(t *testing.T) {
	chatAgent := agent.New(tuiFakeProvider{}, "", 20)
	m := New(config.Config{}, chatAgent, nil)
	m.layout()
	m.textarea.SetValue("Plane einen Launch-Post.")
	m.attachments = []clipboard.Attachment{
		{Path: "/tmp/launch.png", OriginalName: "launch.png", MIMEType: "image/png", Source: "clipboard"},
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if len(m.attachments) != 0 {
		t.Fatalf("attachments should be cleared from pending input after submit: %#v", m.attachments)
	}
	if !containsMessage(m.messages, "Plane einen Launch-Post.") {
		t.Fatalf("visible user text missing: %#v", m.messages)
	}
	history := chatAgent.History()
	if len(history) == 0 {
		t.Fatal("agent history should contain submitted prompt")
	}
	prompt := history[len(history)-1].Content
	if !strings.Contains(prompt, "Plane einen Launch-Post.") ||
		!strings.Contains(prompt, "/tmp/launch.png") ||
		!strings.Contains(prompt, "upload_media") ||
		!strings.Contains(prompt, "media_urls") {
		t.Fatalf("agent prompt missing attachment instructions:\n%s", prompt)
	}
}

func TestEnterAcceptsImageOnlyPostDraft(t *testing.T) {
	chatAgent := agent.New(tuiFakeProvider{}, "", 20)
	m := New(config.Config{}, chatAgent, nil)
	m.layout()
	m.attachments = []clipboard.Attachment{
		{Path: "/tmp/only.png", OriginalName: "only.png", MIMEType: "image/png", Source: "clipboard"},
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if !containsMessage(m.messages, "1 Bild angehaengt") {
		t.Fatalf("image-only submit should still create a visible user turn: %#v", m.messages)
	}
	history := chatAgent.History()
	if len(history) == 0 || !strings.Contains(history[len(history)-1].Content, "/tmp/only.png") {
		t.Fatalf("image-only submit should include attachment path in agent prompt: %#v", history)
	}
}

func TestCtrlXClearsPendingImageAttachments(t *testing.T) {
	m := New(config.Config{}, nil, nil)
	m.layout()
	m.attachments = []clipboard.Attachment{
		{Path: "/tmp/one.png", OriginalName: "one.png", MIMEType: "image/png", Source: "clipboard"},
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	m = model.(Model)

	if len(m.attachments) != 0 {
		t.Fatalf("attachments should be cleared: %#v", m.attachments)
	}
	if !containsMessage(m.messages, "Bildanhaenge entfernt") {
		t.Fatalf("clear confirmation missing: %#v", m.messages)
	}
}

func teaKey(value string) tea.KeyMsg {
	if value == "ctrl+l" {
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

type countingProvider struct {
	calls int
}

func (p *countingProvider) Complete(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.calls++
	ch := make(chan provider.Chunk)
	close(ch)
	return ch, nil
}

func (p *countingProvider) Name() string    { return "counting" }
func (p *countingProvider) ModelID() string { return "counting-model" }

func containsMessage(messages []chatMessage, text string) bool {
	for _, msg := range messages {
		if strings.Contains(msg.Body, text) {
			return true
		}
	}
	return false
}

func visibleWidth(value string) int {
	return lipgloss.Width(value)
}

func stripANSI(value string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if inEscape {
			if ch >= '@' && ch <= '~' {
				inEscape = false
			}
			continue
		}
		if ch == '\x1b' {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

type fakeStringerMsg string

func (m fakeStringerMsg) String() string {
	return string(m)
}

type fakeImageReader struct {
	attachments       []clipboard.Attachment
	err               error
	calls             int
	pastedAttachments []clipboard.Attachment
	pasteErr          error
	pasteCalls        int
	lastPasted        string
}

func (f *fakeImageReader) ReadImages(context.Context) ([]clipboard.Attachment, error) {
	f.calls++
	return append([]clipboard.Attachment(nil), f.attachments...), f.err
}

func (f *fakeImageReader) AttachPastedImagePaths(_ context.Context, value string) ([]clipboard.Attachment, error) {
	f.pasteCalls++
	f.lastPasted = value
	return append([]clipboard.Attachment(nil), f.pastedAttachments...), f.pasteErr
}
