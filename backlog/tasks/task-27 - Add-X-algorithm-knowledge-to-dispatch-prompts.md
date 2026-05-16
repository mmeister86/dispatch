---
id: TASK-27
title: Add X algorithm knowledge to dispatch prompts
status: Done
assignee: []
created_date: '2026-05-16 11:10'
updated_date: '2026-05-16 17:35'
labels: []
dependencies: []
priority: high
ordinal: 27000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add curated X/Twitter algorithm guidance as repo Markdown and embed it into dispatch's agent prompt so X post drafts use it without runtime repo research.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Curated Markdown knowledge document summarizes X algorithm pipeline, positive and negative engagement signals, practical writing rubric, and caveat
- [x] #2 Agent system prompt includes the embedded guidance while preserving existing Postiz confirmation safety rules
- [x] #3 X/Twitter post previews are instructed to show a short Hook/Dwell/Reply/Repost/Risiko rubric
- [x] #4 Unit tests cover embedded knowledge and prompt builder behavior
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing tests for embedded X algorithm knowledge and system prompt behavior
2. Implement embedded Markdown knowledge package
3. Append guidance to dispatch system prompt while preserving Postiz safety rules
4. Run focused tests, full Go tests, and build
5. Check acceptance criteria but leave task In Progress for user confirmation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented embedded X algorithm knowledge document and prompt integration. Verified with focused red/green tests, go test ./..., and make build. Leaving task In Progress pending user confirmation.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified against code and tests. Curated X algorithm guidance is embedded as Markdown, appended to the system prompt, preserves Postiz confirmation safety, and instructs previews to include Hook/Dwell/Reply/Repost/Risiko.
<!-- SECTION:FINAL_SUMMARY:END -->
