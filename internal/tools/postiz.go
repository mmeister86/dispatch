package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matthias/dispatch/internal/provider"
)

var mcpProtocolVersions = []string{
	"2025-11-25",
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
	"2024-10-07",
}

const xPostCharLimit = 280

type Postiz struct {
	apiKey    string
	baseURL   string
	client    *http.Client
	now       func() time.Time
	sessionMu sync.Mutex
	sessionID string
	protocol  string
}

func NewPostiz(apiKey, baseURL string) *Postiz {
	return &Postiz{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (p *Postiz) Definitions() []provider.ToolDefinition {
	return []provider.ToolDefinition{
		{Name: "create_post", Description: "Plant einen Social-Media-Post via Postiz. Nur nach expliziter User-Bestaetigung nutzen.", Schema: objectSchema(map[string]any{
			"integration_id": stringProp("Postiz Integration-ID aus list_channels. Wenn leer, wird platform zum Suchen genutzt."),
			"platform":       stringProp("Zielplattform, z.B. linkedin oder twitter"),
			"content":        stringProp("Post-Text"),
			"thread_parts":   stringArrayProp("Optionale X/Twitter-Thread-Tweets; jeder Eintrag wird ein Tweet im selben Thread"),
			"scheduled_at":   stringProp("ISO-8601 Zeitpunkt"),
			"media":          mediaArrayProp("Optionale hochgeladene Medien aus upload_media mit id und path"),
			"media_urls":     stringArrayProp("Optionale Legacy-Medien-URLs; nach upload_media bevorzugt media mit id und path nutzen"),
			"confirmed":      boolProp("Muss true sein, nachdem der User die Vorschau explizit bestaetigt hat"),
		}, "scheduled_at", "confirmed")},
		{Name: "list_channels", Description: "Listet verbundene Postiz-Social-Channels.", Schema: objectSchema(map[string]any{})},
		{Name: "list_posts", Description: "Listet bestehende Postiz-Posts via Public API. Standard: letzte 30 Tage bis naechste 30 Tage.", Schema: objectSchema(map[string]any{
			"start_date": stringProp("Optionaler Startzeitpunkt als ISO-8601; Standard ist jetzt minus 30 Tage"),
			"end_date":   stringProp("Optionaler Endzeitpunkt als ISO-8601; Standard ist jetzt plus 30 Tage"),
			"customer":   stringProp("Optionaler Postiz Customer-ID Filter"),
		})},
		{Name: "delete_post", Description: "Loescht einen Postiz-Post via Public API. Nur nach expliziter User-Bestaetigung nutzen.", Schema: objectSchema(map[string]any{
			"post_id":   stringProp("Postiz Post-ID"),
			"confirmed": boolProp("Muss true sein, nachdem der User das Loeschen explizit bestaetigt hat"),
		}, "post_id", "confirmed")},
		{Name: "set_post_status", Description: "Setzt einen Postiz-Post auf draft oder schedule. Nur nach expliziter User-Bestaetigung nutzen.", Schema: objectSchema(map[string]any{
			"post_id":   stringProp("Postiz Post-ID"),
			"status":    stringProp("Neuer Status: draft oder schedule"),
			"confirmed": boolProp("Muss true sein, nachdem der User die Statusaenderung explizit bestaetigt hat"),
		}, "post_id", "status", "confirmed")},
		{Name: "upload_media", Description: "Laedt eine lokale Datei zu Postiz hoch und gibt die Upload-Antwort zurueck.", Schema: objectSchema(map[string]any{
			"file_path": stringProp("Lokaler Pfad zur hochzuladenden Datei"),
		}, "file_path")},
		{Name: "get_platform_analytics", Description: "Ruft Postiz Analytics fuer eine Integration bzw. Plattform ab.", Schema: objectSchema(map[string]any{
			"integration_id": stringProp("Postiz Integration-ID"),
			"days":           intProp("Optionaler Rueckblick in Tagen; Standard 7"),
		}, "integration_id")},
		{Name: "get_post_analytics", Description: "Ruft Postiz Analytics fuer einen bestimmten Post ab.", Schema: objectSchema(map[string]any{
			"post_id": stringProp("Postiz Post-ID"),
			"days":    intProp("Optionaler Rueckblick in Tagen; Standard 7"),
		}, "post_id")},
		{Name: "list_missing_post_content", Description: "Listet verfuegbare Provider-Inhalte fuer einen Post mit fehlender Release-ID.", Schema: objectSchema(map[string]any{
			"post_id": stringProp("Postiz Post-ID"),
		}, "post_id")},
		{Name: "connect_post_release", Description: "Verbindet einen Postiz-Post mit einer Provider Release-ID. Nur nach expliziter User-Bestaetigung nutzen.", Schema: objectSchema(map[string]any{
			"post_id":    stringProp("Postiz Post-ID"),
			"release_id": stringProp("Provider-spezifische Release-/Content-ID"),
			"confirmed":  boolProp("Muss true sein, nachdem der User die Verbindung explizit bestaetigt hat"),
		}, "post_id", "release_id", "confirmed")},
	}
}

func (p *Postiz) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if p.apiKey == "" && !strings.Contains(p.baseURL, "/mcp/") {
		return "", fmt.Errorf("Postiz API Key fehlt")
	}
	switch name {
	case "create_post":
		var req struct {
			IntegrationID string        `json:"integration_id"`
			Platform      string        `json:"platform"`
			Content       string        `json:"content"`
			ThreadParts   []string      `json:"thread_parts"`
			ScheduledAt   string        `json:"scheduled_at"`
			Media         []postizMedia `json:"media"`
			MediaURLs     []string      `json:"media_urls"`
			Confirmed     bool          `json:"confirmed"`
		}
		_ = json.Unmarshal(args, &req)
		if !req.Confirmed {
			return "", fmt.Errorf("Postiz create_post erfordert confirmed=true nach expliziter User-Bestaetigung")
		}
		parts, err := createPostParts(req.Platform, req.Content, req.ThreadParts)
		if err != nil {
			return "", err
		}
		integrationID := strings.TrimSpace(req.IntegrationID)
		if integrationID == "" {
			integrationID, err = p.integrationIDForPlatform(ctx, req.Platform)
			if err != nil {
				return "", err
			}
		}
		postType, postDate, err := p.postTiming(req.ScheduledAt)
		if err != nil {
			return "", err
		}
		media := normalizePostizMedia(req.Media, req.MediaURLs)
		payload := map[string]any{
			"socialPost": []map[string]any{{
				"integrationId":    integrationID,
				"isPremium":        false,
				"date":             postDate,
				"shortLink":        false,
				"type":             postType,
				"postsAndComments": mcpPostsAndComments(parts, mediaPaths(media)),
				"settings":         mcpPostSettings(req.Platform),
			}},
		}
		result, err := p.callMCPTool(ctx, "schedulePostTool", payload)
		if err != nil && isUnknownMCPTool(err, "schedulePostTool") {
			return p.createPostPublic(ctx, integrationID, req.Platform, postType, postDate, parts, media)
		}
		return result, err
	case "list_posts":
		var req struct {
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
			Customer  string `json:"customer"`
		}
		_ = json.Unmarshal(args, &req)
		return p.listPostsPublic(ctx, req.StartDate, req.EndDate, req.Customer)
	case "delete_post":
		var req struct {
			PostID    string `json:"post_id"`
			Confirmed bool   `json:"confirmed"`
		}
		_ = json.Unmarshal(args, &req)
		if !req.Confirmed {
			return "", fmt.Errorf("Postiz delete_post erfordert confirmed=true nach expliziter User-Bestaetigung")
		}
		return p.deletePostPublic(ctx, req.PostID)
	case "set_post_status":
		var req struct {
			PostID    string `json:"post_id"`
			Status    string `json:"status"`
			Confirmed bool   `json:"confirmed"`
		}
		_ = json.Unmarshal(args, &req)
		if !req.Confirmed {
			return "", fmt.Errorf("Postiz set_post_status erfordert confirmed=true nach expliziter User-Bestaetigung")
		}
		return p.setPostStatusPublic(ctx, req.PostID, req.Status)
	case "upload_media":
		var req struct {
			FilePath string `json:"file_path"`
		}
		_ = json.Unmarshal(args, &req)
		return p.uploadMediaPublic(ctx, req.FilePath)
	case "get_platform_analytics":
		var req struct {
			IntegrationID string `json:"integration_id"`
			Days          int    `json:"days"`
		}
		_ = json.Unmarshal(args, &req)
		return p.analyticsPublic(ctx, "analytics/"+url.PathEscape(strings.TrimSpace(req.IntegrationID)), req.Days)
	case "get_post_analytics":
		var req struct {
			PostID string `json:"post_id"`
			Days   int    `json:"days"`
		}
		_ = json.Unmarshal(args, &req)
		return p.analyticsPublic(ctx, "analytics/post/"+url.PathEscape(strings.TrimSpace(req.PostID)), req.Days)
	case "list_missing_post_content":
		var req struct {
			PostID string `json:"post_id"`
		}
		_ = json.Unmarshal(args, &req)
		postID := strings.TrimSpace(req.PostID)
		if postID == "" {
			return "", fmt.Errorf("Postiz list_missing_post_content braucht post_id")
		}
		return p.publicAPIRequest(ctx, http.MethodGet, "posts/"+url.PathEscape(postID)+"/missing", nil, nil)
	case "connect_post_release":
		var req struct {
			PostID    string `json:"post_id"`
			ReleaseID string `json:"release_id"`
			Confirmed bool   `json:"confirmed"`
		}
		_ = json.Unmarshal(args, &req)
		if !req.Confirmed {
			return "", fmt.Errorf("Postiz connect_post_release erfordert confirmed=true nach expliziter User-Bestaetigung")
		}
		return p.connectPostReleasePublic(ctx, req.PostID, req.ReleaseID)
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

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		session, err := p.ensureMCPSession(ctx, endpoint)
		if err != nil {
			return "", err
		}
		result, status, bodyText, err := p.postMCPToolCall(ctx, endpoint, body, session)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt == 0 && isExpiredMCPSession(status, bodyText) {
			p.clearMCPSession()
			continue
		}
		return "", err
	}
	return "", lastErr
}

func (p *Postiz) postMCPToolCall(ctx context.Context, endpoint string, body []byte, session mcpSession) (string, int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", session.ProtocolVersion)
	if session.ID != "" {
		req.Header.Set("MCP-Session-Id", session.ID)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	bodyText := strings.TrimSpace(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.StatusCode, bodyText, fmt.Errorf("Postiz MCP returned %s: %s", resp.Status, bodyText)
	}
	result, err := parseMCPToolResult(respBody)
	return result, resp.StatusCode, bodyText, err
}

func isUnknownMCPTool(err error, toolName string) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "Unknown tool") && strings.Contains(message, toolName)
}

