---
id: TASK-6
title: Handle empty GitHub repositories
status: Done
assignee: []
created_date: '2026-05-07 19:07'
updated_date: '2026-05-07 19:17'
labels: []
dependencies: []
priority: high
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fix GitHub recent commit retrieval so a single empty repository returning 409 does not make the whole GitHub tool call fail, and rebuild the local dispatch binary so it reads the current XDG config path.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 get_recent_commits preserves commits from non-empty repositories when another repository is empty
- [x] #2 GitHub 409 responses with message Git Repository is empty are treated as a clear non-fatal repo note
- [x] #3 Auth and permission errors such as 401, 403, and unexpected API errors still fail the tool call
- [x] #4 go test ./internal/tools -run GitHub -v and go test ./... pass
- [x] #5 go run . check and ./bin/dispatch check both report complete configuration after rebuilding
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a failing regression test for mixed GitHub repositories where one commits endpoint returns 409 Git Repository is empty.
2. Update the GitHub client to classify that 409 as an empty-repo condition while preserving other API failures.
3. Re-run targeted and full Go tests.
4. Rebuild ./bin/dispatch and verify both source and binary config checks.
5. Update Backlog acceptance criteria and notes, leaving task In Progress pending user confirmation.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED confirmed: go test ./internal/tools -run GitHub -v fails only on TestGitHubRecentCommitsNotesEmptyRepositories because the current client returns the 409 Git Repository is empty response as a fatal error.

GREEN confirmed: go test ./internal/tools -run GitHub -v now passes, including the mixed non-empty plus empty repository regression.

Verification: go test ./internal/tools -run GitHub -v passed; go test ./... passed; make build rebuilt ./bin/dispatch with date 2026-05-07T19:09:25Z; go run . check and ./bin/dispatch check both reported complete configuration. git rev-parse confirms this directory is not a git repository, so branch/PR finishing is not applicable here.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
User confirmed this task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
