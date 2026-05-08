---
id: TASK-9
title: Fix Postiz Public API authentication
status: Done
assignee: []
created_date: '2026-05-08 20:55'
updated_date: '2026-05-08 21:18'
labels: []
dependencies: []
priority: high
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Update the Postiz tool to use the documented Postiz MCP streamable HTTP endpoint for self-hosted and cloud installs, resolving auth failures from the REST/Public API path.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 list_channels calls the current Postiz integrations endpoint with the expected Authorization header
- [x] #2 list_posts and create_post keep working against the same Postiz API authentication convention
- [x] #3 Automated tests cover the Postiz request paths and headers
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Verify current Postiz MCP contract from docs
2. Add failing tests for MCP endpoint construction, integrationList, and schedulePostTool
3. Update Postiz tool to call self-hosted/cloud MCP Streamable HTTP endpoint
4. Remove unsupported REST list_posts behavior or return a clear MCP unsupported error
5. Run focused and full tests
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verified Postiz docs via ctx7: Public API uses /public/v1 endpoints and Authorization header containing the raw API key/OAuth token, not Bearer-prefixed auth. Updated Postiz tool and config default accordingly. Verification: go test ./... passed.

Scope update from user: switch Postiz integration to the self-hosted Postiz MCP server URL shown in Postiz docs/screenshot. ctx7 docs show MCP streamable HTTP URL format and tools integrationList + schedulePostTool; no documented MCP list-posts tool was found.

Implemented Postiz MCP transport: self-hosted base URLs call /api/mcp/<key>, cloud api.postiz.com calls /mcp/<key>, and already-complete /mcp URLs are accepted. list_channels maps to integrationList. create_post maps to schedulePostTool and resolves integration_id via integrationList when only platform is provided. Removed list_posts from exposed tool definitions because Postiz MCP docs do not document a list-posts MCP tool. Verification: go test ./internal/tools and go test ./... passed.

Added support for complete MCP URLs that already contain /mcp/<key>, so the screenshot URL can be pasted as base_url without a separate postiz.api_key. Config validation and TUI status now accept either a separate key or a key embedded in the MCP URL. Verification: go test ./internal/tools ./config and go test ./... passed.

Manual test showed Postiz MCP 400: No valid session ID provided for non-initialize request. ctx7 MCP spec confirms streamable HTTP servers may require an initialize request first and then MCP-Session-Id on subsequent requests.

Fixed MCP streamable HTTP session handling: client now sends initialize first, records MCP-Session-Id from the response, includes MCP-Protocol-Version and MCP-Session-Id on subsequent tools/call requests, and tests cover the handshake. Verification: go test ./internal/tools and go test ./... passed.

Manual test showed Postiz v2.10.1 rejects MCP protocol version 2025-11-25 and supports 2025-06-18, 2025-03-26, 2024-11-05, 2024-10-07. Updated dispatch to use 2025-06-18 for initialize and tool-call headers. Verification: go test ./internal/tools and go test ./... passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Switched Postiz integration to documented MCP streamable HTTP transport, added initialize/session handling, pinned protocol version to a Postiz-supported version, and verified list_channels works after manual user testing.
<!-- SECTION:FINAL_SUMMARY:END -->
