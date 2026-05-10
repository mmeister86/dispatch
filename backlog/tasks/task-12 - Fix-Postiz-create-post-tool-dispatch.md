---
id: TASK-12
title: Fix Postiz create post tool dispatch
status: Done
assignee:
  - Codex
created_date: '2026-05-08 21:25'
updated_date: '2026-05-10 10:54'
labels: []
dependencies: []
references:
  - TASK-9
modified_files:
  - internal/tools/postiz.go
  - internal/tools/tools_test.go
priority: high
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Investigate and fix the Postiz create_post flow that currently reports `Unknown tool: schedulePostTool` after confirmed post creation attempts. Preserve the documented Postiz MCP flow from TASK-9 while making create_post call the actual exposed Postiz scheduling tool successfully.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 create_post no longer returns `Unknown tool: schedulePostTool` when scheduling through Postiz MCP
- [x] #2 Postiz tool-name discovery or mapping matches the tools exposed by the configured Postiz MCP server
- [x] #3 Automated tests cover the create_post scheduling path and the resolved Postiz MCP tool name
- [x] #4 X/Twitter create_post content over 280 characters is split into a valid thread instead of creating an over-limit single post
- [x] #5 create_post converts past scheduled_at values to immediate publish (`type=now`) instead of leaving posts scheduled in the past
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a regression test where `schedulePostTool` returns an MCP `Unknown tool` error and `create_post` falls back to Postiz Public API scheduling instead of surfacing the MCP tool error.
2. Add/adjust helper coverage for deriving the Postiz API key from either `postiz.api_key` or a full `/mcp/<key>` URL.
3. Implement the smallest create_post fallback path: keep MCP `integrationList` for platform resolution, call Public API `POST /public/v1/posts` only when MCP scheduling reports an unknown schedule tool, and preserve existing MCP success behavior.
4. Run focused `go test ./internal/tools`, then full `go test ./...`.
5. Update TASK-12 notes and check acceptance criteria only after verification.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause evidence: local create_post was hard-coded to call Postiz MCP `schedulePostTool`. Current Context7 Postiz docs still document that tool, but the reported runtime server response is `Unknown tool: schedulePostTool`, so the client must tolerate a configured Postiz MCP server that does not expose the scheduling tool even though `integrationList` works.

Implemented fallback behavior: create_post still resolves the integration via MCP `integrationList` and tries MCP `schedulePostTool` first. If the MCP server returns an `Unknown tool` error for `schedulePostTool`, dispatch schedules through Postiz Public API `POST /public/v1/posts` using the configured API key or the key embedded in a full `/mcp/<key>` URL.

Verification: added a failing regression test for `Unknown tool: schedulePostTool`; verified it failed before implementation with `go test ./internal/tools` outside the sandbox. After implementation and gofmt, `go test ./internal/tools` passes. Full `go test ./...` currently fails in unrelated pre-existing session/TUI tests referencing removed `session.Session` fields and methods (`ID`, `Title`, `CreatedAt`, `Store.List`, `Store.StartNew`, `session.Summary`).

Refined regression coverage so the successful MCP path and fallback path both work when Postiz is configured with a full `/api/mcp/<key>` URL and no separate `postiz.api_key`, matching the documented self-hosted setup from TASK-9. Fresh verification remains: `go test ./internal/tools` passes; `go test ./...` still fails only in unrelated session/TUI test compilation.

Reopened investigation based on user-provided runtime transcript: repeated `create_post` calls still invoke a missing `schedulePostTool` and the agent stops after too many tool rounds. Returning to systematic debugging before making further changes.

Root cause refinement: the runtime transcript matches an MCP `tools/call` response that returns HTTP/JSON-RPC success with `result.isError=true` and text `Unknown tool: schedulePostTool`, not only a JSON-RPC `error`. `parseMCPToolResult` treated that as a successful string result, so `create_post` never entered the existing Public API fallback and the agent kept retrying the apparent tool result.

Added regression coverage for `schedulePostTool` returning `result.isError=true`; verified the test failed before the parser fix because the Public API fallback was not called. Updated the MCP result parser to convert `isError` tool results into Go errors, which lets the existing `isUnknownMCPTool` fallback path schedule via `POST /public/v1/posts`. Verification: `go test ./internal/tools` passes and full `go test ./...` passes.

Second runtime transcript showed the MCP fallback did run, but the Public API response was a Next.js login HTML page. Root cause: for self-hosted full MCP URLs like `/api/mcp/<key>`, `publicAPIEndpoint` stripped `/api` and posted to `/public/v1/posts`, which hits the frontend route on the user's instance. The correct self-hosted backend route derived from that MCP URL is `/api/public/v1/posts`, while cloud `api.postiz.com` remains `/public/v1/posts`.

Added a red endpoint regression test for self-hosted `/api/mcp/<key>` URLs; it failed with `https://postiz.example/public/v1/posts`. Updated endpoint derivation to preserve the `/api` backend base and adjusted fallback tests accordingly. Verification after fix: `go test ./internal/tools` passes and full `go test ./...` passes.

Third runtime transcript reached the corrected self-hosted Public API route and returned Postiz validation errors. Root cause: the fallback payload encoded nil media as JSON `null` for `posts.0.value.0.image`, while Postiz expects an array, and X scheduling requires provider settings including `__type: "x"` and `who_can_reply_post: "everyone"`. Context7 Postiz docs confirm the X settings schema and an empty `image: []` in schedule examples.

Added/updated regression coverage so the Public API fallback for X asserts `image` is a JSON array and settings include `__type: x` plus `who_can_reply_post: everyone`; verified it failed before implementation with `public image must be an empty array, got nil`. Implemented `publicPostImages` and `publicPostSettings`, passed platform through to the fallback, and verified `go test ./internal/tools` plus full `go test ./...` pass.

Fourth runtime issue: Postiz accepted the payload but X preview showed 409/280, meaning dispatch sent one X post instead of a thread. Context7 confirms Postiz represents X threads as multiple MCP `postsAndComments` items or multiple Public API `value` items.

Implemented default X/Twitter behavior to preserve content as a thread rather than truncating: content over 280 runes is split on word boundaries, MCP payload receives multiple `postsAndComments` entries, Public API fallback receives multiple `value` entries, and X reply settings remain set to `everyone`. Regression tests first failed with a single over-limit item, then passed after the split implementation. Verification: `go test ./internal/tools` and `go test ./...` pass.

Fifth runtime issue: Postiz calendar showed a created X post at 02:00 on 2026-05-08, already about 22 hours in the past, so it stayed queued/scheduled instead of publishing to Twitter. Root cause: dispatch always sent `type: schedule` with the model-provided `scheduled_at`, even when that timestamp was already in the past.

Context7 confirms Postiz MCP accepts `type: draft | schedule | now`, and Postiz SDK/Public API docs also list `now` for immediate publishing. Added regression coverage for both MCP and Public API fallback paths: past `scheduled_at` values now produce `type: now` with the current UTC timestamp, while future values remain `type: schedule`. Verification: `go test ./internal/tools` and full `go test ./...` pass.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped robust Postiz create_post dispatch with MCP schedule fallback to Public API, MCP isError handling, self-hosted /api endpoint preservation, X settings/media validation, automatic X thread splitting, and immediate publish for past scheduled_at values. User confirmed task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
