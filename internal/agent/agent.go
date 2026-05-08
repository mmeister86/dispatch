package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/matthias/dispatch/internal/provider"
	"github.com/matthias/dispatch/internal/tools"
)

type Agent struct {
	provider   provider.Provider
	tools      *tools.Registry
	system     string
	maxHistory int

	mu      sync.Mutex
	history []provider.Message
}

func New(p provider.Provider, system string, maxHistory int) *Agent {
	return NewWithTools(p, nil, system, maxHistory)
}

func NewWithTools(p provider.Provider, registry *tools.Registry, system string, maxHistory int) *Agent {
	if maxHistory <= 0 {
		maxHistory = 40
	}
	return &Agent{
		provider:   p,
		tools:      registry,
		system:     system,
		maxHistory: maxHistory,
	}
}

func (a *Agent) Run(ctx context.Context, userMessage string) (<-chan provider.Chunk, error) {
	a.mu.Lock()
	a.history = append(a.history, provider.Message{Role: provider.RoleUser, Content: userMessage})
	a.trimLocked()
	requestHistory := append([]provider.Message(nil), a.history...)
	a.mu.Unlock()

	out := make(chan provider.Chunk)
	go func() {
		defer close(out)
		for round := 0; round < 6; round++ {
			stream, err := a.provider.Complete(ctx, provider.Request{
				System:    a.system,
				Messages:  requestHistory,
				Tools:     a.toolDefinitions(),
				MaxTokens: 1024,
			})
			if err != nil {
				out <- provider.Chunk{Err: err}
				return
			}

			var assistantText string
			var toolCalls []provider.ToolCall
			for chunk := range stream {
				if chunk.Text != "" {
					assistantText += chunk.Text
				}
				if chunk.ToolCall != nil {
					toolCalls = append(toolCalls, *chunk.ToolCall)
				}
				out <- chunk
				if chunk.Err != nil {
					return
				}
			}
			if assistantText != "" {
				a.appendHistory(provider.Message{Role: provider.RoleAssistant, Content: assistantText})
				requestHistory = append(requestHistory, provider.Message{Role: provider.RoleAssistant, Content: assistantText})
			}
			if len(toolCalls) == 0 {
				return
			}

			for _, call := range toolCalls {
				result, err := a.executeTool(ctx, call)
				if err != nil {
					result = fmt.Sprintf("Tool %s fehlgeschlagen: %v", call.Name, err)
				}
				toolMessage := provider.Message{Role: provider.RoleUser, Content: fmt.Sprintf("Tool result for %s:\n%s", call.Name, result)}
				a.appendHistory(toolMessage)
				requestHistory = append(requestHistory, toolMessage)
				out <- provider.Chunk{ToolResult: &provider.ToolResult{Name: call.Name, Result: result}}
			}
		}
		out <- provider.Chunk{Err: fmt.Errorf("agent stopped after too many tool rounds")}
	}()

	return out, nil
}

func (a *Agent) History() []provider.Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]provider.Message(nil), a.history...)
}

func (a *Agent) SetHistory(history []provider.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = append([]provider.Message(nil), history...)
	a.trimLocked()
}

func (a *Agent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = nil
}

func (a *Agent) trimLocked() {
	if len(a.history) <= a.maxHistory {
		return
	}
	a.history = append([]provider.Message(nil), a.history[len(a.history)-a.maxHistory:]...)
}

func (a *Agent) toolDefinitions() []provider.ToolDefinition {
	if a.tools == nil {
		return nil
	}
	return a.tools.Definitions()
}

func (a *Agent) executeTool(ctx context.Context, call provider.ToolCall) (string, error) {
	if a.tools == nil {
		return "", fmt.Errorf("keine Tools registriert")
	}
	return a.tools.Execute(ctx, call.Name, call.Arguments)
}

func (a *Agent) appendHistory(msg provider.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = append(a.history, msg)
	a.trimLocked()
}
