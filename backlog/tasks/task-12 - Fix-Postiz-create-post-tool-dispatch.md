---
id: TASK-12
title: Fix Postiz create post tool dispatch
status: In Progress
assignee:
  - Codex
created_date: '2026-05-08 21:25'
updated_date: '2026-05-08 21:29'
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
<!-- SECTION:NOTES:END -->
