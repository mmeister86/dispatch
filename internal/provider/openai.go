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
	openAIDefaultBaseURL     = "https://api.openai.com"
	openRouterDefaultBaseURL = "https://openrouter.ai/api"
	xAIDefaultBaseURL        = "https://api.x.ai"
)

type OpenAICompatible struct {
	name    string
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAICompatible(name, apiKey, model, baseURL string) *OpenAICompatible {
	return &OpenAICompatible{
		name:    name,
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (o *OpenAICompatible) Name() string {
	return o.name
}

func (o *OpenAICompatible) ModelID() string {
	return o.model
}

func (o *OpenAICompatible) Complete(ctx context.Context, req Request) (<-chan Chunk, error) {
	out := make(chan Chunk)
	payload := newOpenAIRequest(o.model, req)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("accept", "text/event-stream")
	httpReq.Header.Set("authorization", "Bearer "+o.apiKey)

	go func() {
		defer close(out)
		resp, err := o.client.Do(httpReq)
		if err != nil {
			out <- Chunk{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			out <- Chunk{Err: openAIStatusError(o.name, resp)}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				continue
			}

			chunk, ok, err := parseOpenAIData([]byte(data))
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

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	Tools     []openAITool    `json:"tools,omitempty"`
	Stream    bool            `json:"stream"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

func newOpenAIRequest(model string, req Request) openAIRequest {
	messages := make([]openAIMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.System})
	}
	for _, msg := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}
	tools := make([]openAITool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Schema,
			},
		})
	}

	return openAIRequest{
		Model:     model,
		Messages:  messages,
		Tools:     tools,
		Stream:    true,
		MaxTokens: req.MaxTokens,
	}
}

type openAIStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func parseOpenAIData(data []byte) (Chunk, bool, error) {
	var event openAIStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return Chunk{}, false, err
	}
	if len(event.Choices) == 0 {
		return Chunk{}, false, nil
	}

	choice := event.Choices[0]
	if choice.Delta.Content != "" {
		return Chunk{Text: choice.Delta.Content}, true, nil
	}
	if len(choice.Delta.ToolCalls) > 0 {
		call := choice.Delta.ToolCalls[0]
		return Chunk{ToolCall: &ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		}}, true, nil
	}
	if choice.FinishReason != "" {
		return Chunk{StopReason: choice.FinishReason}, true, nil
	}
	return Chunk{}, false, nil
}

func openAIStatusError(providerName string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return fmt.Errorf("%s API returned %s", providerName, resp.Status)
	}
	return fmt.Errorf("%s API returned %s: %s", providerName, resp.Status, bodyText)
}
