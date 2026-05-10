---
id: TASK-7
title: Show tool calls before agent replies
status: Done
assignee: []
created_date: '2026-05-07 19:14'
updated_date: '2026-05-10 10:54'
labels: []
dependencies: []
priority: high
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Adjust the TUI transcript rendering so tool call blocks appear before the associated agent answer and keep displayed tool call content concise when possible.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tool calls render before the agent response they belong to
- [x] #2 Tool call blocks display a shortened, readable representation instead of full verbose payloads where appropriate
- [x] #3 Existing transcript rendering behavior remains stable for normal user and agent messages
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Confirm transcript ordering path in TUI update/render code
2. Delay agent message creation until text arrives so tool blocks can render before the resulting answer
3. Add concise tool-call/result formatting helpers and focused tests
4. Run Go tests and check acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Found root cause: startAgent creates an empty AGENT message before the stream. Later text appends to that early message, so tool blocks added during the stream appear below the answer.

Implemented delayed agent message creation and compact tool formatting. Verified with go test ./... after rerunning outside the sandbox because Go needed access to the normal build cache.

Follow-up fix after manual screenshot: when a tool call arrives after preliminary assistant text, remove that provisional agent message and reset activeMsg so the post-tool answer starts after the tool blocks. Added regression coverage for text → tool call → tool result → answer text. Verified with go test ./... .
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped TUI transcript ordering so tool calls/results render before the associated final agent answer, with compact readable tool formatting and regression coverage. User confirmed task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
