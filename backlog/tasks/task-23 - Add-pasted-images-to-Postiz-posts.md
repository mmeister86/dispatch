---
id: TASK-23
title: Add pasted images to Postiz posts
status: In Progress
assignee: []
created_date: '2026-05-15 07:15'
updated_date: '2026-05-15 11:53'
labels: []
dependencies: []
priority: high
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement cross-platform image paste attachments in the dispatch TUI so pasted images are sent with user text through the existing agent and Postiz upload/create flow.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Ctrl+V captures pasted bitmap images and copied image files as pending attachments without breaking normal text paste
- [x] #2 Submitting text with attachments sends the agent local image paths and instructs it to upload media before create_post
- [x] #3 The UI shows pending image attachment state and allows clearing attachments
- [x] #4 README documents image paste behavior and platform fallbacks
- [x] #5 Go tests and build pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add clipboard package tests and implementation for cross-platform image extraction
2. Add TUI attachment state, paste handling, and submit prompt tests
3. Wire attachment instructions into system prompt and README
4. Verify go test ./..., make build, and git diff --check
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added internal/clipboard TDD coverage and implementation for macOS png/file paste, Linux URI/image commands, Windows PowerShell Clipboard output, and unavailable-tool errors. Verified with GOCACHE=/private/tmp/dispatch-go-build go test ./internal/clipboard.

Added TUI attachment tests and behavior for ctrl+v image attach, ctrl+x clear, text+image submit, image-only submit, and hidden agent upload_media/media_urls instructions. Updated the root system prompt and README image paste docs.

Verification completed: GOCACHE=/private/tmp/dispatch-go-build go test -count=1 ./... passed outside sandbox for httptest local ports; GOCACHE=/private/tmp/dispatch-go-build make build passed outside sandbox for Go module cache writes; git diff --check passed.

Final verification after filename polish: GOCACHE=/private/tmp/dispatch-go-build go test -count=1 ./... passed; GOCACHE=/private/tmp/dispatch-go-build make build passed; git diff --check passed.

Manual test feedback: user saw no confirmation after image paste. Root cause: successful image paste only updated the footer, and macOS bitmap paste without pngpaste was indistinguishable from no image. Added visible system confirmation on successful attach and a visible unavailable message when macOS clipboard contains bitmap data but pngpaste is missing.

Re-verified after paste confirmation fix: GOCACHE=/private/tmp/dispatch-go-build go test -count=1 ./... passed; GOCACHE=/private/tmp/dispatch-go-build make build passed; git diff --check passed.

Manual test feedback: cmd+v did not attach copied image files. Root cause: terminal cmd+v arrives as Bubble Tea bracketed pasted text (KeyRunes with Paste=true), while dispatch only handled KeyCtrlV. Added pasted image path/URI detection before textarea insertion so copied image file paths become attachments and normal pasted text still goes into the textarea.

Re-verified after cmd+v bracketed paste path handling: GOCACHE=/private/tmp/dispatch-go-build go test -count=1 ./... passed; GOCACHE=/private/tmp/dispatch-go-build make build passed; git diff --check passed.

Manual feedback: pngpaste should not be a required system dependency. Implemented bundled macOS clipboard bitmap support via AppKit/Foundation inside the dispatch binary, with pngpaste remaining only as an optional legacy fallback. README now documents that cmd+v is only available if the terminal forwards it to the TUI.

Re-verified after bundling macOS clipboard bitmap support: GOCACHE=/private/tmp/dispatch-go-build go test -count=1 ./... passed; GOCACHE=/private/tmp/dispatch-go-build make build passed; git diff --check passed.
<!-- SECTION:NOTES:END -->
