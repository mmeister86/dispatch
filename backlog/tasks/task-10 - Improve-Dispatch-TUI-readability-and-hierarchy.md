---
id: TASK-10
title: Improve Dispatch TUI readability and hierarchy
status: In Progress
assignee: []
created_date: '2026-05-08 21:03'
updated_date: '2026-05-08 21:03'
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
- [ ] #1 Main response content has stronger contrast and clearer visual hierarchy
- [ ] #2 Tool call and tool result blocks are visually compact and distinguishable from assistant content
- [ ] #3 Long summary lines wrap cleanly within the terminal width
- [ ] #4 Relevant tests or snapshots cover the updated rendering behavior
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Locate TUI rendering and current tests
2. Add failing tests for Markdown cleanup, readable wrapping, and compact tool blocks
3. Update renderer/styles with clearer hierarchy and contrast
4. Run focused and full Go tests
5. Mark verified acceptance criteria, leave task In Progress for user confirmation
<!-- SECTION:PLAN:END -->
