# Contributing

Thanks for helping improve `dispatch`. The project is a small Go CLI/TUI, so the
preferred contribution style is focused changes with tests close to the package
being changed.

## Development Loop

```bash
make test
make build
./bin/dispatch version
```

Run the app locally with:

```bash
go run .
```

For changes that affect configuration, tools, providers, or the TUI, also run:

```bash
go test ./...
```

CI runs the same test suite plus `make build`. Release archive settings live in
`.goreleaser.yml`.

## Release Process

Releases are tag-driven. Create an annotated semver-style tag, push it, and let
the GitHub Actions release workflow publish the GitHub Release:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

The workflow runs GoReleaser with `release --clean`. It builds Linux, macOS, and
Windows archives for amd64 and arm64, embeds version metadata through `ldflags`,
and uploads `checksums.txt` with the release artifacts.

## Task Tracking

This repository uses Backlog.md for non-trivial work. Create and update tasks
with the `backlog` CLI only; do not edit files under `backlog/tasks/` directly.

Typical flow:

```bash
backlog task create "Add focused behavior" \
  -s "In Progress" \
  -d "What is being implemented and why" \
  --ac "Observable acceptance criterion" \
  --priority high
```

Keep the task status in `In Progress` until the user explicitly confirms the
change after manual testing.

## Configuration and Secrets

Do not commit real config files or secrets. The repository includes
[config.toml.example](config.toml.example) with placeholder values only.

Local config lives at:

```text
~/.config/dispatch/config.toml
```

Local session state lives at:

```text
~/.local/state/dispatch/sessions/current.json
```

Both paths honor absolute `XDG_CONFIG_HOME` and `XDG_STATE_HOME` overrides.

## Adding Providers

Providers implement `internal/provider.Provider` and stream `provider.Chunk`
values. Keep provider-specific JSON mapping inside `internal/provider`, and
cover request formatting, streaming, tool calls, and error handling with
`httptest`.

Provider selection is wired through `provider.NewFromConfig`.

## Adding Tools

Tools implement `internal/tools.Tool`, expose JSON schemas through
`Definitions`, and execute by name with raw JSON arguments. Keep network clients
small, use explicit timeouts, and test API behavior with `httptest`.

Tools are registered in `internal/tools.NewRegistry`.

## Documentation

When behavior changes, update the top-level documentation in the same change:

- [README.md](README.md) for user-facing setup, commands, config, and workflows.
- [PLAN.md](PLAN.md) when the implementation status or architecture changes.
- [CHANGELOG.md](CHANGELOG.md) for notable unreleased changes.
- [config.toml.example](config.toml.example) when config keys or defaults change.
