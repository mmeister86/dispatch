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
- Postiz MCP scheduling, channel listing, native X/Twitter thread support, and
  native Public API post management for self-hosted instances.
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
- Postiz: `list_channels`, `create_post`, `list_posts`, `delete_post`,
  `set_post_status`, `upload_media`, `get_platform_analytics`,
  `get_post_analytics`, `list_missing_post_content`, `connect_post_release`

The TUI includes shortcuts for common tool prompts:

| Shortcut | Action |
| --- | --- |
| `ctrl+p` | list Postiz posts from the last 30 days through the next 30 days |
| `ctrl+o` | list connected Postiz accounts/providers |
| `ctrl+r` | summarize GitHub activity from the last 7 days |
| `ctrl+s` | research current developer social topics with citations |
| `ctrl+v` | attach image files or a clipboard image to the current prompt |
| `cmd+v` | attach images on macOS terminals that forward Command as Super |
| `ctrl+x` | clear pending image attachments |
| `ctrl+l` | clear the current chat/session |
| `ctrl+c` | cancel a running agent call, or quit when idle |
| `ctrl+q` | quit |
| `?` | toggle help |

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
/debug keys           # toggle non-persistent key event logs in the chat
```

Session directories are written with `0700` permissions and session files with
`0600` permissions.

## Postiz and X/Twitter Threads

`create_post` requires `confirmed=true`; the agent is instructed to show a
preview before scheduling anything.

Image paste is supported in the TUI with `ctrl+v`. Pasted image files or a
clipboard bitmap are added as pending attachments and sent with the next user
message. The agent uploads each local file through `upload_media` and then uses
the returned media paths in `create_post.media_urls`; the normal preview and
explicit confirmation gate still applies before a Postiz post is created.

On startup, dispatch asks compatible terminals to forward enhanced keyboard
events so `cmd+v` can be detected as `Super+V` on macOS. Terminals that consume
Command shortcuts internally may still hide the key event from TUI programs; in
that case, `ctrl+v` remains the reliable image-paste shortcut. For raw clipboard
images, macOS bitmap support is built into the dispatch binary; Linux uses
`wl-paste`, `xclip`, or `xsel`; Windows uses PowerShell's Clipboard APIs. Use
`ctrl+x` before sending to clear pending image attachments.

To diagnose terminal paste behavior, run `/debug keys` in the TUI or start with
`DISPATCH_DEBUG_KEYS=1 dispatch`. When enabled, dispatch prints non-persistent
debug lines for key events and unknown terminal CSI sequences. If pressing
`cmd+v` produces no debug line, the terminal did not forward the event to
dispatch.

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

`list_posts`, `delete_post`, `set_post_status`, `upload_media`, analytics, and
missing-release helpers use the Postiz Public API directly. Destructive actions
require an explicit confirmation before the agent can call them.

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

## Releases

Versioned releases are created from Git tags named `v*`, for example `v0.1.0`.
Pushing such a tag starts the release workflow, which runs GoReleaser and
attaches Linux, macOS, and Windows archives plus `checksums.txt` to a GitHub
Release.

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

GoReleaser injects the tag version, commit SHA, and build date into
`dispatch version`.

The original TUI wireframe reference lives at
[docs/wireframes.html](docs/wireframes.html).
