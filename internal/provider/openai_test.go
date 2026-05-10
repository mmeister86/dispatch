package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matthias/dispatch/config"
)

func TestOpenAICompleteStreamsDeltaContent(t *testing.T) {
	var gotRequest openAIRequest
	var gotRequestJSON map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequestJSON); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		raw, err := json.Marshal(gotRequestJSON)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		if err := json.Unmarshal(raw, &gotRequest); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hallo \"},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Welt\"},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewOpenAICompatible("openai", "test-key", "gpt-test", server.URL)
	ch, err := p.Complete(context.Background(), Request{
		System:    "Du bist dispatch.",
		MaxTokens: 88,
		Messages: []Message{
			{Role: RoleUser, Content: "Sag Hallo"},
			{Role: RoleAssistant, Content: "Klar."},
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	var text, stop string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		text += chunk.Text
		if chunk.StopReason != "" {
			stop = chunk.StopReason
		}
	}

	if text != "Hallo Welt" {
		t.Fatalf("streamed text = %q", text)
	}
	if stop != "stop" {
		t.Fatalf("stop reason = %q", stop)
	}
	if gotRequest.Model != "gpt-test" || gotRequest.MaxCompletionTokens != 88 || !gotRequest.Stream {
		t.Fatalf("unexpected request: %#v", gotRequest)
	}
	if _, ok := gotRequestJSON["max_tokens"]; ok {
		t.Fatalf("request used unsupported max_tokens parameter: %#v", gotRequestJSON)
	}
	if gotRequestJSON["max_completion_tokens"] != float64(88) {
		t.Fatalf("max_completion_tokens = %#v", gotRequestJSON["max_completion_tokens"])
	}
	if len(gotRequest.Messages) != 3 {
		t.Fatalf("messages length = %d", len(gotRequest.Messages))
	}
	if gotRequest.Messages[0].Role != "system" || gotRequest.Messages[0].Content != "Du bist dispatch." {
		t.Fatalf("system message = %#v", gotRequest.Messages[0])
	}
	if gotRequest.Messages[1].Role != "user" || gotRequest.Messages[1].Content != "Sag Hallo" {
		t.Fatalf("user message = %#v", gotRequest.Messages[1])
	}
	if gotRequest.Messages[2].Role != "assistant" || gotRequest.Messages[2].Content != "Klar." {
		t.Fatalf("assistant message = %#v", gotRequest.Messages[2])
	}
}

func TestOpenAICompleteCombinesStreamingToolCallDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"list_posts\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"start_date\\\":\"}}]},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"2026-04-01T00:00:00Z\\\"}\"}}]},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewOpenAICompatible("openai", "test-key", "gpt-test", server.URL)
	ch, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "List posts"}},
		Tools: []ToolDefinition{{
			Name:   "list_posts",
			Schema: map[string]any{"type": "object"},
		}},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	var toolCalls []ToolCall
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		if chunk.ToolCall != nil {
			toolCalls = append(toolCalls, *chunk.ToolCall)
		}
	}

	if len(toolCalls) != 1 {
		t.Fatalf("tool calls = %#v", toolCalls)
	}
	if toolCalls[0].ID != "call_1" || toolCalls[0].Name != "list_posts" {
		t.Fatalf("tool call = %#v", toolCalls[0])
	}
	if toolCalls[0].Arguments != `{"start_date":"2026-04-01T00:00:00Z"}` {
		t.Fatalf("arguments = %q", toolCalls[0].Arguments)
	}
}

func TestOpenAICompleteReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewOpenAICompatible("openai", "test-key", "gpt-test", server.URL)
	ch, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Complete returned immediate error: %v", err)
	}

	chunk, ok := <-ch
	if !ok {
		t.Fatal("expected error chunk")
	}
	if chunk.Err == nil {
		t.Fatal("expected chunk error")
	}
}

func TestNewFromConfigSupportsOpenAICompatibleProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{name: "openai", provider: "openai", model: "gpt-4o-mini"},
		{name: "openrouter", provider: "openrouter", model: "openai/gpt-4o-mini"},
		{name: "xai", provider: "xai", model: "grok-3-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewFromConfig(config.Config{LLM: config.LLMConfig{
				Provider: tt.provider,
				Model:    tt.model,
				APIKey:   "key",
			}})
			if err != nil {
				t.Fatalf("NewFromConfig returned error: %v", err)
			}
			if p.Name() != tt.provider {
				t.Fatalf("provider name = %q", p.Name())
			}
			if p.ModelID() != tt.model {
				t.Fatalf("model = %q", p.ModelID())
			}
		})
	}
}
