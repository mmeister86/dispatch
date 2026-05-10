---
id: TASK-11
title: Add MCP protocol version negotiation
status: Done
assignee: []
created_date: '2026-05-08 21:20'
updated_date: '2026-05-10 10:54'
labels: []
dependencies: []
priority: high
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Make the Postiz MCP client compatible with servers that support different MCP protocol versions by negotiating a mutually supported version during initialize and reusing it for subsequent requests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Client retries initialize with a supported MCP version when the server rejects the first version and reports supported versions
- [x] #2 Subsequent MCP tool calls use the negotiated protocol version and session ID
- [x] #3 If the session expires, the client reinitializes and retries the tool call once
- [x] #4 Automated tests cover version fallback and expired-session retry
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a regression test for MCP initialize protocol fallback
2. Implement initialize version negotiation and cache the accepted version
3. Add a regression test for expired session retry
4. Implement one reinitialize-and-retry path for expired sessions
5. Run focused and full Go test suites
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Plan saved to docs/superpowers/plans/2026-05-08-mcp-protocol-version-negotiation.md.

Implemented and verified MCP protocol fallback: initialize tries known versions newest-first, parses server supported versions from JSON-RPC error data/message, and stores the accepted protocol version for later tool calls. Verification: go test ./internal/tools passed.

Implemented expired-session recovery: a tool call that receives HTTP 404, or a 400 body containing "No valid session ID", clears cached MCP session state, reinitializes, and retries the same tool call once. Verification: go test ./internal/tools and go test ./... passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped MCP protocol version negotiation for Postiz, including fallback initialize attempts, negotiated protocol reuse, expired-session reinitialization, and regression tests. User confirmed task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
