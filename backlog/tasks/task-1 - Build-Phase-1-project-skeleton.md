---
id: TASK-1
title: Build Phase 1 project skeleton
status: Done
assignee: []
created_date: '2026-05-07 12:26'
updated_date: '2026-05-07 13:05'
labels: []
dependencies: []
priority: high
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement Phase 1 from PLAN.md, corrected to the current product name dispatch: scaffold the Go dispatch CLI/TUI, configuration loading, and setup flow based on the provided wireframes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Go module, CLI entrypoint, and project structure exist and build
- [x] #2 Cobra run command starts a minimal Bubble Tea TUI with header, chat viewport, input footer, and shortcuts matching the wireframes
- [x] #3 Configuration loads from defaults, config file, and environment variables, with clear validation errors
- [x] #4 Setup command can collect keys, mask secret input, and write config.toml to the user config directory
- [x] #5 Core files include a safe config example, gitignore, README, and docs wireframe reference
- [x] #6 go test ./... passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Scaffold Go module and repo files
2. Implement config loading, validation, example config, and setup writer
3. Implement Cobra commands for run, --setup, --check, and --version
4. Implement minimal Bubble Tea TUI matching the provided wireframes
5. Add focused tests and run gofmt/go test ./...
6. Record verified acceptance criteria and leave task open for user confirmation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented Phase 1 scaffold for dispatch: Go module, Cobra CLI, Viper config, setup writer, Bubble Tea TUI skeleton, README/config example/gitignore, and docs wireframe copy.

User correction applied: product/tool name is dispatch, not the outdated plan name. Renamed module path, CLI use/version/setup strings, config directory, TUI labels, README, PLAN.md, Backlog project name, and wireframes to dispatch.

Verification after rename: go test ./... passed; go run . version returned dispatch dev; setup wrote /private/tmp/dispatch-test.toml; go run . check --config /private/tmp/dispatch-test.toml passed. Task remains In Progress pending user manual confirmation before Done.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Phase 1 shipped and manually accepted: dispatch Go scaffold, Cobra CLI, Viper config/setup flow, minimal Bubble Tea TUI, README/config example/gitignore, and verified tests/build smoke checks.
<!-- SECTION:FINAL_SUMMARY:END -->
