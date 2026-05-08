package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
