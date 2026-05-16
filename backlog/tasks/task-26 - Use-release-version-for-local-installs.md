---
id: TASK-26
title: Use release version for local installs
status: In Progress
assignee: []
created_date: '2026-05-15 13:36'
updated_date: '2026-05-15 13:39'
labels: []
dependencies: []
priority: high
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Make local build and install targets report the current dispatch release version instead of always embedding dev.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 make install embeds 0.1.0 or the current Git tag-derived release version when available
- [x] #2 dispatch version no longer reports dev for a tagged release checkout
- [x] #3 Automated or scripted verification covers version metadata resolution
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Identify how local version metadata is computed today
2. Add a failing regression check for tagged/default version resolution
3. Update the build metadata path minimally
4. Verify make install/build and Go tests
5. Record results and leave task In Progress for user confirmation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause: Makefile defaulted VERSION to dev and COMMIT to none for local build/install paths, while the repository already has tag v0.1.0. Added a Makefile metadata regression check and changed defaults to derive VERSION from the latest Git tag without leading v and COMMIT from the current short SHA. Verification: make test-version passed; make build produced dispatch 0.1.0; make test passed outside sandbox because httptest needs local port binding; make install updated the installed dispatch binary; dispatch version returned dispatch 0.1.0 (6fd693a, 2026-05-15T13:38:52Z).
<!-- SECTION:NOTES:END -->
