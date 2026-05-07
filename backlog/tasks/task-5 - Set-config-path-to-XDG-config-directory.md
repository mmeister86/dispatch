---
id: TASK-5
title: Set config path to XDG config directory
status: Done
assignee: []
created_date: '2026-05-07 18:49'
updated_date: '2026-05-07 19:17'
labels: []
dependencies: []
modified_files:
  - config/config.go
  - config/config_test.go
priority: high
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fix dispatch setup/config path resolution so the planned location ~/.config/dispatch/config.toml is used instead of macOS Application Support.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Setup reports ~/.config/dispatch/config.toml as the config target on macOS/Linux when no override is present
- [x] #2 Existing tests cover the config path behavior
- [x] #3 Relevant test suite passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Locate current config path resolution and setup display path
2. Add a failing test for the expected XDG config path
3. Change path resolution to ~/.config/dispatch/config.toml when no explicit config is passed
4. Run relevant Go tests and update acceptance criteria
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root cause confirmed: config.DefaultPath/newViper use os.UserConfigDir(), which returns ~/Library/Application Support on macOS. The plan/README expect ~/.config/dispatch/config.toml instead. Added failing test TestDefaultPathUsesDotConfigDirectory; it fails with the current macOS Application Support path.

Implemented defaultConfigDir() to use XDG_CONFIG_HOME when absolute, otherwise HOME/.config. DefaultPath(), DisplayPath(), Write(), and newViper now converge on ~/.config/dispatch/config.toml by default. Verified setup smoke output with HOME=/private/tmp/dispatch-test-home: Config-Ziel points to /private/tmp/dispatch-test-home/.config/dispatch/config.toml. Final verification: go test ./... passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
User confirmed this task can be closed.
<!-- SECTION:FINAL_SUMMARY:END -->
