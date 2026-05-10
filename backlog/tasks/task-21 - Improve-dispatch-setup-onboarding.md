---
id: TASK-21
title: Improve dispatch setup onboarding
status: In Progress
assignee: []
created_date: '2026-05-10 17:24'
updated_date: '2026-05-10 17:27'
labels: []
dependencies: []
priority: high
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Make dispatch --setup a compact wizard with selectable providers/models, visible masked secret entry, a non-secret summary, and clear next steps.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Provider and model are selectable from curated lists with a custom model escape hatch
- [x] #2 Secret prompts echo masked characters in interactive terminals and preserve non-TTY fallback behavior
- [x] #3 Setup shows a non-secret summary and confirms before writing config
- [x] #4 Tests cover selection defaults, invalid selection retry, custom model entry, save confirmation, non-TTY secret fallback, and masked input edge cases
- [x] #5 go test ./cmd ./config ./... and git diff --check pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add setup wizard regression tests for provider/model selection, custom models, confirmation, and non-TTY secret fallback
2. Add masked input tests around paste, backspace, empty enter, and interrupt
3. Implement curated provider/model choices and prompt helpers
4. Implement interactive masked secret entry with safe fallback
5. Add final summary/confirmation and next-step output
6. Run focused and full verification commands
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented compact setup wizard with curated provider/model choices, custom model escape hatch, masked secret input with star echo, save confirmation, and non-secret summary. Verification: go test ./cmd ./config ./... passed; git diff --check passed.
<!-- SECTION:NOTES:END -->
