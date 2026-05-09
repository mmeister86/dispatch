# dispatch

`dispatch` is a terminal-first assistant for developer social media planning. It
combines a Bubble Tea TUI, streaming LLM providers, GitHub context, Perplexity
research, and Postiz scheduling in one local CLI.

The project is BYOK: API keys are read from a local config file or environment
variables and are not committed to the repository.

## Features

- Streaming chat UI in the terminal with persisted local sessions.
- Anthropic, OpenAI, OpenRouter, xAI, and Gemini provider adapters.
- GitHub tools for recent commits, combined activity, and repository file reads.
- Perplexity/Sonar search with citations.
- Postiz MCP scheduling, channel listing, and native X/Twitter thread support.
- Local build, install, test, and version targets through `make`.

## Quickstart

```bash
make build
./bin/dispatch --setup
./bin/dispatch
```

During development you can run the TUI directly:

```bash
go run .
```

Install the command into `GOPATH/bin`:

```bash
make install
dispatch version
```

## Commands

```bash
dispatch             # start the TUI
dispatch run         # start the TUI
dispatch --setup     # write a local config.toml interactively
dispatch check       # validate required configuration
dispatch version     # print version, commit, and build date
```

You can point the CLI at a custom config file with `--config`:

```bash
dispatch --config ./config.toml check
```

## Configuration

Default config path:

```text
~/.config/dispatch/config.toml
```

If `XDG_CONFIG_HOME` is set to an absolute path, dispatch uses:

```text
$XDG_CONFIG_HOME/dispatch/config.toml
```

Configuration precedence is:

```text
environment variables > config.toml > defaults
```

The committed [config.toml.example](config.toml.example) contains placeholders
only. Real secrets belong in the user config directory or environment.

### Environment Variables

| Variable | Config key | Required |
| --- | --- | --- |
| `LLM_API_KEY` | `llm.api_key` | yes |
| `LLM_PROVIDER` | `llm.provider` | no |
| `LLM_MODEL` | `llm.model` | no |
| `POSTIZ_API_KEY` | `postiz.api_key` | yes, unless `postiz.base_url` is already an MCP URL |
| `POSTIZ_BASE_URL` | `postiz.base_url` | no |
| `GITHUB_TOKEN` | `github.token` | yes for GitHub tools |
| `SEARCH_API_KEY` | `search.api_key` | optional |
| `SEARCH_PROVIDER` | `search.provider` | optional, currently Perplexity-oriented |
| `SEARCH_MODEL` | `search.model` | optional |
| `SEARCH_RECENCY` | `search.recency` | optional |

`dispatch check` validates the LLM, Postiz, and GitHub values. Search is optional;
if no search key is configured, the search tool reports that at runtime.

## Providers

Supported provider names:

- `anthropic`
- `openai`
- `openrouter`
- `xai`
- `gemini`

`openai`, `openrouter`, and `xai` share the OpenAI-compatible chat completions
streaming adapter with provider-specific base URLs.

## Tools

Configured providers receive these tool definitions:

- GitHub: `get_recent_commits`, `get_activity`, `get_file`
- Search: `search` via Perplexity/Sonar
- Postiz: `list_channels`, `create_post`

The TUI includes shortcuts for common tool prompts:

| Shortcut | Action |
| --- | --- |
| `ctrl+p` | list connected Postiz channels |
| `ctrl+r` | summarize GitHub activity from the last 7 days |
| `ctrl+s` | research current developer social topics with citations |
| `ctrl+l` | clear the current chat/session |
| `ctrl+c` | cancel a running agent call, or quit when idle |
| `ctrl+q` | quit |
| `?` | toggle help |
| `up` | reuse the last input when the prompt is empty |

## Sessions

Chats are saved locally and restored automatically.

Default session path:

```text
~/.local/state/dispatch/sessions/current.json
```

If `XDG_STATE_HOME` is set to an absolute path, dispatch uses:

```text
$XDG_STATE_HOME/dispatch/sessions/current.json
```

Inside the TUI:

```text
/session              # list saved sessions
/session new          # start a new session
/session open <id>    # open a saved session
/session delete <id>  # delete a non-active session
```

Session directories are written with `0700` permissions and session files with
`0600` permissions.

## Postiz and X/Twitter Threads

`create_post` requires `confirmed=true`; the agent is instructed to show a
preview before scheduling anything.

For X/Twitter threads, pass `thread_parts`. Each array item becomes one tweet in
the same Postiz thread:

```json
{
  "platform": "x",
  "thread_parts": ["First tweet", "Second tweet", "Final tweet"],
  "scheduled_at": "2026-05-08T09:00:00Z",
  "confirmed": true
}
```

Postiz defaults to `https://api.postiz.com`. For self-hosted instances, set
`postiz.base_url` or `POSTIZ_BASE_URL`. Direct MCP URLs containing `/mcp/` are
also supported.

## Development

```bash
make test
make build
./bin/dispatch version
```

Useful targets:

```bash
make help
make run
make clean
```

CI runs `go test ./...` and `make build` on push and pull requests. Release
archives are configured through GoReleaser for Linux, macOS, and Windows on
amd64 and arm64.

The original TUI wireframe reference lives at
[docs/wireframes.html](docs/wireframes.html).
