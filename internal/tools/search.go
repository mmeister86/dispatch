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

type Search struct {
	apiKey   string
	provider string
	model    string
	recency  string
	baseURL  string
	client   *http.Client
}

func NewSearch(apiKey, providerName, model, recency, baseURL string) *Search {
	return &Search{apiKey: apiKey, provider: providerName, model: model, recency: recency, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 45 * time.Second}}
}

func (s *Search) Definitions() []provider.ToolDefinition {
	return []provider.ToolDefinition{{
		Name:        "search",
		Description: "Recherchiert ein aktuelles Thema im Web via Perplexity und gibt Antwort mit Quellen zurueck.",
		Schema: objectSchema(map[string]any{
			"query":   stringProp("Suchanfrage in natuerlicher Sprache"),
			"recency": stringProp(`"hour", "day", "week" oder "month"`),
			"model":   stringProp("Perplexity-Modell, z.B. sonar-pro"),
		}, "query"),
	}}
}

func (s *Search) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if name != "search" {
		return "", fmt.Errorf("Search Tool %q unbekannt", name)
	}
	if s.apiKey == "" {
		return "", fmt.Errorf("Search API Key fehlt")
	}
	var req struct {
		Query   string `json:"query"`
		Recency string `json:"recency"`
		Model   string `json:"model"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return "", err
	}
	if req.Model == "" {
		req.Model = s.model
	}
	if req.Recency == "" {
		req.Recency = s.recency
	}
	return s.perplexity(ctx, req.Query, req.Model, req.Recency)
}

func (s *Search) perplexity(ctx context.Context, query, model, recency string) (string, error) {
	payload := map[string]any{
		"model":                 model,
		"messages":              []map[string]string{{"role": "user", "content": query}},
		"search_recency_filter": recency,
		"return_citations":      true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("Search API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Citations []string `json:"citations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	var out []string
	if len(result.Choices) > 0 {
		out = append(out, result.Choices[0].Message.Content)
	}
	if len(result.Citations) > 0 {
		out = append(out, "\nQuellen:")
		for i, citation := range result.Citations {
			out = append(out, fmt.Sprintf("[%d] %s", i+1, citation))
		}
	}
	return strings.Join(out, "\n"), nil
}