func isExpiredMCPSession(status int, body string) bool {
	if status == http.StatusNotFound {
		return true
	}
	return status == http.StatusBadRequest && strings.Contains(body, "No valid session ID")
}

func (p *Postiz) clearMCPSession() {
	p.sessionMu.Lock()
	defer p.sessionMu.Unlock()
	p.sessionID = ""
	p.protocol = ""
}

type mcpSession struct {
	ID              string
	ProtocolVersion string
}

func (p *Postiz) ensureMCPSession(ctx context.Context, endpoint string) (mcpSession, error) {
	p.sessionMu.Lock()
	defer p.sessionMu.Unlock()
	if p.sessionID != "" {
		return mcpSession{ID: p.sessionID, ProtocolVersion: p.protocol}, nil
	}

	var lastErr error
	candidates := append([]string(nil), mcpProtocolVersions...)
	tried := make(map[string]bool)
	for len(candidates) > 0 {
		version := candidates[0]
		candidates = candidates[1:]
		if tried[version] {
			continue
		}
		tried[version] = true
		session, supported, err := p.initializeMCPSession(ctx, endpoint, version)
		if err == nil {
			p.sessionID = session.ID
			p.protocol = session.ProtocolVersion
			return session, nil
		}
		lastErr = err
		if len(supported) > 0 {
			candidates = commonMCPVersions(supported, tried)
		}
	}
	if lastErr != nil {
		return mcpSession{}, lastErr
	}
	return mcpSession{}, fmt.Errorf("Postiz MCP initialize failed: no compatible protocol version")
}

