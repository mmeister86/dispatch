package tui

import (
	"strings"
	"testing"

	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/provider"
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
