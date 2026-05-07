package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com"
	anthropicVersion        = "2023-06-01"
	defaultMaxTokens        = 1024
)

type Anthropic struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewAnthropic(apiKey, model string) *Anthropic {
	return NewAnthropicWithBaseURL(apiKey, model, anthropicDefaultBaseURL)
}

func NewAnthropicWithBaseURL(apiKey, model, baseURL string) *Anthropic {
	return &Anthropic{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (a *Anthropic) Name() string {
	return "anthropic"
}

func (a *Anthropic) ModelID() string {
	return a.model
}

func (a *Anthropic) Complete(ctx context.Context, req Request) (<-chan Chunk, error) {
	out := make(chan Chunk)
	payload := newAnthropicRequest(a.model, req)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("x-api-key", a.apiKey)

	go func() {
		defer close(out)
		resp, err := a.client.Do(httpReq)
		if err != nil {
			out <- Chunk{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			out <- Chunk{Err: anthropicStatusError(resp)}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		var currentTool *ToolCall
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				continue
			}

			chunk, ok, err := parseAnthropicData([]byte(data))
			if err != nil {
				out <- Chunk{Err: err}
				return
			}
			var event anthropicStreamEvent
			_ = json.Unmarshal([]byte(data), &event)
			if event.Type == "content_block_start" && event.ContentBlock.Type == "tool_use" {
				currentTool = &ToolCall{ID: event.ContentBlock.ID, Name: event.ContentBlock.Name}
				continue
			}
			if event.Type == "content_block_delta" && event.Delta.Type == "input_json_delta" && currentTool != nil {
				currentTool.Arguments += event.Delta.PartialJSON
				continue
			}
			if event.Type == "content_block_stop" && currentTool != nil {
				out <- Chunk{ToolCall: currentTool}
				currentTool = nil
				continue
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

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string               `json:"role"`
	Content []anthropicTextBlock `json:"content"`
}

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

func newAnthropicRequest(model string, req Request) anthropicRequest {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	messages := make([]anthropicMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, anthropicMessage{
			Role: string(msg.Role),
			Content: []anthropicTextBlock{{
				Type: "text",
				Text: msg.Content,
			}},
		})
	}
	tools := make([]anthropicTool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Schema,
		})
	}

	return anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    req.System,
		Stream:    true,
		Messages:  messages,
		Tools:     tools,
	}
}

type anthropicStreamEvent struct {
	Type         string `json:"type"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
}

func parseAnthropicData(data []byte) (Chunk, bool, error) {
	var event anthropicStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return Chunk{}, false, err
	}

	switch event.Type {
	case "content_block_delta":
		if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			return Chunk{Text: event.Delta.Text}, true, nil
		}
	case "message_delta":
		if event.Delta.StopReason != "" {
			return Chunk{StopReason: event.Delta.StopReason}, true, nil
		}
	}

	return Chunk{}, false, nil
}

func anthropicStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return fmt.Errorf("anthropic API returned %s", resp.Status)
	}
	return fmt.Errorf("anthropic API returned %s: %s", resp.Status, bodyText)
}
