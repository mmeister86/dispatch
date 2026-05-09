package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
			if r.Header.Get("MCP-Protocol-Version") != mcpProtocolVersions[0] || req.Params.ProtocolVersion != mcpProtocolVersions[0] {
				t.Fatalf("protocol version header=%q param=%q", r.Header.Get("MCP-Protocol-Version"), req.Params.ProtocolVersion)
			}
			w.Header().Set("MCP-Session-Id", "session-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"` + mcpProtocolVersions[0] + `","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
			return
		}
		if r.Header.Get("MCP-Session-Id") != "session-1" {
			t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
		}
		if r.Header.Get("MCP-Protocol-Version") != mcpProtocolVersions[0] {
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

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	tool.now = func() time.Time { return time.Date(2026, 5, 7, 20, 0, 0, 0, time.UTC) }
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

func TestPostizCreatePostSchemaSupportsThreadParts(t *testing.T) {
	tool := NewPostiz("postiz-key", "https://postiz.example")
	var createPostSchema map[string]any
	for _, definition := range tool.Definitions() {
		if definition.Name == "create_post" {
			createPostSchema = definition.Schema
			break
		}
	}
	if createPostSchema == nil {
		t.Fatal("create_post definition not found")
	}
	properties, ok := createPostSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %#v", createPostSchema["properties"])
	}
	threadParts, ok := properties["thread_parts"].(map[string]any)
	if !ok {
		t.Fatalf("thread_parts property missing from schema: %#v", properties)
	}
	if threadParts["type"] != "array" {
		t.Fatalf("thread_parts type = %#v", threadParts["type"])
	}
	items, ok := threadParts["items"].(map[string]any)
	if !ok || items["type"] != "string" {
		t.Fatalf("thread_parts items = %#v", threadParts["items"])
	}
	required, ok := createPostSchema["required"].([]string)
	if !ok {
		t.Fatalf("required = %#v", createPostSchema["required"])
	}
	for _, field := range required {
		if field == "content" {
			t.Fatalf("content should not be required when thread_parts is available: %#v", required)
		}
	}
	if !stringSliceContains(required, "scheduled_at") || !stringSliceContains(required, "confirmed") {
		t.Fatalf("required = %#v", required)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestPostizCreatePostUsesExplicitThreadPartsForMCPThread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
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
		switch req.Params.Name {
		case "integrationList":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"x\"}]"}]}}`))
		case "schedulePostTool":
			var args struct {
				SocialPost []struct {
					PostsAndComments []struct {
						Content     string   `json:"content"`
						Attachments []string `json:"attachments"`
					} `json:"postsAndComments"`
				} `json:"socialPost"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				t.Fatalf("decode schedule args: %v", err)
			}
			if len(args.SocialPost) != 1 {
				t.Fatalf("socialPost = %#v", args.SocialPost)
			}
			items := args.SocialPost[0].PostsAndComments
			if len(items) != 3 {
				t.Fatalf("postsAndComments = %#v", items)
			}
			wantContent := []string{"<p>First tweet</p>", "<p>Second tweet</p>", "<p>Final tweet</p>"}
			for i, item := range items {
				if item.Content != wantContent[i] {
					t.Fatalf("item %d content = %q", i, item.Content)
				}
				if i == 0 {
					if len(item.Attachments) != 1 || item.Attachments[0] != "https://example.com/image.png" {
						t.Fatalf("first attachments = %#v", item.Attachments)
					}
					continue
				}
				if len(item.Attachments) != 0 {
					t.Fatalf("item %d attachments = %#v", i, item.Attachments)
				}
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"id\":\"post-1\",\"status\":\"scheduled\"}"}]}}`))
		default:
			t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	result, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"x","thread_parts":[" First tweet ","Second tweet","Final tweet"],"scheduled_at":"2026-05-08T09:00:00Z","media_urls":["https://example.com/image.png"],"confirmed":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "post-1") {
		t.Fatalf("result = %s", result)
	}
}

func TestPostizCreatePostSplitsLongXContentIntoMCPThread(t *testing.T) {
	longContent := strings.Repeat("alpha ", 60)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
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
		switch req.Params.Name {
		case "integrationList":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"x\"}]"}]}}`))
		case "schedulePostTool":
			var args struct {
				SocialPost []struct {
					PostsAndComments []struct {
						Content string `json:"content"`
					} `json:"postsAndComments"`
					Settings []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					} `json:"settings"`
				} `json:"socialPost"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				t.Fatalf("decode schedule args: %v", err)
			}
			if len(args.SocialPost) != 1 || len(args.SocialPost[0].PostsAndComments) < 2 {
				t.Fatalf("expected thread posts, got %#v", args.SocialPost)
			}
			for _, item := range args.SocialPost[0].PostsAndComments {
				text := strings.TrimSuffix(strings.TrimPrefix(item.Content, "<p>"), "</p>")
				if len([]rune(text)) > 280 {
					t.Fatalf("thread item too long: %d chars in %q", len([]rune(text)), text)
				}
			}
			if len(args.SocialPost[0].Settings) != 1 || args.SocialPost[0].Settings[0].Key != "who_can_reply_post" || args.SocialPost[0].Settings[0].Value != "everyone" {
				t.Fatalf("settings = %#v", args.SocialPost[0].Settings)
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"id\":\"post-1\",\"status\":\"scheduled\"}"}]}}`))
		default:
			t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(fmt.Sprintf(`{"platform":"x","content":%q,"scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`, longContent)))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestPostizCreatePostPublishesNowWhenScheduledAtIsPastForMCP(t *testing.T) {
	now := time.Date(2026, 5, 8, 20, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
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
		switch req.Params.Name {
		case "integrationList":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"x\"}]"}]}}`))
		case "schedulePostTool":
			var args struct {
				SocialPost []struct {
					Type string `json:"type"`
					Date string `json:"date"`
				} `json:"socialPost"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				t.Fatalf("decode schedule args: %v", err)
			}
			if len(args.SocialPost) != 1 || args.SocialPost[0].Type != "now" || args.SocialPost[0].Date != now.Format(time.RFC3339) {
				t.Fatalf("socialPost = %#v", args.SocialPost)
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"id\":\"post-1\",\"status\":\"published\"}"}]}}`))
		default:
			t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	tool.now = func() time.Time { return now }
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"x","content":"Hallo","scheduled_at":"2000-01-01T00:00:00Z","confirmed":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestPostizCreatePostFallsBackToPublicAPIWhenMCPScheduleToolIsMissing(t *testing.T) {
	var calls []string
	var publicCalled bool
	longContent := strings.Repeat("alpha ", 60)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mcp/postiz-key":
			var req struct {
				Method string `json:"method"`
				Params struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				} `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode MCP request: %v", err)
			}
			if req.Method == "initialize" {
				w.Header().Set("MCP-Session-Id", "session-1")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
				return
			}
			if r.Header.Get("MCP-Session-Id") != "session-1" {
				t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
			}
			calls = append(calls, req.Params.Name)
			switch req.Params.Name {
			case "integrationList":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"x\"}]"}]}}`))
			case "schedulePostTool":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"Unknown tool: schedulePostTool"}}`))
			default:
				t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
			}
		case "/api/public/v1/posts":
			publicCalled = true
			if r.Method != http.MethodPost {
				t.Fatalf("public method = %s", r.Method)
			}
			if r.Header.Get("authorization") != "postiz-key" {
				t.Fatalf("authorization = %q", r.Header.Get("authorization"))
			}
			var req struct {
				Type      string `json:"type"`
				Date      string `json:"date"`
				ShortLink bool   `json:"shortLink"`
				Posts     []struct {
					Integration struct {
						ID string `json:"id"`
					} `json:"integration"`
					Value []struct {
						Content string `json:"content"`
						Image   []any  `json:"image"`
					} `json:"value"`
					Settings struct {
						Type            string `json:"__type"`
						WhoCanReplyPost string `json:"who_can_reply_post"`
					} `json:"settings"`
				} `json:"posts"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode public request: %v", err)
			}
			if req.Type != "schedule" || req.Date != "2026-05-08T09:00:00Z" || req.ShortLink {
				t.Fatalf("public schedule fields = %#v", req)
			}
			if len(req.Posts) != 1 || req.Posts[0].Integration.ID != "integration-1" {
				t.Fatalf("public posts = %#v", req.Posts)
			}
			if len(req.Posts[0].Value) < 2 {
				t.Fatalf("public value = %#v", req.Posts[0].Value)
			}
			for _, value := range req.Posts[0].Value {
				if len([]rune(value.Content)) > 280 {
					t.Fatalf("thread item too long: %d chars in %q", len([]rune(value.Content)), value.Content)
				}
				if value.Image == nil {
					t.Fatalf("public image must be an empty array, got nil")
				}
			}
			if req.Posts[0].Settings.Type != "x" || req.Posts[0].Settings.WhoCanReplyPost != "everyone" {
				t.Fatalf("public settings = %#v", req.Posts[0].Settings)
			}
			_, _ = w.Write([]byte(`{"id":"post-public-1","state":"QUEUE"}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	tool.now = func() time.Time { return time.Date(2026, 5, 7, 20, 0, 0, 0, time.UTC) }
	result, err := tool.Execute(context.Background(), "create_post", json.RawMessage(fmt.Sprintf(`{"platform":"x","content":%q,"scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`, longContent)))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !publicCalled {
		t.Fatal("public API fallback was not called")
	}
	if !strings.Contains(result, "post-public-1") {
		t.Fatalf("result = %s", result)
	}
	if strings.Join(calls, ",") != "integrationList,schedulePostTool" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestPostizCreatePostUsesExplicitThreadPartsForPublicAPIThreadFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mcp/postiz-key":
			var req struct {
				Method string `json:"method"`
				Params struct {
					Name string `json:"name"`
				} `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode MCP request: %v", err)
			}
			if req.Method == "initialize" {
				w.Header().Set("MCP-Session-Id", "session-1")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
				return
			}
			switch req.Params.Name {
			case "integrationList":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"x\"}]"}]}}`))
			case "schedulePostTool":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"Unknown tool: schedulePostTool"}}`))
			default:
				t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
			}
		case "/api/public/v1/posts":
			var req struct {
				Posts []struct {
					Value []struct {
						Content string `json:"content"`
						Image   []any  `json:"image"`
					} `json:"value"`
					Settings struct {
						Type            string `json:"__type"`
						WhoCanReplyPost string `json:"who_can_reply_post"`
					} `json:"settings"`
				} `json:"posts"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode public request: %v", err)
			}
			if len(req.Posts) != 1 {
				t.Fatalf("posts = %#v", req.Posts)
			}
			values := req.Posts[0].Value
			if len(values) != 3 {
				t.Fatalf("value = %#v", values)
			}
			wantContent := []string{"First tweet", "Second tweet", "Final tweet"}
			for i, value := range values {
				if value.Content != wantContent[i] {
					t.Fatalf("value %d content = %q", i, value.Content)
				}
				if value.Image == nil {
					t.Fatalf("value %d image must be an array", i)
				}
				if i == 0 && len(value.Image) != 1 {
					t.Fatalf("first value image = %#v", value.Image)
				}
				if i > 0 && len(value.Image) != 0 {
					t.Fatalf("value %d image = %#v", i, value.Image)
				}
			}
			if req.Posts[0].Settings.Type != "x" || req.Posts[0].Settings.WhoCanReplyPost != "everyone" {
				t.Fatalf("settings = %#v", req.Posts[0].Settings)
			}
			_, _ = w.Write([]byte(`{"id":"post-public-1","state":"QUEUE"}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	result, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"twitter","thread_parts":["First tweet","Second tweet"," Final tweet "],"scheduled_at":"2026-05-08T09:00:00Z","media_urls":["https://example.com/image.png"],"confirmed":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "post-public-1") {
		t.Fatalf("result = %s", result)
	}
}

func TestPostizCreatePostRejectsOverlongExplicitThreadPart(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("Postiz should not be called for invalid thread_parts")
	}))
	defer server.Close()

	tool := NewPostiz("postiz-key", server.URL)
	overLimit := strings.Repeat("a", xPostCharLimit+1)
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(fmt.Sprintf(`{"integration_id":"integration-1","platform":"x","thread_parts":["%s"],"scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`, overLimit)))
	if err == nil {
		t.Fatal("expected overlong thread part error")
	}
	if !strings.Contains(err.Error(), "thread_parts[0]") || !strings.Contains(err.Error(), "280") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("Postiz was called")
	}
}

func TestPostizCreatePostRejectsThreadPartsForNonXPlatform(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("Postiz should not be called for unsupported thread_parts platform")
	}))
	defer server.Close()

	tool := NewPostiz("postiz-key", server.URL)
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"integration_id":"integration-1","platform":"linkedin","thread_parts":["First","Second"],"scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`))
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}
	if !strings.Contains(err.Error(), "thread_parts") || !strings.Contains(err.Error(), "x/twitter") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("Postiz was called")
	}
}

