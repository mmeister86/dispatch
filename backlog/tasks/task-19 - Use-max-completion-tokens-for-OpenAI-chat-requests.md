---
id: TASK-19
title: Use max completion tokens for OpenAI chat requests
status: Done
assignee: []
created_date: '2026-05-10 13:36'
updated_date: '2026-05-16 17:34'
labels: []
dependencies: []
priority: high
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
OpenAI models can reject chat completion requests that send max_tokens; the provider should send max_completion_tokens for OpenAI chat completions while preserving request behavior.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 OpenAI-compatible request JSON uses max_completion_tokens instead of max_tokens
- [x] #2 Regression coverage verifies the serialized request parameter name
- [x] #3 Relevant Go tests pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a regression test for OpenAI request JSON token parameter naming
2. Verify the test fails on max_tokens
3. Update the OpenAI request payload to emit max_completion_tokens
4. Run focused provider tests and the full suite
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause: the OpenAI-compatible Chat Completions payload still serialized Request.MaxTokens as max_tokens. Newer OpenAI models reject that field and require max_completion_tokens. Added regression coverage against the raw JSON request and updated openAIRequest to emit max_completion_tokens. Verification: go test ./internal/provider -count=1 and go test ./... -count=1 passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified against code and tests. OpenAI-compatible requests now serialize max_completion_tokens instead of max_tokens, with regression coverage and passing Go tests.
<!-- SECTION:FINAL_SUMMARY:END -->
