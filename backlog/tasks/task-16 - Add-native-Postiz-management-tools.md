---
id: TASK-16
title: Add native Postiz management tools
status: Done
assignee: []
created_date: '2026-05-10 09:01'
updated_date: '2026-05-10 10:54'
labels: []
dependencies: []
priority: high
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement native Postiz Public API management tools alongside the existing MCP integration so self-hosted users can list and manage posts without a postiz CLI binary.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 list_posts uses native Public API with default date window and optional filters
- [x] #2 delete_post set_post_status and connect_post_release require confirmed=true before network calls
- [x] #3 upload_media sends multipart file uploads through the Public API
- [x] #4 analytics and missing-release tools call the documented Public API endpoints
- [x] #5 self-hosted MCP URLs preserve /api when deriving Public API endpoints
- [x] #6 TUI and README expose the new posts overview shortcut
- [x] #7 go test ./internal/tools -count=1 and go test ./... pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing Postiz management API tests
2. Implement native Public API helpers and tool handlers
3. Add TUI shortcut and docs
4. Run focused and full verification
5. Update Backlog notes and acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented native Postiz Public API management tools: list_posts, delete_post, set_post_status, upload_media, get_platform_analytics, get_post_analytics, list_missing_post_content, and connect_post_release. Kept create_post MCP-first behavior unchanged. Added reusable endpoint derivation for self-hosted /api/mcp URLs, confirmation gates for destructive tools, ctrl+o posts shortcut, README/system prompt updates, and httptest coverage for endpoints, auth, query params, multipart upload, default date windows, status validation, and confirmation rejection. Verification: go test ./internal/tools -count=1 passed; go test ./... passed; git diff --check passed.

Adjusted shortcut behavior per user request: ctrl+p now lists Postiz posts for the default last-30/next-30-day window instead of listing channels. Removed the temporary ctrl+o posts shortcut from TUI help/footer and README so there is a single posts shortcut. Verification: go test ./internal/tui ./internal/tools -count=1 passed; git diff --check passed.

Adjusted shortcut split per user request: ctrl+p lists Postiz posts, while ctrl+o lists connected Postiz accounts/providers via the existing channel listing prompt. Updated TUI help/footer and README. Verification: go test ./internal/tui ./internal/tools -count=1 passed; git diff --check passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped native Postiz Public API management tools for listing and managing posts, upload, analytics, and missing-release repair; added self-hosted endpoint derivation, destructive confirmation gates, ctrl+p posts and ctrl+o connected accounts shortcuts, README/system prompt updates, and regression coverage. Merged into main and pushed to origin/main after go test ./... passed.
<!-- SECTION:FINAL_SUMMARY:END -->
