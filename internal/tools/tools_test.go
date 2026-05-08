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
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/mcp/postiz-key" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("accept"), "application/json") {
			t.Fatalf("accept = %q", r.Header.Get("accept"))
		}
		var req struct {
			Method string `json:"method"`
			Params struct {
				ProtocolVersion string          `json:"protocolVersion"`
				Name            string          `json:"name"`
				Arguments       json.RawMessage `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method == "initialize" {
			if r.Header.Get("MCP-Session-Id") != "" {
				t.Fatalf("initialize session = %q", r.Header.Get("MCP-Session-Id"))
			}
			if r.Header.Get("MCP-Protocol-Version") != "2025-06-18" || req.Params.ProtocolVersion != "2025-06-18" {
				t.Fatalf("protocol version header=%q param=%q", r.Header.Get("MCP-Protocol-Version"), req.Params.ProtocolVersion)
			}
			w.Header().Set("MCP-Session-Id", "session-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
			return
		}
		if r.Header.Get("MCP-Session-Id") != "session-1" {
			t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
		}
		if r.Header.Get("MCP-Protocol-Version") != "2025-06-18" {
			t.Fatalf("protocol version = %q", r.Header.Get("MCP-Protocol-Version"))
		}
		if req.Method != "tools/call" {
			t.Fatalf("method = %q", req.Method)
		}
		calls = append(calls, req.Params.Name)
		switch req.Params.Name {
		case "integrationList":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"linkedin\"}]"}]}}`))
		case "schedulePostTool":
			var args struct {
				SocialPost []struct {
					IntegrationID    string `json:"integrationId"`
					Date             string `json:"date"`
					PostsAndComments []struct {
						Content string `json:"content"`
					} `json:"postsAndComments"`
				} `json:"socialPost"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				t.Fatalf("decode schedule args: %v", err)
			}
			if len(args.SocialPost) != 1 {
				t.Fatalf("socialPost = %#v", args.SocialPost)
			}
			post := args.SocialPost[0]
			if post.IntegrationID != "integration-1" || post.Date != "2026-05-08T09:00:00Z" {
				t.Fatalf("post = %#v", post)
			}
			if len(post.PostsAndComments) != 1 || post.PostsAndComments[0].Content != "<p>Hallo</p>" {
				t.Fatalf("postsAndComments = %#v", post.PostsAndComments)
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"id\":\"post-1\",\"status\":\"scheduled\"}"}]}}`))
		default:
			t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
		}
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
	if strings.Join(calls, ",") != "integrationList,schedulePostTool" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestPostizListChannelsUsesMCPIntegrationList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/mcp/postiz-key" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method == "initialize" {
			w.Header().Set("MCP-Session-Id", "session-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
			return
		}
		if r.Header.Get("MCP-Session-Id") != "session-1" {
			t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
		}
		if req.Method != "tools/call" || req.Params.Name != "integrationList" {
			t.Fatalf("request = %#v", req)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"channel-1\",\"platform\":\"linkedin\"}]"}]}}`))
	}))
	defer server.Close()

	tool := NewPostiz("postiz-key", server.URL)
	result, err := tool.Execute(context.Background(), "list_channels", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "channel-1") {
		t.Fatalf("result = %s", result)
	}
}

func TestPostizListChannelsAcceptsFullMCPEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mcp/postiz-key" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var req struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method == "initialize" {
			w.Header().Set("MCP-Session-Id", "session-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
			return
		}
		if r.Header.Get("MCP-Session-Id") != "session-1" {
			t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"channel-1\",\"platform\":\"linkedin\"}]"}]}}`))
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	result, err := tool.Execute(context.Background(), "list_channels", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "channel-1") {
		t.Fatalf("result = %s", result)
	}
}

func TestPostizListPostsReportsUnsupportedMCPTool(t *testing.T) {
	tool := NewPostiz("postiz-key", "https://postiz.example")
	_, err := tool.Execute(context.Background(), "list_posts", json.RawMessage(`{"status":"scheduled","limit":3}`))
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if !strings.Contains(err.Error(), "Postiz MCP") || !strings.Contains(err.Error(), "list_posts") {
		t.Fatalf("err = %v", err)
	}
}

func TestPostizCreatePostRequiresConfirmation(t *testing.T) {
	tool := NewPostiz("postiz-key", "https://postiz.example")
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"linkedin","content":"Hallo","scheduled_at":"2026-05-08T09:00:00Z"}`))
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}
