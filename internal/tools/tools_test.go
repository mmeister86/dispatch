package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubRecentCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("authorization") != "Bearer gh-token" {
			t.Fatalf("authorization = %q", r.Header.Get("authorization"))
		}
		switch {
		case r.URL.Path == "/user/repos":
			_, _ = w.Write([]byte(`[{"name":"dispatch","full_name":"matthias/dispatch"}]`))
		case r.URL.Path == "/repos/matthias/dispatch/commits":
			_, _ = w.Write([]byte(`[{"sha":"abcdef123","commit":{"message":"feat: ship tools","author":{"date":"2026-05-07T12:00:00Z"}}}]`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	tool := NewGitHub("gh-token", server.URL)
	result, err := tool.Execute(context.Background(), "get_recent_commits", json.RawMessage(`{"repos":["all"],"since":"7d","limit":1}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "matthias/dispatch") || !strings.Contains(result, "feat: ship tools") {
		t.Fatalf("result = %s", result)
	}
}

func TestGitHubRecentCommitsNotesEmptyRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("authorization") != "Bearer gh-token" {
			t.Fatalf("authorization = %q", r.Header.Get("authorization"))
		}
		switch {
		case r.URL.Path == "/user/repos":
			_, _ = w.Write([]byte(`[
				{"name":"dispatch","full_name":"matthias/dispatch"},
				{"name":"empty","full_name":"matthias/empty"}
			]`))
		case r.URL.Path == "/repos/matthias/dispatch/commits":
			_, _ = w.Write([]byte(`[{"sha":"abcdef123","commit":{"message":"feat: ship tools","author":{"date":"2026-05-07T12:00:00Z"}}}]`))
		case r.URL.Path == "/repos/matthias/empty/commits":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message":"Git Repository is empty.","documentation_url":"https://docs.github.com/rest/commits/commits#list-commits","status":"409"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	tool := NewGitHub("gh-token", server.URL)
	result, err := tool.Execute(context.Background(), "get_recent_commits", json.RawMessage(`{"repos":["all"],"since":"7d","limit":1}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "matthias/dispatch") || !strings.Contains(result, "feat: ship tools") {
		t.Fatalf("result = %s", result)
	}
	if !strings.Contains(result, "matthias/empty") || !strings.Contains(result, "Keine Commits vorhanden") {
		t.Fatalf("result = %s", result)
	}
}

func TestGitHubRecentCommitsPreservesPermissionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user/repos":
			_, _ = w.Write([]byte(`[{"name":"private","full_name":"matthias/private"}]`))
		case r.URL.Path == "/repos/matthias/private/commits":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"Resource not accessible by personal access token","status":"403"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	tool := NewGitHub("gh-token", server.URL)
	_, err := tool.Execute(context.Background(), "get_recent_commits", json.RawMessage(`{"repos":["all"],"since":"7d","limit":1}`))
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !strings.Contains(err.Error(), "403 Forbidden") {
		t.Fatalf("err = %v", err)
	}
}

func TestGitHubGetFileDecodesBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/matthias/dispatch/contents/README.md" {
			encoded := base64.StdEncoding.EncodeToString([]byte("# dispatch"))
			_, _ = w.Write([]byte(`{"encoding":"base64","content":"` + encoded + `"}`))
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.String())
	}))
	defer server.Close()

	tool := NewGitHub("gh-token", server.URL)
	result, err := tool.Execute(context.Background(), "get_file", json.RawMessage(`{"repo":"matthias/dispatch","path":"README.md"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "# dispatch" {
		t.Fatalf("result = %q", result)
	}
}

func TestSearchPerplexityFormatsCitations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("authorization") != "Bearer search-key" {
			t.Fatalf("authorization = %q", r.Header.Get("authorization"))
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Antwort"}}],"citations":["https://example.com/a"]}`))
	}))
	defer server.Close()

	tool := NewSearch("search-key", "perplexity", "sonar-pro", "week", server.URL)
	result, err := tool.Execute(context.Background(), "search", json.RawMessage(`{"query":"Go news"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "Antwort") || !strings.Contains(result, "[1] https://example.com/a") {
		t.Fatalf("result = %s", result)
	}
}

func TestPostizCreatePost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/posts" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("authorization") != "Bearer postiz-key" {
			t.Fatalf("authorization = %q", r.Header.Get("authorization"))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"post-1","status":"scheduled"}`))
	}))
	defer server.Close()

	tool := NewPostiz("postiz-key", server.URL)
	result, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"linkedin","content":"Hallo","scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "post-1") {
		t.Fatalf("result = %s", result)
	}
}

func TestPostizCreatePostRequiresConfirmation(t *testing.T) {
	tool := NewPostiz("postiz-key", "https://postiz.example")
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"linkedin","content":"Hallo","scheduled_at":"2026-05-08T09:00:00Z"}`))
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}
