package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matthias/dispatch/internal/provider"
)

func TestStoreSaveLoadRoundTripWithPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dispatch", "sessions", "current.json")
	store := NewStore(path)

	want := Session{
		Messages: []Message{
			{Kind: KindUser, Body: "Plane bitte LinkedIn-Posts."},
			{Kind: KindAgent, Body: "Ich entwerfe einen Plan."},
		},
		AgentHistory: []provider.Message{
			{Role: provider.RoleUser, Content: "Plane bitte LinkedIn-Posts."},
			{Role: provider.RoleAssistant, Content: "Ich entwerfe einen Plan."},
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat session dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("session dir permissions = %v, want 0700", dirInfo.Mode().Perm())
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("session file permissions = %v, want 0600", fileInfo.Mode().Perm())
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("version = %d, want 1", got.Version)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("updated_at should be set")
	}
	if len(got.Messages) != 2 || got.Messages[0].Body != want.Messages[0].Body {
		t.Fatalf("messages roundtrip mismatch: %#v", got.Messages)
	}
	if len(got.AgentHistory) != 2 || got.AgentHistory[1].Content != want.AgentHistory[1].Content {
		t.Fatalf("agent history roundtrip mismatch: %#v", got.AgentHistory)
	}
}

func TestStoreLoadMissingFileReturnsEmptySession(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing", "current.json"))

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("version = %d, want 1", got.Version)
	}
	if len(got.Messages) != 0 || len(got.AgentHistory) != 0 {
		t.Fatalf("missing file should load empty session: %#v", got)
	}
}

func TestStoreLoadCorruptFileReturnsHelpfulError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("write corrupt session: %v", err)
	}

	_, err := NewStore(path).Load()
	if err == nil {
		t.Fatal("expected corrupt session error")
	}
	if !strings.Contains(err.Error(), "load session") {
		t.Fatalf("error should mention loading session: %v", err)
	}
}

func TestStoreClearRemovesSessionFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current.json")
	store := NewStore(path)
	if err := store.Save(Session{Messages: []Message{{Kind: KindUser, Body: "hi"}}}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("session file should be removed, stat err = %v", err)
	}
}

func TestDefaultPathUsesXDGStateHome(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}

	want := filepath.Join(stateHome, "dispatch", "sessions", "current.json")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPathFallsBackToLocalState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}

	want := filepath.Join(home, ".local", "state", "dispatch", "sessions", "current.json")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestStoreListReturnsSessionsSortedByUpdateAndMarksCurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dispatch", "sessions", "current.json")
	store := NewStore(path)
	oldTime := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 5, 8, 13, 0, 0, 0, time.UTC)

	if err := store.Save(Session{
		ID:        "old-plan",
		Title:     "Old plan",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
		Messages:  []Message{{Kind: KindUser, Body: "Old plan"}},
	}); err != nil {
		t.Fatalf("Save old session: %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := store.Save(Session{
		ID:        "new-plan",
		Title:     "New plan",
		CreatedAt: newTime,
		UpdatedAt: newTime,
		Messages: []Message{
			{Kind: KindUser, Body: "New plan"},
			{Kind: KindAgent, Body: "Draft"},
		},
	}); err != nil {
		t.Fatalf("Save new session: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("summary length = %d, want 2: %#v", len(got), got)
	}
	if got[0].ID != "new-plan" || got[1].ID != "old-plan" {
		t.Fatalf("sessions should be sorted newest first: %#v", got)
	}
	if !got[0].Current || got[1].Current {
		t.Fatalf("current marker mismatch: %#v", got)
	}
	if got[0].MessageCount != 2 {
		t.Fatalf("message count = %d, want 2", got[0].MessageCount)
	}
}

func TestStoreStartNewPreservesPreviousSession(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "dispatch", "sessions", "current.json"))
	if err := store.Save(Session{
		ID:       "existing-plan",
		Messages: []Message{{Kind: KindUser, Body: "Existing plan"}},
	}); err != nil {
		t.Fatalf("Save existing session: %v", err)
	}

	started, err := store.StartNew()
	if err != nil {
		t.Fatalf("StartNew returned error: %v", err)
	}
	if started.ID == "" || started.ID == "existing-plan" {
		t.Fatalf("new session should have a fresh id: %#v", started)
	}

	summaries, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("summary length = %d, want 2: %#v", len(summaries), summaries)
	}
	if summaries[0].ID != started.ID || !summaries[0].Current {
		t.Fatalf("new session should be current and newest: %#v", summaries)
	}
}

func TestStoreOpenMakesSessionActive(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "dispatch", "sessions", "current.json"))
	first := Session{
		ID:       "first-plan",
		Title:    "First plan",
		Messages: []Message{{Kind: KindUser, Body: "First plan"}},
		AgentHistory: []provider.Message{
			{Role: provider.RoleUser, Content: "First plan"},
		},
	}
	if err := store.Save(first); err != nil {
		t.Fatalf("Save first session: %v", err)
	}
	if _, err := store.StartNew(); err != nil {
		t.Fatalf("StartNew returned error: %v", err)
	}

	opened, err := store.Open("first-plan")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if opened.ID != "first-plan" || len(opened.AgentHistory) != 1 {
		t.Fatalf("opened session mismatch: %#v", opened)
	}

	active, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if active.ID != "first-plan" || active.Messages[0].Body != "First plan" {
		t.Fatalf("active session mismatch after open: %#v", active)
	}
}
