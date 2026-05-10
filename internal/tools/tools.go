package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/provider"
)

type Tool interface {
	Definitions() []provider.ToolDefinition
	Execute(ctx context.Context, name string, args json.RawMessage) (string, error)
}

type Registry struct {
	tools []Tool
}

func NewRegistry(cfg config.Config) *Registry {
	return &Registry{tools: []Tool{
		NewGitHub(cfg.GitHub.Token, "https://api.github.com"),
		NewSearch(cfg.Search.APIKey, cfg.Search.Provider, cfg.Search.Model, cfg.Search.Recency, "https://api.perplexity.ai"),
		NewPostiz(cfg.Postiz.APIKey, cfg.Postiz.BaseURL),
	}}
}

func (r *Registry) Definitions() []provider.ToolDefinition {
	var defs []provider.ToolDefinition
	for _, tool := range r.tools {
		defs = append(defs, tool.Definitions()...)
	}
	return defs
}

func (r *Registry) Execute(ctx context.Context, name string, args string) (string, error) {
	raw := json.RawMessage(args)
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	for _, tool := range r.tools {
		for _, def := range tool.Definitions() {
			if def.Name == name {
				return tool.Execute(ctx, name, raw)
			}
		}
	}
	return "", fmt.Errorf("unbekanntes Tool %q", name)
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}

func stringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func intProp(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func boolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func stringArrayProp(description string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items":       map[string]any{"type": "string"},
	}
}