func (p *Postiz) initializeMCPSession(ctx context.Context, endpoint, version string) (mcpSession, []string, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "dispatch",
				"version": "dev",
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return mcpSession{}, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return mcpSession{}, nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", version)
	resp, err := p.client.Do(req)
	if err != nil {
		return mcpSession{}, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcpSession{}, supportedMCPVersions(respBody), fmt.Errorf("Postiz MCP initialize returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	protocol, err := parseMCPInitializeProtocol(respBody)
	if err != nil {
		return mcpSession{}, nil, err
	}
	if protocol == "" {
		protocol = version
	}
	return mcpSession{ID: resp.Header.Get("MCP-Session-Id"), ProtocolVersion: protocol}, nil, nil
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

func (p *Postiz) publicAPIEndpoint() (string, error) {
	return p.publicAPIEndpointFor("posts")
}

func (p *Postiz) publicAPIEndpointFor(endpoint string) (string, error) {
	if strings.TrimSpace(p.baseURL) == "" {
		return "", fmt.Errorf("Postiz Base URL fehlt")
	}
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return "", err
	}
	path := strings.TrimRight(u.Path, "/")
	if idx := strings.Index(path, "/api/mcp/"); idx != -1 {
		path = strings.TrimRight(path[:idx]+"/api", "/")
	} else {
		for _, marker := range []string{"/mcp/"} {
			idx := strings.Index(path, marker)
			if idx == -1 {
				continue
			}
			path = strings.TrimRight(path[:idx], "/")
			break
		}
	}
	endpoint = strings.TrimLeft(endpoint, "/")
	u.Path = path + "/public/v1/" + endpoint
	u.RawQuery = ""
	return u.String(), nil
}

func (p *Postiz) postizAPIKey() (string, error) {
	if strings.TrimSpace(p.apiKey) != "" {
		return p.apiKey, nil
	}
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "mcp" {
			continue
		}
		key, err := url.PathUnescape(parts[i+1])
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(key) != "" {
			return key, nil
		}
	}
	return "", fmt.Errorf("Postiz API Key fehlt")
}

func (p *Postiz) createPostPublic(ctx context.Context, integrationID, platform, postType, postDate string, parts []string, media []postizMedia) (string, error) {
	endpoint, err := p.publicAPIEndpoint()
	if err != nil {
		return "", err
	}
	apiKey, err := p.postizAPIKey()
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"type":      postType,
		"date":      postDate,
		"shortLink": false,
		"tags":      []any{},
		"posts": []map[string]any{{
			"integration": map[string]any{"id": integrationID},
			"value":       publicPostValues(parts, media),
			"settings":    publicPostSettings(platform),
		}},
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
	req.Header.Set("authorization", apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	bodyText := strings.TrimSpace(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz Public API returned %s: %s", resp.Status, bodyText)
	}
	return bodyText, nil
}

func (p *Postiz) listPostsPublic(ctx context.Context, startDate, endDate, customer string) (string, error) {
	now := p.currentTime()
	startDate = strings.TrimSpace(startDate)
	if startDate == "" {
		startDate = now.AddDate(0, 0, -30).Format(time.RFC3339)
	}
	endDate = strings.TrimSpace(endDate)
	if endDate == "" {
		endDate = now.AddDate(0, 0, 30).Format(time.RFC3339)
	}
	query := url.Values{}
	query.Set("startDate", startDate)
	query.Set("endDate", endDate)
	if customer = strings.TrimSpace(customer); customer != "" {
		query.Set("customer", customer)
	}
	return p.publicAPIRequest(ctx, http.MethodGet, "posts", nil, query)
}

func (p *Postiz) deletePostPublic(ctx context.Context, postID string) (string, error) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return "", fmt.Errorf("Postiz delete_post braucht post_id")
	}
	return p.publicAPIRequest(ctx, http.MethodDelete, "posts/"+url.PathEscape(postID), nil, nil)
}

