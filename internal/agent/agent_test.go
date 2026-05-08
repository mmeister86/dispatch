package agent

import (
	"context"
	"testing"

	"github.com/matthias/dispatch/internal/provider"
)

type fakeProvider struct {
	requests []provider.Request
	chunks   []provider.Chunk
}

func (f *fakeProvider) Complete(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	f.requests = append(f.requests, req)
	ch := make(chan provider.Chunk, len(f.chunks))
	for _, chunk := range f.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (f *fakeProvider) Name() string    { return "fake" }
func (f *fakeProvider) ModelID() string { return "fake-model" }

func TestAgentRunStreamsChunksAndStoresHistory(t *testing.T) {
	fp := &fakeProvider{
		chunks: []provider.Chunk{
			{Text: "Hallo "},
			{Text: "Matthias"},
			{StopReason: "end_turn"},
		},
	}
	a := New(fp, "Du bist dispatch.", 20)

	ch, err := a.Run(context.Background(), "Moin")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var text string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		text += chunk.Text
	}

	if text != "Hallo Matthias" {
		t.Fatalf("streamed text = %q", text)
	}
	if len(fp.requests) != 1 {
		t.Fatalf("provider calls = %d", len(fp.requests))
	}
	if fp.requests[0].System != "Du bist dispatch." {
		t.Fatalf("system = %q", fp.requests[0].System)
	}
	if len(fp.requests[0].Messages) != 1 || fp.requests[0].Messages[0].Content != "Moin" {
		t.Fatalf("request history = %#v", fp.requests[0].Messages)
	}

	history := a.History()
	if len(history) != 2 {
		t.Fatalf("history length = %d", len(history))
	}
	if history[0].Role != provider.RoleUser || history[0].Content != "Moin" {
		t.Fatalf("user history = %#v", history[0])
	}
	if history[1].Role != provider.RoleAssistant || history[1].Content != "Hallo Matthias" {
		t.Fatalf("assistant history = %#v", history[1])
	}
}

func TestAgentTrimsOldHistory(t *testing.T) {
	fp := &fakeProvider{chunks: []provider.Chunk{{Text: "ok"}}}
	a := New(fp, "", 2)

	for _, input := range []string{"one", "two", "three"} {
		ch, err := a.Run(context.Background(), input)
		if err != nil {
			t.Fatalf("Run(%q): %v", input, err)
		}
		for range ch {
		}
	}

	history := a.History()
	if len(history) != 2 {
		t.Fatalf("history length = %d", len(history))
	}
	if history[0].Content != "three" || history[1].Content != "ok" {
		t.Fatalf("trimmed history = %#v", history)
	}
}

func TestAgentSetHistoryRestoresContextForNextRun(t *testing.T) {
	fp := &fakeProvider{chunks: []provider.Chunk{{Text: "weiter"}}}
	a := New(fp, "", 20)
	a.SetHistory([]provider.Message{
		{Role: provider.RoleUser, Content: "Erstelle einen Plan."},
		{Role: provider.RoleAssistant, Content: "Abschnitt 1 steht."},
	})

	ch, err := a.Run(context.Background(), "Mach weiter.")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for range ch {
	}

	if len(fp.requests) != 1 {
		t.Fatalf("provider calls = %d", len(fp.requests))
	}
	got := fp.requests[0].Messages
	if len(got) != 3 {
		t.Fatalf("request history length = %d, want 3: %#v", len(got), got)
	}
	if got[0].Content != "Erstelle einen Plan." || got[1].Content != "Abschnitt 1 steht." || got[2].Content != "Mach weiter." {
		t.Fatalf("request history mismatch: %#v", got)
	}
}

func TestAgentClearHistoryRemovesRestoredContext(t *testing.T) {
	fp := &fakeProvider{chunks: []provider.Chunk{{Text: "neu"}}}
	a := New(fp, "", 20)
	a.SetHistory([]provider.Message{
		{Role: provider.RoleUser, Content: "Alter Kontext"},
	})
	a.ClearHistory()

	ch, err := a.Run(context.Background(), "Neuer Start")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for range ch {
	}

	got := fp.requests[0].Messages
	if len(got) != 1 || got[0].Content != "Neuer Start" {
		t.Fatalf("cleared history should only include new input: %#v", got)
	}
}
