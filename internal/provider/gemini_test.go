package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiCompleteStreamsText(t *testing.T) {
	var got geminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:streamGenerateContent" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Fatalf("alt = %q", r.URL.Query().Get("alt"))
		}
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Fatalf("x-goog-api-key = %q", r.Header.Get("x-goog-api-key"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hallo \"}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Welt\"}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"finishReason\":\"STOP\"}]}\n\n"))
	}))
	defer server.Close()

	p := NewGeminiWithBaseURL("test-key", "gemini-test", server.URL)
	ch, err := p.Complete(context.Background(), Request{
		System: "Du bist dispatch.",
		Messages: []Message{
			{Role: RoleUser, Content: "Sag Hallo"},
			{Role: RoleAssistant, Content: "Klar"},
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
		t.Fatalf("text = %q", text)
	}
	if stop != "STOP" {
		t.Fatalf("stop = %q", stop)
	}
	if got.SystemInstruction == nil || got.SystemInstruction.Parts[0].Text != "Du bist dispatch." {
		t.Fatalf("system = %#v", got.SystemInstruction)
	}
	if got.Contents[0].Role != "user" || got.Contents[1].Role != "model" {
		t.Fatalf("roles = %#v", got.Contents)
	}
}
