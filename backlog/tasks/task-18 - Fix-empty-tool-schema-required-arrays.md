---
id: TASK-18
title: Fix empty tool schema required arrays
status: Done
assignee: []
created_date: '2026-05-10 13:34'
updated_date: '2026-05-16 17:34'
labels: []
dependencies: []
priority: high
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
OpenAI rejects tools like list_channels when their JSON schema serializes required as null instead of an array.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tool schemas without required fields serialize required as an empty array, not null
- [x] #2 A regression test covers no-argument tool schemas
- [x] #3 Relevant Go tests pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a regression test for no-argument tool schemas
2. Verify the test fails on required: null
3. Change schema construction to emit an empty required array
4. Run focused and full relevant Go tests
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause: objectSchema left the variadic required slice nil for no-argument tools, which JSON marshaled as required:null. OpenAI expects required to be an array. Added regression coverage for JSON serialization and changed objectSchema to use an empty []string when no required fields are provided. Verification: go test ./internal/tools -count=1 and go test ./... -count=1 passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified against code and tests. Tool schemas without required fields serialize required as an empty array, with regression coverage and passing Go tests.
<!-- SECTION:FINAL_SUMMARY:END -->
