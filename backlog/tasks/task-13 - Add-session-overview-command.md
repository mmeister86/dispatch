---
id: TASK-13
title: Add session overview command
status: In Progress
assignee: []
created_date: '2026-05-08 21:26'
updated_date: '2026-05-08 21:31'
labels: []
dependencies: []
priority: high
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a local /session command that lists persisted dispatch sessions and lets the user start or open sessions without sending the command to the LLM.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Typing /session renders a local overview of persisted sessions without calling the agent
- [x] #2 The session store can persist and list multiple session files with metadata sorted by last update
- [x] #3 Typing /session new starts a fresh active session and preserves the previous session
- [x] #4 Typing /session open <id> restores the selected transcript and agent history
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing tests for multi-session store listing and slash-command behavior
2. Extend session store with IDs, metadata, active-session handling, list/new/open support
3. Teach TUI to intercept /session commands before the agent
4. Keep legacy current.json auto-resume behavior compatible
5. Run go test ./... and check acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented multi-session store metadata and local /session slash commands for list, new, open, and delete. RED tests failed on missing Session metadata, Store methods, and TUI command wiring before implementation. Verification so far: go test ./... passed.
<!-- SECTION:NOTES:END -->
