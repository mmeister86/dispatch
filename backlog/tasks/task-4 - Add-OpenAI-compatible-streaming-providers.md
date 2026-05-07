---
id: TASK-4
title: Add OpenAI-compatible streaming providers
status: Done
assignee: []
created_date: '2026-05-07 13:05'
updated_date: '2026-05-07 19:17'
labels: []
dependencies: []
priority: high
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement OpenAI-compatible streaming support for dispatch so llm.provider=openai works, with OpenRouter and xAI sharing the same adapter via base URL.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Provider factory supports openai, openrouter, and xai
- [x] #2 OpenAI-compatible adapter sends chat completions streaming requests with system and message history
- [x] #3 Adapter parses SSE delta content and stop reasons
- [x] #4 Tests cover request shape, streamed text, stop reason, provider factory, and API errors
- [x] #5 README/config example document supported LLM providers and base behavior
- [x] #6 go test ./... passes and make build succeeds
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Fetch current OpenAI chat completions streaming docs via Context7
2. Write failing tests for OpenAI-compatible streaming and provider factory
3. Implement OpenAI-compatible adapter and factory cases
4. Update README/config example for supported providers
5. Run gofmt, go test ./..., make build, and binary smoke test
6. Check acceptance criteria and leave task open pending user confirmation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented OpenAI-compatible streaming adapter for openai, openrouter, and xai. Used Context7 OpenAI API reference for chat completions streaming: POST /v1/chat/completions, stream=true, delta.content, finish_reason. Added tests for request shape, streamed content, stop reasons, API errors, and factory support. Verification so far: internal/provider tests passed; go test ./... passed; make build passed; ./bin/dispatch check with llm.provider=openai config passed.

Final verification after Backlog update: go test ./... passed; make build passed; ./bin/dispatch check --config /private/tmp/dispatch-openai-test.toml passed; ./bin/dispatch version returned dispatch dev. Task remains In Progress pending user manual confirmation before Done.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
User confirmed this task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
