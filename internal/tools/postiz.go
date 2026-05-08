package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/matthias/dispatch/internal/provider"
)

const mcpProtocolVersion = "2025-06-18"

type Postiz struct {
	apiKey    string
	baseURL   string
	client    *http.Client
	sessionMu sync.Mutex
	sessionID string
}

func NewPostiz(apiKey, baseURL string) *Postiz {
	return &Postiz{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Postiz) Definitions() []provider.ToolDefinition {
	return []provider.ToolDefinition{
		{Name: "create_post", Description: "Plant einen Social-Media-Post via Postiz. Nur nach expliziter User-Bestaetigung nutzen.", Schema: objectSchema(map[string]any{
			"integration_id": stringProp("Postiz Integration-ID aus list_channels. Wenn leer, wird platform zum Suchen genutzt."),
			"platform":       stringProp("Zielplattform, z.B. linkedin oder twitter"),
			"content":        stringProp("Post-Text"),
			"scheduled_at":   stringProp("ISO-8601 Zeitpunkt"),
			"media_urls":     stringArrayProp("Optionale Medien-URLs"),
			"confirmed":      boolProp("Muss true sein, nachdem der User die Vorschau explizit bestaetigt hat"),
		}, "content", "scheduled_at", "confirmed")},
		{Name: "list_channels", Description: "Listet verbundene Postiz-Social-Channels.", Schema: objectSchema(map[string]any{})},
	}
}

func (p *Postiz) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if p.apiKey == "" && !strings.Contains(p.baseURL, "/mcp/") {
		return "", fmt.Errorf("Postiz API Key fehlt")
	}
	switch name {
	case "create_post":
		var req struct {
			IntegrationID string   `json:"integration_id"`
			Platform      string   `json:"platform"`
			Content       string   `json:"content"`
			ScheduledAt   string   `json:"scheduled_at"`
			MediaURLs     []string `json:"media_urls"`
			Confirmed     bool     `json:"confirmed"`
		}
		_ = json.Unmarshal(args, &req)
		if !req.Confirmed {
			return "", fmt.Errorf("Postiz create_post erfordert confirmed=true nach expliziter User-Bestaetigung")
		}
		integrationID := strings.TrimSpace(req.IntegrationID)
		if integrationID == "" {
			var err error
			integrationID, err = p.integrationIDForPlatform(ctx, req.Platform)
			if err != nil {
				return "", err
			}
		}
		payload := map[string]any{
			"socialPost": []map[string]any{{
				"integrationId": integrationID,
				"isPremium":     false,
				"date":          req.ScheduledAt,
				"shortLink":     false,
				"type":          "schedule",
				"postsAndComments": []map[string]any{{
					"content":     htmlContent(req.Content),
					"attachments": req.MediaURLs,
				}},
				"settings": []map[string]any{},
			}},
		}
		return p.callMCPTool(ctx, "schedulePostTool", payload)
	case "list_posts":
		return "", fmt.Errorf("Postiz MCP bietet laut Dokumentation kein list_posts Tool an")
	case "list_channels":
		return p.callMCPTool(ctx, "integrationList", map[string]any{})
	default:
		return "", fmt.Errorf("Postiz Tool %q unbekannt", name)
	}
}

func (p *Postiz) callMCPTool(ctx context.Context, toolName string, arguments any) (string, error) {
	endpoint, err := p.mcpEndpoint()
	if err != nil {
		return "", err
	}
	sessionID, err := p.ensureMCPSession(ctx, endpoint)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": arguments,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", mcpProtocolVersion)
	if sessionID != "" {
		req.Header.Set("MCP-Session-Id", sessionID)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz MCP returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return parseMCPToolResult(respBody)
}

func (p *Postiz) ensureMCPSession(ctx context.Context, endpoint string) (string, error) {
	p.sessionMu.Lock()
	defer p.sessionMu.Unlock()
	if p.sessionID != "" {
		return p.sessionID, nil
	}

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "dispatch",
				"version": "dev",
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", mcpProtocolVersion)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz MCP initialize returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if _, err := parseMCPToolResult(respBody); err != nil {
		return "", err
	}
	p.sessionID = resp.Header.Get("MCP-Session-Id")
	return p.sessionID, nil
}

func (p *Postiz) mcpEndpoint() (string, error) {
	if strings.TrimSpace(p.baseURL) == "" {
		return "", fmt.Errorf("Postiz Base URL fehlt")
	}
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return "", err
	}
	if strings.Contains(u.Path, "/mcp/") {
		u.RawQuery = ""
		return strings.TrimRight(u.String(), "/"), nil
	}
	key := url.PathEscape(p.apiKey)
	path := strings.TrimRight(u.Path, "/")
	switch {
	case strings.EqualFold(u.Host, "api.postiz.com"):
		u.Path = path + "/mcp/" + key
	case strings.HasSuffix(path, "/api"):
		u.Path = path + "/mcp/" + key
	default:
		u.Path = path + "/api/mcp/" + key
	}
	u.RawQuery = ""
	return u.String(), nil
}

func parseMCPToolResult(body []byte) (string, error) {
	body = bytes.TrimSpace(body)
	if bytes.HasPrefix(body, []byte("event:")) || bytes.HasPrefix(body, []byte("data:")) {
		body = lastSSEData(body)
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("Postiz MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(resp.Result, &result); err == nil {
		if len(result.StructuredContent) > 0 {
			return string(result.StructuredContent), nil
		}
		for _, item := range result.Content {
			if item.Text != "" {
				return item.Text, nil
			}
		}
	}
	return string(resp.Result), nil
}

func lastSSEData(body []byte) []byte {
	var data []byte
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if value != "" && value != "[DONE]" {
			data = []byte(value)
		}
	}
	return data
}

func (p *Postiz) integrationIDForPlatform(ctx context.Context, platform string) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" {
		return "", fmt.Errorf("Postiz create_post braucht integration_id oder platform")
	}
	result, err := p.callMCPTool(ctx, "integrationList", map[string]any{})
	if err != nil {
		return "", err
	}
	var integrations []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Platform string `json:"platform"`
	}
	if err := json.Unmarshal([]byte(result), &integrations); err != nil {
		return "", fmt.Errorf("Postiz integrationList konnte nicht gelesen werden: %w", err)
	}
	var matches []string
	for _, integration := range integrations {
		if strings.EqualFold(integration.ID, platform) ||
			strings.EqualFold(integration.Platform, platform) ||
			strings.EqualFold(integration.Name, platform) {
			matches = append(matches, integration.ID)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("keine Postiz Integration fuer %q gefunden; nutze list_channels und uebergib integration_id", platform)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("mehrere Postiz Integrationen fuer %q gefunden; nutze list_channels und uebergib integration_id", platform)
	}
	return matches[0], nil
}

func htmlContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return trimmed
	}
	return "<p>" + html.EscapeString(trimmed) + "</p>"
}