func TestPostizCreatePostRejectsBlankExplicitThreadPart(t *testing.T) {
	tool := NewPostiz("postiz-key", "https://postiz.example")
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"integration_id":"integration-1","platform":"x","thread_parts":["First","   "],"scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`))
	if err == nil {
		t.Fatal("expected blank thread part error")
	}
	if !strings.Contains(err.Error(), "thread_parts[1]") || !strings.Contains(err.Error(), "leer") {
		t.Fatalf("err = %v", err)
	}
}

func TestPostizCreatePostPublishesNowWhenScheduledAtIsPastForPublicFallback(t *testing.T) {
	now := time.Date(2026, 5, 8, 20, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mcp/postiz-key":
			var req struct {
				Method string `json:"method"`
				Params struct {
					Name string `json:"name"`
				} `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode MCP request: %v", err)
			}
			if req.Method == "initialize" {
				w.Header().Set("MCP-Session-Id", "session-1")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
				return
			}
			switch req.Params.Name {
			case "integrationList":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"x\"}]"}]}}`))
			case "schedulePostTool":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"Unknown tool: schedulePostTool"}}`))
			default:
				t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
			}
		case "/api/public/v1/posts":
			var req struct {
				Type string `json:"type"`
				Date string `json:"date"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode public request: %v", err)
			}
			if req.Type != "now" || req.Date != now.Format(time.RFC3339) {
				t.Fatalf("public timing = %#v", req)
			}
			_, _ = w.Write([]byte(`{"id":"post-public-1","state":"PUBLISHED"}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	tool.now = func() time.Time { return now }
	_, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"x","content":"Hallo","scheduled_at":"2000-01-01T00:00:00Z","confirmed":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestPostizPublicAPIEndpointKeepsAPIBaseForSelfHostedMCPURL(t *testing.T) {
	tool := NewPostiz("", "https://postiz.example/api/mcp/postiz-key")
	endpoint, err := tool.publicAPIEndpoint()
	if err != nil {
		t.Fatalf("publicAPIEndpoint returned error: %v", err)
	}
	if endpoint != "https://postiz.example/api/public/v1/posts" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestPostizCreatePostFallsBackWhenMCPScheduleToolReturnsToolErrorResult(t *testing.T) {
	var publicCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mcp/postiz-key":
			var req struct {
				Method string `json:"method"`
				Params struct {
					Name string `json:"name"`
				} `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode MCP request: %v", err)
			}
			if req.Method == "initialize" {
				w.Header().Set("MCP-Session-Id", "session-1")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
				return
			}
			switch req.Params.Name {
			case "integrationList":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"integration-1\",\"name\":\"Work\",\"platform\":\"linkedin\"}]"}]}}`))
			case "schedulePostTool":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"isError":true,"content":[{"type":"text","text":"Unknown tool: schedulePostTool"}]}}`))
			default:
				t.Fatalf("unexpected MCP tool: %s", req.Params.Name)
			}
		case "/api/public/v1/posts":
			publicCalled = true
			_, _ = w.Write([]byte(`{"id":"post-public-1","state":"QUEUE"}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tool := NewPostiz("", server.URL+"/api/mcp/postiz-key")
	result, err := tool.Execute(context.Background(), "create_post", json.RawMessage(`{"platform":"linkedin","content":"Hallo","scheduled_at":"2026-05-08T09:00:00Z","confirmed":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !publicCalled {
		t.Fatal("public API fallback was not called")
	}
	if !strings.Contains(result, "post-public-1") {
		t.Fatalf("result = %s", result)
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

func TestPostizListChannelsNegotiatesSupportedMCPVersion(t *testing.T) {
	var initializeVersions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mcp/postiz-key" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var req struct {
			Method string `json:"method"`
			Params struct {
				ProtocolVersion string `json:"protocolVersion"`
				Name            string `json:"name"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		switch req.Method {
		case "initialize":
			initializeVersions = append(initializeVersions, req.Params.ProtocolVersion)
			switch req.Params.ProtocolVersion {
			case "2025-11-25":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"Bad Request: Unsupported protocol version (supported versions: 2025-06-18, 2025-03-26, 2024-11-05, 2024-10-07)"},"id":1}`))
			case "2025-06-18":
				w.Header().Set("MCP-Session-Id", "session-1")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
			default:
				t.Fatalf("unexpected protocol version: %s", req.Params.ProtocolVersion)
			}
		case "tools/call":
			if r.Header.Get("MCP-Session-Id") != "session-1" {
				t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
			}
			if r.Header.Get("MCP-Protocol-Version") != "2025-06-18" {
				t.Fatalf("tool protocol version = %q", r.Header.Get("MCP-Protocol-Version"))
			}
			if req.Params.Name != "integrationList" {
				t.Fatalf("tool = %q", req.Params.Name)
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"channel-1\",\"platform\":\"x\"}]"}]}}`))
		default:
			t.Fatalf("method = %q", req.Method)
		}
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
	if strings.Join(initializeVersions, ",") != "2025-11-25,2025-06-18" {
		t.Fatalf("initialize versions = %v", initializeVersions)
	}
}

func TestPostizListChannelsReinitializesAfterExpiredSession(t *testing.T) {
	var initializeCount int
	var toolSessions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		switch req.Method {
		case "initialize":
			initializeCount++
			w.Header().Set("MCP-Session-Id", fmt.Sprintf("session-%d", initializeCount))
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"` + mcpProtocolVersions[0] + `","capabilities":{},"serverInfo":{"name":"postiz","version":"test"}}}`))
		case "tools/call":
			toolSessions = append(toolSessions, r.Header.Get("MCP-Session-Id"))
			if len(toolSessions) == 1 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`session expired`))
				return
			}
			if r.Header.Get("MCP-Session-Id") != "session-2" {
				t.Fatalf("session = %q", r.Header.Get("MCP-Session-Id"))
			}
			if req.Params.Name != "integrationList" {
				t.Fatalf("tool = %q", req.Params.Name)
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"[{\"id\":\"channel-1\",\"platform\":\"x\"}]"}]}}`))
		default:
			t.Fatalf("method = %q", req.Method)
		}
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
	if initializeCount != 2 {
		t.Fatalf("initializeCount = %d", initializeCount)
	}
	if strings.Join(toolSessions, ",") != "session-1,session-2" {
		t.Fatalf("toolSessions = %v", toolSessions)
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
