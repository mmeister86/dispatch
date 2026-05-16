---
id: TASK-17
title: Add tag-based GoReleaser releases
status: Done
assignee: []
created_date: '2026-05-10 11:57'
updated_date: '2026-05-16 17:34'
labels: []
dependencies: []
priority: high
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Configure versioned releases so dispatch tags build reproducible cross-platform CLI packages and publish them through GitHub Releases.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Version tags matching v* trigger a GitHub Actions release workflow
- [x] #2 GoReleaser builds Linux, macOS, and Windows archives for amd64 and arm64 with version metadata embedded
- [x] #3 Release artifacts include checksums and are attached to a GitHub Release
- [x] #4 Repository documentation explains how to create a release tag
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add .goreleaser.yml for cross-platform dispatch archives with ldflags metadata and checksums.
2. Add .github/workflows/release.yml triggered by v* tags using goreleaser/goreleaser-action.
3. Update README/CONTRIBUTING/CHANGELOG/PLAN with the release process and actual current status.
4. Verify with go test ./..., make build, and GoReleaser config validation or snapshot build if the tool is available.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented GoReleaser v2 configuration, tag-triggered GitHub Actions release workflow, and release documentation. Verified with go test ./..., make build, goreleaser check, and a local goreleaser snapshot release that produced Linux/macOS/Windows amd64/arm64 archives plus checksums.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified against code and fresh checks. Tag-triggered GoReleaser releases, cross-platform archives, checksums, GitHub Releases, and release documentation are implemented.
<!-- SECTION:FINAL_SUMMARY:END -->
