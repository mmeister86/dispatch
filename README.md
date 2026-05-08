# dispatch

`dispatch` is a terminal-first assistant for developer social media planning.
It is currently in Phase 1: a Go CLI, local configuration, and a minimal Bubble Tea TUI based on the provided wireframes.

## Quickstart

```bash
make build
./bin/dispatch --setup
./bin/dispatch
```

During development you can also run directly:

```bash
go run . --setup
go run .
```

Configuration is loaded from `~/.config/dispatch/config.toml`, then environment variables, then defaults.

To make `dispatch` available on your shell `PATH`, run:

```bash
make install
dispatch version
```

## Commands

```bash
dispatch run
dispatch --setup
dispatch check
dispatch version
```

## Environment

- `LLM_API_KEY`
- `POSTIZ_API_KEY`
- `GITHUB_TOKEN`
- `SEARCH_API_KEY`

## LLM Providers

Supported streaming providers:

- `anthropic`
- `openai`
- `openrouter`
- `xai`
- `gemini`

`openai`, `openrouter`, and `xai` use the same OpenAI-compatible chat completions streaming adapter with provider-specific base URLs.

## Tools

Configured providers receive tool definitions for:

- GitHub: recent commits, combined activity, and file reads
- Search: Perplexity/Sonar web research with citations
- Postiz MCP: list channels and create scheduled posts

Use `ctrl+r` for repo activity, `ctrl+p` for Postiz channels, and `ctrl+s` for research from the TUI.

The committed `config.toml.example` contains placeholders only. Real secrets belong in the user config directory or environment.

## Wireframes

The original TUI reference lives at [docs/wireframes.html](docs/wireframes.html).
