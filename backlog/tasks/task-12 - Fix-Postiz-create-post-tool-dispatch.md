---
id: TASK-12
title: Fix Postiz create post tool dispatch
status: In Progress
assignee: []
created_date: '2026-05-08 21:25'
labels: []
dependencies: []
references:
  - TASK-9
priority: high
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Investigate and fix the Postiz create_post flow that currently reports `Unknown tool: schedulePostTool` after confirmed post creation attempts. Preserve the documented Postiz MCP flow from TASK-9 while making create_post call the actual exposed Postiz scheduling tool successfully.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 create_post no longer returns `Unknown tool: schedulePostTool` when scheduling through Postiz MCP
- [ ] #2 Postiz tool-name discovery or mapping matches the tools exposed by the configured Postiz MCP server
- [ ] #3 Automated tests cover the create_post scheduling path and the resolved Postiz MCP tool name
<!-- AC:END -->
