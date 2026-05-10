---
id: TASK-15
title: Update repository documentation
status: Done
assignee: []
created_date: '2026-05-09 05:35'
updated_date: '2026-05-10 10:55'
labels: []
dependencies: []
priority: high
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Refresh the README and related repository documentation so it matches the current codebase, commands, and workflows.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 README accurately describes the project, setup, usage, and development workflow
- [x] #2 Related documentation files are reviewed and updated where stale
- [x] #3 Documentation changes are checked for consistency against the codebase
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Audit README and related docs against the current Go CLI codebase
2. Update stale setup, configuration, provider, session, and tool documentation
3. Verify docs for consistency and mark acceptance criteria that were checked
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Refreshed README, CONTRIBUTING, PLAN, CHANGELOG, and config.toml.example against the current CLI/TUI implementation.
Verification: go test ./... passed; git diff --check passed; stale-doc search found no matches for removed Phase 1/list_posts/old config wording.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped refreshed repository documentation covering README, CONTRIBUTING, PLAN, CHANGELOG, and config example alignment with the current CLI/TUI behavior. User confirmed task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
