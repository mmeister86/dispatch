package provider

import (
	"context"
	"fmt"

	"github.com/matthias/dispatch/config"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Provider interface {
	Complete(ctx context.Context, req Request) (<-chan Chunk, error)
	Name() string
	ModelID() string
}

type Request struct {
	System    string
	Messages  []Message
	Tools     []ToolDefinition
	MaxTokens int
}

type Message struct {
	Role    Role
	Content string
}

type Chunk struct {
	Text       string
	ToolCall   *ToolCall
	ToolResult *ToolResult
	StopReason string
	Err        error
}

type ToolDefinition struct {
	Name        string
	Description string
	Schema      map[string]any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ToolResult struct {
	Name   string
	Result string
}

func NewFromConfig(cfg config.Config) (Provider, error) {
	if cfg.LLM.APIKey == "" {
		return nil, nil
	}

	switch cfg.LLM.Provider {
	case "anthropic":
		return NewAnthropic(cfg.LLM.APIKey, cfg.LLM.Model), nil
	case "openai":
		return NewOpenAICompatible("openai", cfg.LLM.APIKey, cfg.LLM.Model, openAIDefaultBaseURL), nil
	case "openrouter":
		return NewOpenAICompatible("openrouter", cfg.LLM.APIKey, cfg.LLM.Model, openRouterDefaultBaseURL), nil
	case "xai":
		return NewOpenAICompatible("xai", cfg.LLM.APIKey, cfg.LLM.Model, xAIDefaultBaseURL), nil
	case "gemini":
		return NewGemini(cfg.LLM.APIKey, cfg.LLM.Model), nil
	default:
		return nil, fmt.Errorf("provider %q ist nicht implementiert", cfg.LLM.Provider)
	}
}
