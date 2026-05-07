package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/matthias/dispatch/internal/provider"
)

type Postiz struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewPostiz(apiKey, baseURL string) *Postiz {
	return &Postiz{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Postiz) Definitions() []provider.ToolDefinition {
	return []provider.ToolDefinition{
		{Name: "create_post", Description: "Plant einen Social-Media-Post via Postiz. Nur nach expliziter User-Bestaetigung nutzen.", Schema: objectSchema(map[string]any{
			"platform":     stringProp("Zielplattform, z.B. linkedin oder twitter"),
			"content":      stringProp("Post-Text"),
			"scheduled_at": stringProp("ISO-8601 Zeitpunkt"),
			"media_urls":   stringArrayProp("Optionale Medien-URLs"),
			"confirmed":    boolProp("Muss true sein, nachdem der User die Vorschau explizit bestaetigt hat"),
		}, "platform", "content", "scheduled_at", "confirmed")},
		{Name: "list_posts", Description: "Listet geplante, veroeffentlichte oder Draft-Posts aus Postiz.", Schema: objectSchema(map[string]any{
			"status": stringProp(`"scheduled", "published" oder "draft"`),
			"limit":  intProp("Maximale Anzahl, bis 20"),
		})},
		{Name: "list_channels", Description: "Listet verbundene Postiz-Social-Channels.", Schema: objectSchema(map[string]any{})},
	}
}

func (p *Postiz) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("Postiz API Key fehlt")
	}
	switch name {
	case "create_post":
		var req struct {
			Confirmed bool `json:"confirmed"`
		}
		_ = json.Unmarshal(args, &req)
		if !req.Confirmed {
			return "", fmt.Errorf("Postiz create_post erfordert confirmed=true nach expliziter User-Bestaetigung")
		}
		return p.post(ctx, "/api/posts", args)
	case "list_posts":
		var req struct {
			Status string `json:"status"`
			Limit  int    `json:"limit"`
		}
		_ = json.Unmarshal(args, &req)
		if req.Limit <= 0 || req.Limit > 20 {
			req.Limit = 20
		}
		path := fmt.Sprintf("/api/posts?status=%s&limit=%d", req.Status, req.Limit)
		return p.get(ctx, path)
	case "list_channels":
		return p.get(ctx, "/api/integrations")
	default:
		return "", fmt.Errorf("Postiz Tool %q unbekannt", name)
	}
}

func (p *Postiz) post(ctx context.Context, path string, payload []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

func (p *Postiz) get(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Postiz API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}
