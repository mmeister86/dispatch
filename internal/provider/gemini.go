package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com"

type Gemini struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	return NewGeminiWithBaseURL(apiKey, model, geminiDefaultBaseURL)
}

func NewGeminiWithBaseURL(apiKey, model, baseURL string) *Gemini {
	return &Gemini{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (g *Gemini) Name() string {
	return "gemini"
}

func (g *Gemini) ModelID() string {
	return g.model
}

func (g *Gemini) Complete(ctx context.Context, req Request) (<-chan Chunk, error) {
	out := make(chan Chunk)
	payload := newGeminiRequest(req)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", g.baseURL, url.PathEscape(g.model))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.apiKey)

	go func() {
		defer close(out)
		resp, err := g.client.Do(httpReq)
		if err != nil {
			out <- Chunk{Err: err}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			out <- Chunk{Err: geminiStatusError(resp)}
			return
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			chunk, ok, err := parseGeminiData([]byte(data))
			if err != nil {
				out <- Chunk{Err: err}
				return
			}
			if ok {
				out <- chunk
			}
		}
		if err := scanner.Err(); err != nil {
			out <- Chunk{Err: err}
		}
	}()
	return out, nil
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

func newGeminiRequest(req Request) geminiRequest {
	contents := make([]geminiContent, 0, len(req.Messages))
	for _, msg := range req.Messages {
		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}
	var system *geminiContent
	if req.System != "" {
		system = &geminiContent{Parts: []geminiPart{{Text: req.System}}}
	}
	return geminiRequest{SystemInstruction: system, Contents: contents}
}

type geminiStreamEvent struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

func parseGeminiData(data []byte) (Chunk, bool, error) {
	var event geminiStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return Chunk{}, false, err
	}
	if len(event.Candidates) == 0 {
		return Chunk{}, false, nil
	}
	candidate := event.Candidates[0]
	if len(candidate.Content.Parts) > 0 && candidate.Content.Parts[0].Text != "" {
		return Chunk{Text: candidate.Content.Parts[0].Text}, true, nil
	}
	if candidate.FinishReason != "" {
		return Chunk{StopReason: candidate.FinishReason}, true, nil
	}
	return Chunk{}, false, nil
}

func geminiStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return fmt.Errorf("gemini API returned %s", resp.Status)
	}
	return fmt.Errorf("gemini API returned %s: %s", resp.Status, bodyText)
}
