---
id: TASK-20
title: Combine OpenAI streaming tool call deltas
status: Done
assignee: []
created_date: '2026-05-10 13:47'
updated_date: '2026-05-16 17:34'
labels: []
dependencies: []
priority: high
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
OpenAI streams tool calls across multiple deltas. dispatch currently treats each delta as a full call, causing repeated executions of an empty tool name after valid tool calls.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 OpenAI streaming tool call deltas are accumulated into one complete tool call per index
- [x] #2 Empty-name tool call deltas are not emitted or executed as standalone tool calls
- [x] #3 Regression coverage reproduces split tool call streaming
- [x] #4 Relevant Go tests pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a provider regression test for split OpenAI tool_call SSE deltas
2. Verify it fails by emitting an empty-name tool call
3. Add stateful accumulation in the OpenAI streaming loop
4. Run focused provider tests and the full suite
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause: OpenAI streams function/tool calls as deltas. dispatch emitted every delta as a complete ToolCall, so later argument-only deltas had empty names and were executed as unknown tool "". Added stateful accumulation by tool call index in the OpenAI streaming loop and emit completed calls only when finish_reason is tool_calls. Verification: go test ./internal/provider -count=1 and go test ./... -count=1 passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified against code and tests. OpenAI streaming tool call deltas are accumulated per index and emitted only once complete, preventing empty-name standalone tool calls.
<!-- SECTION:FINAL_SUMMARY:END -->
