---
id: TASK-25
title: Add autocomplete for existing slash commands
status: Done
assignee: []
created_date: '2026-05-15 12:58'
updated_date: '2026-05-16 17:34'
labels: []
dependencies: []
priority: high
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement autocomplete suggestions for existing dispatch slash commands so partial input like /ses shows matching session commands.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Typing a partial slash command shows matching existing commands
- [x] #2 Session commands are suggested for input such as /ses
- [x] #3 Suggestions integrate with the existing dispatch command input flow without breaking normal input
- [x] #4 Automated tests cover the autocomplete matching behavior
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Confirm autocomplete interaction and rendering behavior
2. Add failing TUI tests for slash-command suggestions
3. Implement command registry, matching, and footer suggestion rendering
4. Verify focused and full Go tests
5. Ask for manual-test confirmation before closing task
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Context: slash commands are currently handled in internal/tui/app.go on enter via isSessionCommand/isDebugCommand. The composer uses bubbles textarea and renderFooter currently shows input, shortcuts, and image paste hint. Existing session-command tests live in internal/tui/app_test.go.

Starting implementation from the approved extensible registry plan. Existing worktree has unrelated modified files and existing TUI edits; autocomplete changes will be scoped to internal/tui/app.go and internal/tui/app_test.go and will preserve current behavior.

RED: added TUI tests for /ses suggestions, tab completion, placeholder command completion, /debug suggestions, hidden suggestions for plain text, and registry-driven future commands. Initial go test ./internal/tui -count=1 failed on undefined slashCommands/slashCommand types as expected before implementation.

GREEN iteration: added slash command registry and footer rendering. First focused run exposed two fixes: wrapped footer text needed normalized assertion, and local /session commands with arguments needed prefix recognition from the registry.

Verification: GOCACHE=/private/tmp/dispatch-go-build go test ./internal/tui -count=1 passed. git diff --check passed. Full GOCACHE=/private/tmp/dispatch-go-build go test ./... -count=1 first failed inside the sandbox because httptest could not bind loopback ports, then passed outside the sandbox.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified against code and tests. Existing slash commands are registry-backed, partial inputs such as /ses render suggestions, and tab completion integrates with the current TUI input flow.
<!-- SECTION:FINAL_SUMMARY:END -->
