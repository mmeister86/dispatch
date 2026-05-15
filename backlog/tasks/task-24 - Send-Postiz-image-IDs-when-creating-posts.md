---
id: TASK-24
title: Send Postiz image IDs when creating posts
status: In Progress
assignee: []
created_date: '2026-05-15 12:51'
updated_date: '2026-05-15 12:55'
labels: []
dependencies: []
priority: high
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fix the Postiz posting flow so uploaded images are passed to create_post with their required image IDs instead of URL-only media payloads.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Creating an X post with an uploaded image sends media with a non-empty image id
- [x] #2 The user is not asked for a second confirmation after a media payload formatting error that can be resolved from the existing upload response
- [x] #3 Targeted tests cover the Postiz image media payload
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Trace the uploaded image metadata from upload response to create_post payload
2. Add a failing test that captures Postiz image IDs in the media payload
3. Implement the smallest payload fix
4. Run targeted verification and update acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause evidence: runtime Postiz validation rejected `image.0.id` as missing. Local code only accepts `media_urls` and builds Public API images as `{path: mediaURL}`. ctx7 Postiz docs confirm upload returns `id` + `path`, and create-post image arrays must include both `id` and `path`.

Implemented create_post.media support for uploaded Postiz media objects, preserving id and path in Public API image payloads and using paths for the legacy MCP attachment shape. Updated system/TUI prompts and README to instruct agents to use create_post.media after upload_media. Verification: GOCACHE=/private/tmp/dispatch-go-build go test ./internal/tools -count=1; go test ./internal/tui -count=1; go test ./... -count=1 all passed.
<!-- SECTION:NOTES:END -->
