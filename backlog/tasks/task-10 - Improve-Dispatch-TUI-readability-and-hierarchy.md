---
id: TASK-10
title: Improve Dispatch TUI readability and hierarchy
status: In Progress
assignee: []
created_date: '2026-05-08 21:03'
updated_date: '2026-05-08 21:14'
labels: []
dependencies: []
priority: high
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Improve the Dispatch terminal UI so generated reports are easier to scan, tool calls are visually quieter, and final summaries stand out.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Main response content has stronger contrast and clearer visual hierarchy
- [x] #2 Tool call and tool result blocks are visually compact and distinguishable from assistant content
- [x] #3 Long summary lines wrap cleanly within the terminal width
- [x] #4 Relevant tests or snapshots cover the updated rendering behavior
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Locate TUI rendering and current tests
2. Add failing tests for Markdown cleanup, readable wrapping, and compact tool blocks
3. Update renderer/styles with clearer hierarchy and contrast
4. Run focused and full Go tests
5. Mark verified acceptance criteria, leave task In Progress for user confirmation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added failing TUI tests for Markdown cleanup, readable message width, and compact tool-result rendering. Red phase confirmed with go test ./internal/tui.

Implemented readable message-width cap, terminal Markdown cleanup for headings/rules/bold emphasis, compact tool labels/previews, and a higher-contrast TUI palette. Verified with go test ./internal/tui and go test ./... .

Clarification: the focused TUI tests were already present in the branch and served as the red regression coverage; implementation now satisfies those tests.

Follow-up root cause: transparent terminals showed through because the rendered view exceeded the terminal height and did not explicitly paint every terminal row/cell with the app background. Added paintCanvas plus exact layout height accounting and a regression test for full-width/full-height background rendering.

Follow-up root cause 2: macOS Terminal showed dark text islands because inner label/message/textarea styles set foreground colors without explicit backgrounds. Added explicit backgrounds to nested text styles and textarea focused/blurred styles, plus a regression test that message lines carry the surface background.
<!-- SECTION:NOTES:END -->
