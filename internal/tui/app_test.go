package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/agent"
	"github.com/matthias/dispatch/internal/provider"
	"github.com/matthias/dispatch/internal/session"
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

type memorySessionStore struct {
	loaded  session.Session
	saved   []session.Session
	cleared bool
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

func teaKey(value string) tea.KeyMsg {
	if value == "ctrl+l" {
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}
