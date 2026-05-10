---
id: TASK-22
title: Fix stale user input after agent responses
status: In Progress
assignee: []
created_date: '2026-05-10 18:20'
updated_date: '2026-05-10 18:23'
labels: []
dependencies: []
priority: high
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Investigate and fix the intermittent issue where previous user inputs remain in the composer input field even after the agent has answered.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Identify the root cause or a narrowly supported hypothesis for stale composer input state
- [x] #2 Add or update regression coverage for the stale input scenario where feasible
- [x] #3 Implement a scoped fix that clears submitted user input after successful handoff without dropping active edits
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Locate composer/input state and submission flow
2. Compare submit success/failure/streaming paths
3. Reproduce or add focused regression coverage
4. Implement scoped fix
5. Run relevant tests and record notes
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Located submit flow in internal/tui/app.go: Enter trims textarea.Value(), stores lastInput, calls textarea.Reset(), then starts agent/session command. Generic textarea.Update runs only for unhandled messages after the switch. Need test whether Reset leaves textarea internals in a state that can re-render stale content, especially after completion.

Evidence: Bubbles v1.0.0 textarea.Reset() reinitializes value to an empty single-row buffer, resets row/col, and moves its viewport to top. Current app calls Reset only in the Enter submit path; direct startAgent callers from ctrl+p/ctrl+o/ctrl+r/ctrl+s do not clear or snapshot the composer.

RED: Added TestPlainUpDoesNotRestoreSubmittedInput. It fails on current code because pressing plain up after submit restores the answered prompt into the textarea. Hypothesis confirmed: the stale input is caused by the plain-up lastInput recall path, likely triggered accidentally while reading/scrolling after an answer.

GREEN: Removed plain-up last-input recall from the TUI model and removed the README shortcut entry. Focused regression test and full go test ./... now pass; full suite needed outside sandbox because httptest binds loopback ports.
<!-- SECTION:NOTES:END -->
