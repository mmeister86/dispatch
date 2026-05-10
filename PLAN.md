# dispatch - Projektplan und aktueller Stand

`dispatch` ist ein CLI/TUI-Assistent fuer Developer Social-Media-Planung. Der
aktuelle Durchstich umfasst lokale Konfiguration, Streaming-Provider, einen
Agent-Loop mit Tools, eine Bubble Tea TUI, Postiz MCP Scheduling, GitHub-Kontext,
Perplexity-Recherche und persistierte Sessions.

## Tech-Stack

| Schicht | Paket / Tool | Zweck |
| --- | --- | --- |
| CLI | `github.com/spf13/cobra` | Root-Command, `run`, `check`, `version`, `--setup` |
| Config | `github.com/spf13/viper` | TOML, Defaults, Env-Overrides |
| TUI | `github.com/charmbracelet/bubbletea` | Elm-Architektur fuer Terminal-UI |
| TUI Components | `github.com/charmbracelet/bubbles` | Textarea, Viewport, Spinner |
| Styling | `github.com/charmbracelet/lipgloss` | Layout, Farben, Status-Anzeigen |
| HTTP | `net/http` | Provider-, GitHub-, Search- und Postiz-Clients |
| State | JSON-Dateien | Lokale Session-Persistenz unter XDG-State |

## Architektur

```text
TUI
  |
  v
Agent Core
  |-- Provider: Anthropic / OpenAI / OpenRouter / xAI / Gemini
  |
  |-- Tools
      |-- GitHub REST API
      |-- Perplexity Sonar
      |-- Postiz MCP, mit Public-API-Fallback fuer Scheduling
```

Der Agent haelt eine begrenzte History, streamt Text-Chunks an die TUI und
fuehrt Tool-Calls in bis zu sechs Runden aus. Tool-Ergebnisse werden wieder in
die History geschrieben, damit der Provider eine finale Antwort erzeugen kann.

## Projektstruktur

```text
dispatch/
├── cmd/
│   └── root.go                 # Cobra commands, setup, TUI bootstrap
├── config/
│   └── config.go               # Defaults, TOML, Env, XDG paths, validation
├── internal/
│   ├── agent/
│   │   └── agent.go            # Streaming loop, tool rounds, history
│   ├── provider/
│   │   ├── anthropic.go
│   │   ├── gemini.go
│   │   ├── openai.go
│   │   └── provider.go
│   ├── session/
│   │   └── session.go          # JSON session store and summaries
│   ├── tools/
│   │   ├── github.go
│   │   ├── postiz.go
│   │   ├── search.go
│   │   └── tools.go
│   └── tui/
│       ├── app.go
│       └── styles.go
├── docs/
│   ├── superpowers/plans/
│   └── wireframes.html
├── config.toml.example
├── Makefile
├── README.md
└── CHANGELOG.md
```

## Config und State

Config wird aus `~/.config/dispatch/config.toml` gelesen, oder aus
`$XDG_CONFIG_HOME/dispatch/config.toml`, wenn `XDG_CONFIG_HOME` absolut gesetzt
ist. Die Reihenfolge ist:

```text
environment variables > config.toml > defaults
```

Sessions werden automatisch unter `~/.local/state/dispatch/sessions/current.json`
gespeichert, oder unter `$XDG_STATE_HOME/dispatch/sessions/current.json`.
Zusaetzliche Session-Dateien liegen im selben Ordner und koennen in der TUI ueber
`/session`, `/session new`, `/session open <id>` und `/session delete <id>`
verwaltet werden.

## Provider

Implementiert:

- Anthropic native Messages API
- OpenAI-compatible Chat Completions fuer OpenAI, OpenRouter und xAI
- Gemini `streamGenerateContent`

Alle Provider implementieren `internal/provider.Provider`:

```go
type Provider interface {
    Complete(ctx context.Context, req Request) (<-chan Chunk, error)
    Name() string
    ModelID() string
}
```

## Tools

### GitHub

Implementierte Tool-Namen:

- `get_recent_commits`
- `get_activity`
- `get_file`

`get_activity` liefert aktuell Commits plus einen Hinweis, dass PR-/Issue-
Aggregation vorbereitet ist. Leere Repositories werden als erwartbarer Zustand
behandelt.

### Search

Implementierter Tool-Name:

- `search`

Die aktuelle Implementierung nutzt Perplexity Sonar unter
`https://api.perplexity.ai/chat/completions` und gibt Antwort plus Quellenliste
zurueck. `SEARCH_API_KEY` ist optional; ohne Key meldet das Tool den fehlenden
Key zur Laufzeit.

### Postiz

Implementierte Tool-Namen:

- `list_channels`
- `create_post`

Postiz wird ueber MCP angesprochen. Der Client handelt eine kompatible
`MCP-Protocol-Version` aus, cached die Session-ID und erneuert sie bei
abgelaufenen Sessions. Scheduling nutzt `schedulePostTool`; wenn dieses Tool auf
einer Instanz nicht verfuegbar ist, faellt `create_post` auf die Public API
zurueck.

Fuer X/Twitter-Threads akzeptiert `create_post` `thread_parts`. Jeder Eintrag
wird ein Tweet im selben Thread. `confirmed=true` ist Pflicht.

## TUI

Wichtige Interaktionen:

| Shortcut / Command | Aktion |
| --- | --- |
| `ctrl+p` | Postiz-Kanaele auflisten |
| `ctrl+r` | GitHub-Aktivitaet der letzten 7 Tage abrufen |
| `ctrl+s` | Aktuelle Developer-Social-Themen recherchieren |
| `ctrl+l` | Aktuellen Chat und aktive Session leeren |
| `ctrl+c` | Laufenden Agent-Call abbrechen, sonst beenden |
| `ctrl+q` | Beenden |
| `?` | Hilfe anzeigen |
| `/session` | Sessions auflisten |
| `/session new` | Neue Session starten |
| `/session open <id>` | Session oeffnen |
| `/session delete <id>` | Nicht-aktive Session loeschen |

## Umsetzungsstatus

| Phase | Status | Hinweise |
| --- | --- | --- |
| Projektgeruest | Erledigt | Cobra, Config, Makefile, Tests, TUI-Basis |
| Provider | Erledigt | Anthropic, OpenAI-compatible, Gemini |
| Agent Core | Erledigt | Streaming, History, Tool-Runden |
| GitHub Tool | Teilweise | Commits und Datei-Lesen implementiert; PR-/Issue-Aggregation noch ausbauen |
| Search Tool | Erledigt | Perplexity/Sonar mit Citations |
| Postiz Tool | Erledigt | MCP, Protocol Negotiation, Public-API-Fallback, X-Threads |
| Sessions | Erledigt | Auto-Resume, Session-Liste, Open/New/Delete |
| Release-Vorbereitung | Teilweise | CI, GoReleaser-Konfig und Tag-Workflow vorhanden; erster echter Release noch validieren |

## Naechste sinnvolle Schritte

- GitHub `get_activity` um echte PR- und Issue-Aggregation erweitern.
- `dispatch check` optional detaillierter machen, z.B. Search separat warnen.
- Ersten echten GoReleaser-Tag-Release ausfuehren und Release-Artefakte validieren.
- README mit Screenshots oder einer kurzen TUI-Aufnahme erweitern, sobald das UI
  stabil ist.
