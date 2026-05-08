---
id: TASK-9
title: Fix Postiz Public API authentication
status: In Progress
assignee: []
created_date: '2026-05-08 20:55'
updated_date: '2026-05-08 21:03'
labels: []
dependencies: []
priority: high
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Update the Postiz tool to call the current Public API endpoints and send authorization in the format Postiz expects, resolving 401 responses from list_channels.
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
<!-- SECTION:NOTES:END -->