func (p *Postiz) setPostStatusPublic(ctx context.Context, postID, status string) (string, error) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return "", fmt.Errorf("Postiz set_post_status braucht post_id")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "draft" && status != "schedule" {
		return "", fmt.Errorf("Postiz set_post_status status muss draft oder schedule sein")
	}
	return p.publicAPIRequest(ctx, http.MethodPut, "posts/"+url.PathEscape(postID)+"/status", map[string]any{"status": status}, nil)
}

func (p *Postiz) connectPostReleasePublic(ctx context.Context, postID, releaseID string) (string, error) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return "", fmt.Errorf("Postiz connect_post_release braucht post_id")
	}
	releaseID = strings.TrimSpace(releaseID)
	if releaseID == "" {
		return "", fmt.Errorf("Postiz connect_post_release braucht release_id")
	}
	return p.publicAPIRequest(ctx, http.MethodPut, "posts/"+url.PathEscape(postID)+"/release-id", map[string]any{"releaseId": releaseID}, nil)
}

func (p *Postiz) analyticsPublic(ctx context.Context, endpoint string, days int) (string, error) {
	endpoint = strings.Trim(endpoint, "/")
	if endpoint == "analytics" || endpoint == "analytics/post" {
		return "", fmt.Errorf("Postiz analytics braucht eine ID")
	}
	if days <= 0 {
		days = 7
	}
	query := url.Values{}
	query.Set("date", fmt.Sprintf("%d", days))
	return p.publicAPIRequest(ctx, http.MethodGet, endpoint, nil, query)
}

