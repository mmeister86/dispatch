---
id: TASK-2
title: Add first streaming LLM provider
status: Done
assignee: []
created_date: '2026-05-07 12:42'
updated_date: '2026-05-07 19:17'
labels: []
dependencies: []
priority: high
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement Phase 2 slice for dispatch: define the provider interface and agent loop for plain streamed question-answering, then add an Anthropic Messages streaming adapter without tool-use yet.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Provider package defines shared request, message, chunk, and provider types
- [x] #2 Agent package can stream provider chunks for a user message and maintain in-memory history
- [x] #3 Anthropic adapter sends a Messages API request and parses streaming text deltas and stop events
- [x] #4 Cobra/TUI wiring can construct the configured provider and stream real assistant text into the chat viewport
- [x] #5 Focused tests cover agent history and Anthropic stream parsing with httptest
- [x] #6 go test ./... passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Fetch current Anthropic Messages streaming docs
2. Write failing tests for provider stream parsing and agent history streaming
3. Implement provider shared types and Anthropic SSE adapter
4. Implement agent streaming loop without tools
5. Wire provider/agent into the TUI and CLI startup
6. Run gofmt and go test ./...; record verified acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented Phase 2 streaming slice. Used Context7 docs for Anthropic streaming. Added provider shared types, Anthropic Messages SSE adapter, agent streaming loop with in-memory history trimming, and TUI wiring that streams chunks into the assistant message when Anthropic is configured. RED tests failed before implementation because provider/agent types were missing; GREEN tests then passed. Verification: go test ./... passed; go run . version returned dispatch dev; go run . check --config /private/tmp/dispatch-test.toml passed; rg found no stale social-planner references.

Final verification after Backlog update: go test ./... passed across cmd, config, internal/agent, internal/provider, and internal/tui. Task remains In Progress pending user manual confirmation before Done.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
User confirmed this task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
