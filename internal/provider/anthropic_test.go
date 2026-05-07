package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicCompleteStreamsTextDeltas(t *testing.T) {
	var gotRequest anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("x-api-key header = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Fatal("missing anthropic-version header")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message_start\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\"}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hallo \"}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Welt\"}}\n\n"))
		_, _ = w.Write([]byte("event: message_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n"))
		_, _ = w.Write([]byte("event: message_stop\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	p := NewAnthropicWithBaseURL("test-key", "claude-test", server.URL)
	ch, err := p.Complete(context.Background(), Request{
		System:    "Antworte kurz.",
		MaxTokens: 42,
		Messages:  []Message{{Role: RoleUser, Content: "Sag Hallo"}},
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
	if stop != "end_turn" {
		t.Fatalf("stop reason = %q", stop)
	}
	if gotRequest.Model != "claude-test" || gotRequest.MaxTokens != 42 || !gotRequest.Stream {
		t.Fatalf("unexpected request: %#v", gotRequest)
	}
	if gotRequest.System != "Antworte kurz." {
		t.Fatalf("system = %q", gotRequest.System)
	}
	if gotRequest.Messages[0].Role != string(RoleUser) {
		t.Fatalf("role = %q", gotRequest.Messages[0].Role)
	}
	if gotRequest.Messages[0].Content[0].Text != "Sag Hallo" {
		t.Fatalf("content = %#v", gotRequest.Messages[0].Content)
	}
}

func TestAnthropicCompleteReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewAnthropicWithBaseURL("test-key", "claude-test", server.URL)
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