func (p *Postiz) publicAPIRequest(ctx context.Context, method, endpoint string, payload any, query url.Values) (string, error) {
	endpointURL, err := p.publicAPIEndpointFor(endpoint)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(endpointURL)
	if err != nil {
		return "", err
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	apiKey, err := p.postizAPIKey()
	if err != nil {
		return "", err
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", apiKey)
	if payload != nil {
		req.Header.Set("content-type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	bodyText := strings.TrimSpace(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz Public API returned %s: %s", resp.Status, bodyText)
	}
	return bodyText, nil
}

func (p *Postiz) uploadMediaPublic(ctx context.Context, filePath string) (string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", fmt.Errorf("Postiz upload_media braucht file_path")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	endpoint, err := p.publicAPIEndpointFor("upload")
	if err != nil {
		return "", err
	}
	apiKey, err := p.postizAPIKey()
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", apiKey)
	req.Header.Set("content-type", writer.FormDataContentType())
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	bodyText := strings.TrimSpace(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz Public API returned %s: %s", resp.Status, bodyText)
	}
	return bodyText, nil
}

func (p *Postiz) postTiming(scheduledAt string) (string, string, error) {
	scheduledAt = strings.TrimSpace(scheduledAt)
	if scheduledAt == "" {
		return "", "", fmt.Errorf("Postiz create_post braucht scheduled_at")
	}
	scheduled, err := time.Parse(time.RFC3339Nano, scheduledAt)
	if err != nil {
		return "", "", fmt.Errorf("Postiz create_post scheduled_at ist kein gueltiger ISO-8601 Zeitpunkt: %w", err)
	}
	now := p.currentTime()
	if !scheduled.After(now) {
		return "now", now.Format(time.RFC3339), nil
	}
	return "schedule", scheduled.UTC().Format(time.RFC3339), nil
}

func (p *Postiz) currentTime() time.Time {
	if p.now == nil {
		return time.Now().UTC()
	}
	return p.now().UTC()
}

func mcpPostsAndComments(parts []string, mediaURLs []string) []map[string]any {
	items := make([]map[string]any, 0, len(parts))
	for i, part := range parts {
		attachments := []string{}
		if i == 0 {
			attachments = mediaURLs
		}
		items = append(items, map[string]any{
			"content":     htmlContent(part),
			"attachments": attachments,
		})
	}
	return items
}

func mcpPostSettings(platform string) []map[string]any {
	if publicPostPlatform(platform) != "x" {
		return []map[string]any{}
	}
	return []map[string]any{{"key": "who_can_reply_post", "value": "everyone"}}
}

type postizMedia struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func normalizePostizMedia(media []postizMedia, mediaURLs []string) []postizMedia {
	normalized := make([]postizMedia, 0, len(media)+len(mediaURLs))
	for _, item := range media {
		id := strings.TrimSpace(item.ID)
		path := strings.TrimSpace(item.Path)
		if id == "" && path == "" {
			continue
		}
		normalized = append(normalized, postizMedia{ID: id, Path: path})
	}
	for _, mediaURL := range mediaURLs {
		mediaURL = strings.TrimSpace(mediaURL)
		if mediaURL == "" {
			continue
		}
		normalized = append(normalized, postizMedia{Path: mediaURL})
	}
	return normalized
}

func mediaPaths(media []postizMedia) []string {
	paths := make([]string, 0, len(media))
	for _, item := range media {
		path := strings.TrimSpace(item.Path)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func publicPostValues(parts []string, media []postizMedia) []map[string]any {
	values := make([]map[string]any, 0, len(parts))
	for i, part := range parts {
		images := []any{}
		if i == 0 {
			images = publicPostImages(media)
		}
		values = append(values, map[string]any{
			"content": strings.TrimSpace(part),
			"image":   images,
		})
	}
	return values
}

func publicPostImages(media []postizMedia) []any {
	if len(media) == 0 {
		return []any{}
	}
	images := make([]any, 0, len(media))
	for _, item := range media {
		id := strings.TrimSpace(item.ID)
		path := strings.TrimSpace(item.Path)
		if id == "" && path == "" {
			continue
		}
		image := map[string]any{}
		if id != "" {
			image["id"] = id
		}
		if path != "" {
			image["path"] = path
		}
		images = append(images, image)
	}
	if len(images) == 0 {
		return []any{}
	}
	return images
}

func publicPostSettings(platform string) map[string]any {
	platform = publicPostPlatform(platform)
	if platform == "" {
		return map[string]any{}
	}
	settings := map[string]any{"__type": platform}
	if platform == "x" {
		settings["who_can_reply_post"] = "everyone"
	}
	return settings
}

func publicPostPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "twitter":
		return "x"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func createPostParts(platform, content string, threadParts []string) ([]string, error) {
	content = strings.TrimSpace(content)
	hasThreadParts := len(threadParts) > 0
	if content != "" && hasThreadParts {
		return nil, fmt.Errorf("Postiz create_post braucht entweder content oder thread_parts, nicht beides")
	}
	if hasThreadParts {
		if publicPostPlatform(platform) != "x" {
			return nil, fmt.Errorf("Postiz create_post thread_parts wird nur fuer x/twitter unterstuetzt")
		}
		parts := make([]string, 0, len(threadParts))
		for i, part := range threadParts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				return nil, fmt.Errorf("Postiz create_post thread_parts[%d] ist leer", i)
			}
			if length := len([]rune(trimmed)); length > xPostCharLimit {
				return nil, fmt.Errorf("Postiz create_post thread_parts[%d] hat %d Zeichen; maximal erlaubt sind %d", i, length, xPostCharLimit)
			}
			parts = append(parts, trimmed)
		}
		return parts, nil
	}
	if content == "" {
		return nil, fmt.Errorf("Postiz create_post braucht content oder thread_parts")
	}
	return postContentParts(platform, content), nil
}

func postContentParts(platform, content string) []string {
	content = strings.TrimSpace(content)
	if content == "" || publicPostPlatform(platform) != "x" || len([]rune(content)) <= xPostCharLimit {
		return []string{content}
	}
	return splitTextByRuneLimit(content, xPostCharLimit)
}

func splitTextByRuneLimit(text string, limit int) []string {
	var parts []string
	var current string
	for _, word := range strings.Fields(text) {
		if len([]rune(word)) > limit {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			parts = append(parts, splitLongWord(word, limit)...)
			continue
		}
		if current == "" {
			current = word
			continue
		}
		if len([]rune(current))+1+len([]rune(word)) <= limit {
			current += " " + word
			continue
		}
		parts = append(parts, current)
		current = word
	}
	if current != "" {
		parts = append(parts, current)
	}
	if len(parts) == 0 {
		return []string{""}
	}
	return parts
}

func splitLongWord(word string, limit int) []string {
	runes := []rune(word)
	parts := make([]string, 0, (len(runes)+limit-1)/limit)
	for len(runes) > 0 {
		end := limit
		if len(runes) < end {
			end = len(runes)
		}
		parts = append(parts, string(runes[:end]))
		runes = runes[end:]
	}
	return parts
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
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(resp.Result, &result); err == nil {
		if result.IsError {
			var texts []string
			for _, item := range result.Content {
				if item.Text != "" {
					texts = append(texts, item.Text)
				}
			}
			if len(texts) > 0 {
				return "", fmt.Errorf("Postiz MCP tool result error: %s", strings.Join(texts, "\n"))
			}
			if len(result.StructuredContent) > 0 {
				return "", fmt.Errorf("Postiz MCP tool result error: %s", string(result.StructuredContent))
			}
			return "", fmt.Errorf("Postiz MCP tool result error")
		}
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

func parseMCPInitializeProtocol(body []byte) (string, error) {
	body = bytes.TrimSpace(body)
	if bytes.HasPrefix(body, []byte("event:")) || bytes.HasPrefix(body, []byte("data:")) {
		body = lastSSEData(body)
	}
	var resp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
		Error *struct {
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
	return resp.Result.ProtocolVersion, nil
}

func supportedMCPVersions(body []byte) []string {
	body = bytes.TrimSpace(body)
	if bytes.HasPrefix(body, []byte("event:")) || bytes.HasPrefix(body, []byte("data:")) {
		body = lastSSEData(body)
	}
	var resp struct {
		Error *struct {
			Message string `json:"message"`
			Data    struct {
				Supported []string `json:"supported"`
			} `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Error == nil {
		return nil
	}
	if len(resp.Error.Data.Supported) > 0 {
		return resp.Error.Data.Supported
	}
	message := resp.Error.Message
	start := strings.Index(message, "supported versions:")
	if start == -1 {
		return nil
	}
	start += len("supported versions:")
	end := strings.Index(message[start:], ")")
	if end != -1 {
		message = message[start : start+end]
	} else {
		message = message[start:]
	}
	var versions []string
	for _, value := range strings.Split(message, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			versions = append(versions, value)
		}
	}
	return versions
}

func commonMCPVersions(supported []string, tried map[string]bool) []string {
	supportedSet := make(map[string]bool, len(supported))
	for _, version := range supported {
		supportedSet[version] = true
	}
	var common []string
	for _, version := range mcpProtocolVersions {
		if supportedSet[version] && !tried[version] {
			common = append(common, version)
		}
	}
	return common
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
	rawPlatform := strings.ToLower(strings.TrimSpace(platform))
	platform = publicPostPlatform(platform)
	if rawPlatform == "" {
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
		if strings.EqualFold(integration.ID, rawPlatform) ||
			strings.EqualFold(integration.ID, platform) ||
			strings.EqualFold(integration.Platform, rawPlatform) ||
			strings.EqualFold(integration.Platform, platform) ||
			strings.EqualFold(integration.Name, rawPlatform) ||
			strings.EqualFold(integration.Name, platform) {
			matches = append(matches, integration.ID)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("keine Postiz Integration fuer %q gefunden; nutze list_channels und uebergib integration_id", rawPlatform)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("mehrere Postiz Integrationen fuer %q gefunden; nutze list_channels und uebergib integration_id", rawPlatform)
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
