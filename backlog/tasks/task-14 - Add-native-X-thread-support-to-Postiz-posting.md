---
id: TASK-14
title: Add native X thread support to Postiz posting
status: Done
assignee: []
created_date: '2026-05-09 04:54'
updated_date: '2026-05-09 05:13'
labels: []
dependencies: []
priority: high
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add explicit thread_parts support to Postiz create_post so multiple tweets are posted as one X/Twitter thread while preserving existing single-post and legacy auto-split behavior.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Explicit X thread_parts use one Postiz thread payload for MCP and Public API fallback
- [x] #2 Overlong explicit thread parts fail clearly before calling Postiz
- [x] #3 thread_parts is rejected for non-X platforms
- [x] #4 Legacy single-post and long X content auto-split behavior remains working
- [x] #5 go test ./... passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing tests for create_post thread_parts schema, MCP payload, Public API fallback payload, validation errors, and legacy auto-split behavior.
2. Implement thread_parts normalization and validation in internal/tools/postiz.go.
3. Update Postiz MCP and Public API payload builders to accept resolved post parts.
4. Update the system prompt and README documentation.
5. Run focused and full Go test suites, then update Backlog acceptance criteria and notes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED verified for native thread support: focused Postiz tests fail because create_post lacks thread_parts schema, explicit thread_parts are ignored, twitter does not resolve to x integration, and validation errors occur only after network calls.

Implemented native X thread support: create_post now accepts thread_parts, validates explicit thread items before network calls, sends one MCP/Public API Postiz thread payload, preserves legacy content auto-splitting, and updates the system prompt plus README. Verification: go test ./internal/tools -count=1 -v passed; go test ./... passed.

Final verification after test refactor: go test ./... passed; git diff --check passed. Task remains In Progress pending user manual confirmation before Done.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shipped native X/Twitter thread support for Postiz create_post via thread_parts, including MCP and Public API thread payloads, validation for explicit thread parts, prompt/README updates, and regression coverage. Verification: go test ./... passed before merge.
<!-- SECTION:FINAL_SUMMARY:END -->
