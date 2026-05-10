---
id: TASK-8
title: Add auto-resume sessions
status: Done
assignee: []
created_date: '2026-05-08 19:28'
updated_date: '2026-05-10 10:54'
labels: []
dependencies: []
priority: high
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Persist the active dispatch planning session so visible transcript and agent history survive crashes or restarts.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The active TUI transcript is restored automatically from local app state on startup
- [x] #2 Agent history is restored for subsequent LLM calls without storing secrets
- [x] #3 ctrl+l clears both the visible transcript and the persisted active session
- [x] #4 Session writes are atomic, stored with private permissions, and update during streaming chunks
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Verify baseline and session-related code paths
2. Add failing tests for session persistence, agent history restoration, and TUI auto-resume
3. Implement session store and wire it into agent/TUI startup
4. Persist changes during transcript updates and clear persisted session on ctrl+l
5. Run go test ./... and check acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Created implementation branch codex-auto-resume-sessions. Baseline go test ./... passed from cache; go mod download required network and passed after approval.

Implemented internal/session store, TUI auto-load/persist wiring, Agent SetHistory/ClearHistory, and CLI startup store wiring. Verified RED failures for missing session/agent/TUI APIs before implementation. Verification: go test ./... passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped auto-resume session persistence with transcript and agent history restoration, ctrl+l clearing, private atomic session storage, and test coverage. User confirmed task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
