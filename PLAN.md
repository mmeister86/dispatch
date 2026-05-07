# dispatch — Implementierungsplan

> CLI-Chatbot mit TUI zur KI-gestützten Social Media-Planung.  
> Zugriff auf GitHub-Repos, Web-Recherche via Perplexity Sonar,  
> Scheduling via Postiz API. Geschrieben in Go, BYOK, Open Source.

---

## Inhaltsverzeichnis

1. [Tech-Stack](#1-tech-stack)
2. [Architektur](#2-architektur)
3. [Projektstruktur](#3-projektstruktur)
4. [Config & Secrets](#4-config--secrets)
5. [Provider-Interface (BYOK)](#5-provider-interface-byok)
6. [Agent Core](#6-agent-core)
7. [Tools](#7-tools)
   - [Postiz](#71-postiz-tool)
   - [GitHub](#72-github-tool)
   - [Search / Perplexity Sonar](#73-search-tool--perplexity-sonar)
8. [TUI (bubbletea)](#8-tui-bubbletea)
9. [Implementierungsphasen](#9-implementierungsphasen)
10. [Open-Source-Checkliste](#10-open-source-checkliste)

---

## 1. Tech-Stack

| Schicht        | Paket / Tool                          | Zweck                                      |
|----------------|---------------------------------------|--------------------------------------------|
| TUI            | `github.com/charmbracelet/bubbletea`  | Elm-Architektur für das Terminal-UI        |
| TUI Styling    | `github.com/charmbracelet/lipgloss`   | Farben, Borders, Layout                    |
| TUI Components | `github.com/charmbracelet/bubbles`    | Textarea, Viewport, Spinner, TextInput     |
| HTTP           | `net/http` (stdlib)                   | API-Calls zu allen Diensten                |
| Config         | `github.com/spf13/viper`              | config.toml + Env-Vars + CLI-Flags         |
| CLI-Flags      | `github.com/spf13/cobra`              | `run`, `--setup`, `--version`              |
| Go-Version     | Go 1.22+                              | Range-over-func, improved toolchain        |

Keine ORM, keine Datenbank, keine schweren Frameworks. Alles bleibt so stdlib-nah wie möglich.

---

## 2. Architektur

```
┌─────────────────────────────────────────┐
│              TUI layer                  │
│      bubbletea · lipgloss · bubbles     │
└──────────────────┬──────────────────────┘
                   │ Msg / Cmd
┌──────────────────▼──────────────────────┐
│             Agent Core                  │
│     tool_use loop · streaming           │◄──── LLM Provider (BYOK)
└──────┬───────────┬──────────────┬───────┘      Anthropic / OpenAI /
       │           │              │               Gemini / OpenRouter / xAI
┌──────▼──┐  ┌─────▼──┐  ┌───────▼──────┐
│ Postiz  │  │ GitHub │  │    Search    │
│  Tool   │  │  Tool  │  │    Tool      │
└──────┬──┘  └─────┬──┘  └───────┬──────┘
       │           │              │
  Postiz API  GitHub API    Perplexity Sonar
  (REST)      (REST)        (sonar-pro)
                            ± Brave / SerpAPI
```

### Datenfluss

1. User tippt Eingabe → bubbletea sendet `UserInputMsg`
2. Agent Core ruft LLM Provider mit Nachrichten-History + Tool-Definitionen auf
3. LLM antwortet mit Text (streaming) **oder** einem Tool-Call
4. Bei Tool-Call: Agent führt das entsprechende Tool aus, fügt Result zur History hinzu
5. Schleife bis LLM eine finale Textantwort ohne Tool-Call liefert
6. TUI rendert den gestreamten Text in den Viewport

---

## 3. Projektstruktur

```
dispatch/
│
├── cmd/
│   └── root.go                  # Cobra root command: run + --setup + --version
│
├── internal/
│   │
│   ├── tui/
│   │   ├── app.go               # bubbletea Model, Init, Update, View
│   │   ├── viewport.go          # Chat-Verlauf (scrollbar, Message-Rendering)
│   │   ├── input.go             # Eingabefeld, Keyboard-Shortcuts
│   │   ├── statusbar.go         # Header: App-Name, Provider, Integration-Status
│   │   └── styles.go            # lipgloss Styles (zentral, kein Inline-Styling)
│   │
│   ├── agent/
│   │   ├── agent.go             # Haupt-Loop: tool_use Schleife, History-Management
│   │   ├── tools.go             # Tool-Definitionen im internen Format
│   │   └── converter.go         # Übersetzt internes Format ↔ Provider-Format
│   │
│   ├── provider/
│   │   ├── provider.go          # Interface + gemeinsame Typen
│   │   ├── anthropic.go         # Anthropic native API
│   │   ├── openai.go            # OpenAI API (auch für OpenRouter + xAI)
│   │   └── gemini.go            # Google Gemini API
│   │
│   └── tools/
│       ├── postiz.go            # Postiz REST API Wrapper
│       ├── github.go            # GitHub API Wrapper
│       └── search.go            # Search-Interface + Perplexity Backend
│
├── config/
│   └── config.go                # Viper-Setup, Validierung, Pfade
│
├── main.go                      # Einstiegspunkt
│
├── config.toml.example          # Template mit Platzhaltern (committed)
├── .gitignore                   # config.toml, .env, Binaries
├── go.mod
├── go.sum
├── LICENSE                      # MIT
├── README.md
└── docs/
    └── wireframes.html          # TUI-Wireframes (Design-Referenz)
```

---

## 4. Config & Secrets

### Speicherort

```
~/.config/dispatch/config.toml   ← nie im Repo
```

### `config.toml.example` (committed)

```toml
[llm]
provider = "anthropic"        # anthropic | openai | gemini | openrouter | xai
model    = "claude-sonnet-4-6"
api_key  = ""                 # oder Env-Var: LLM_API_KEY

[postiz]
api_key  = ""                 # oder Env-Var: POSTIZ_API_KEY
base_url = "https://app.postiz.com"   # self-hosted: eigene URL eintragen

[github]
token = ""                    # oder Env-Var: GITHUB_TOKEN
                              # Scopes: repo (read), metadata

[search]
provider = "perplexity"       # perplexity | brave | serpapi
api_key  = ""                 # oder Env-Var: SEARCH_API_KEY
model    = "sonar-pro"        # sonar | sonar-pro | sonar-reasoning | sonar-reasoning-pro
recency  = "week"             # hour | day | week | month (default für alle Suchanfragen)
```

### Prioritätsreihenfolge (Viper)

```
Env-Variable  >  config.toml  >  Default-Wert
```

Env-Variablen werden automatisch gemappt:
`LLM_API_KEY` → `llm.api_key`, `GITHUB_TOKEN` → `github.token`, usw.

### Validierung beim Start

`config.go` prüft beim Start alle Pflicht-Keys. Fehlende Keys erzeugen eine klare Fehlermeldung — kein Panic, kein Stack-Trace:

```
✗ LLM API Key fehlt.
  Setze LLM_API_KEY als Umgebungsvariable oder führe --setup aus.
```

### `--setup` Flag

Startet einen geführten Konfigurations-Dialog im Terminal:
- Provider-Auswahl (interaktive Liste)
- Key-Eingabe (masked)
- Verbindungstest (ein Test-API-Call)
- Speicherung in `~/.config/dispatch/config.toml`

---

## 5. Provider-Interface (BYOK)

### Interface

```go
// internal/provider/provider.go

type Provider interface {
    Complete(ctx context.Context, req Request) (<-chan Chunk, error)
    Name() string
    ModelID() string
}

type Request struct {
    System   string
    Messages []Message
    Tools    []ToolDefinition
    MaxTokens int
}

type Chunk struct {
    Text       string  // gestreamter Text-Anteil
    ToolCall   *ToolCall
    StopReason string  // "end_turn" | "tool_use"
    Err        error
}

type Message struct {
    Role    string // "user" | "assistant"
    Content []ContentBlock
}

type ContentBlock struct {
    Type       string // "text" | "tool_use" | "tool_result"
    Text       string
    ToolUse    *ToolUse
    ToolResult *ToolResult
}
```

### Implementierungen

#### Anthropic (`anthropic.go`)
- Endpoint: `https://api.anthropic.com/v1/messages`
- Eigenes JSON-Format für Messages und Tools
- Streaming via Server-Sent Events (`text_delta`, `tool_use`)
- Header: `anthropic-version: 2023-06-01`, `x-api-key`

#### OpenAI (`openai.go`)
- Endpoint: `https://api.openai.com/v1/chat/completions`
- OpenAI-Format: `function_call` / `tool_calls`
- Streaming via SSE (`data: {...}`)
- Auch genutzt für **OpenRouter** (`https://openrouter.ai/api/v1`) und **xAI** (`https://api.x.ai/v1`) — nur `BaseURL` und `AuthHeader` abweichend

#### Gemini (`gemini.go`)
- Endpoint: `https://generativelanguage.googleapis.com/v1beta/models/{model}:streamGenerateContent`
- Eigenes Format: `contents`, `parts`, `functionCall`
- Streaming via chunked JSON

### Factory

```go
// config.go → provider wird einmalig beim Start instanziiert

func NewProvider(cfg Config) (provider.Provider, error) {
    switch cfg.LLM.Provider {
    case "anthropic":
        return provider.NewAnthropic(cfg.LLM.APIKey, cfg.LLM.Model)
    case "openai":
        return provider.NewOpenAI(cfg.LLM.APIKey, cfg.LLM.Model, "")
    case "openrouter":
        return provider.NewOpenAI(cfg.LLM.APIKey, cfg.LLM.Model, "https://openrouter.ai/api/v1")
    case "xai":
        return provider.NewOpenAI(cfg.LLM.APIKey, cfg.LLM.Model, "https://api.x.ai/v1")
    case "gemini":
        return provider.NewGemini(cfg.LLM.APIKey, cfg.LLM.Model)
    default:
        return nil, fmt.Errorf("unbekannter Provider: %s", cfg.LLM.Provider)
    }
}
```

---

## 6. Agent Core

### Aufgaben

- Nachrichten-History verwalten
- Tool-Definitionen an den LLM übergeben
- Tool-Use-Schleife ausführen (mehrere Runden möglich)
- Streaming-Chunks an die TUI weiterleiten

### Tool-Use-Schleife

```
Agent.Run(userMessage):
  1. userMessage zur History hinzufügen
  2. Provider.Complete(history, tools) aufrufen → Stream
  3. Stream lesen:
     a. Text-Chunks → an TUI streamen (bubbletea Cmd)
     b. ToolCall empfangen:
        i.  Tool ausführen
        ii. ToolResult zur History hinzufügen
        iii. Weiter bei Schritt 2 (neue LLM-Runde)
  4. StopReason == "end_turn" → fertig
```

### History-Management

Die gesamte Konversations-History wird in-memory gehalten (Slice of `Message`). Bei sehr langen Gesprächen: älteste Nachrichten werden nach einem konfigurierbaren Limit (`max_history_messages`) verworfen, Tool-Results immer behalten.

### System-Prompt

```
Du bist dispatch, ein Assistent für Developer, der bei der
Social Media-Planung hilft. Du hast Zugriff auf:
- GitHub: Commits, PRs und Issues in den Repos des Users
- Postiz: Erstellen und Planen von Social Media-Posts
- Search: Web-Recherche via Perplexity Sonar mit Quellenangaben

Antworte auf Deutsch, außer der User schreibt in einer anderen Sprache.
Bevor du einen Post planst, zeige immer eine Vorschau und bitte um Bestätigung.
```

---

## 7. Tools

### 7.1 Postiz Tool

**Tool-Definition für den LLM:**

```
Name: create_post
Beschreibung: Plant einen Social Media-Post via Postiz.
              Immer erst Vorschau zeigen und Bestätigung abwarten.
Parameter:
  platform      (string)  "linkedin" | "twitter" | "mastodon" | ...
  content       (string)  Post-Text
  scheduled_at  (string)  ISO 8601 Zeitpunkt, z.B. "2025-05-08T09:00:00Z"
  media_urls    ([]string) Optional: URLs zu Bildern

Name: list_posts
Beschreibung: Listet geplante Posts auf.
Parameter:
  status  (string) "scheduled" | "published" | "draft"
  limit   (int)    Anzahl, max 20

Name: list_channels
Beschreibung: Listet verbundene Social Media-Accounts auf.
Parameter: keine
```

**API-Endpunkte (Postiz REST):**

| Action         | Method | Endpoint                      |
|----------------|--------|-------------------------------|
| Post erstellen | POST   | `/api/posts`                  |
| Posts listen   | GET    | `/api/posts?status={status}`  |
| Channels listen| GET    | `/api/integrations`           |

**Bestätigungs-Flow:**

Bevor `create_post` ausgeführt wird, sendet der Agent eine `ActionRequired`-Message an die TUI. Die TUI zeigt die Optionen `j / e / p / z / n` und wartet auf Tastendruck. Erst danach löst die TUI die Bestätigung aus (bubbletea `Cmd`).

### 7.2 GitHub Tool

**Tool-Definition für den LLM:**

```
Name: get_recent_commits
Beschreibung: Holt aktuelle Commits aus einem oder allen Repos.
Parameter:
  repos   ([]string) Repo-Namen oder ["all"] für alle
  since   (string)   "24h" | "7d" | "30d"
  limit   (int)      Max Commits pro Repo, default 5

Name: get_activity
Beschreibung: Holt Commits, PRs und Issues kombiniert.
Parameter:
  repos    ([]string) Repo-Namen oder ["all"]
  since    (string)   "24h" | "7d" | "30d"
  include  ([]string) ["commits", "prs", "issues"]

Name: get_file
Beschreibung: Liest den Inhalt einer Datei aus einem Repo (z.B. README, CHANGELOG).
Parameter:
  repo  (string) Repo-Name
  path  (string) Dateipfad, z.B. "README.md"
```

**GitHub REST API Endpunkte:**

| Action              | Endpoint                                              |
|---------------------|-------------------------------------------------------|
| Repos auflisten     | `GET /user/repos`                                     |
| Commits holen       | `GET /repos/{owner}/{repo}/commits?since={iso}`       |
| PRs holen           | `GET /repos/{owner}/{repo}/pulls?state=open`          |
| Issues holen        | `GET /repos/{owner}/{repo}/issues?state=open`         |
| Datei lesen         | `GET /repos/{owner}/{repo}/contents/{path}`           |

**Token-Scopes:** `repo` (read-only reicht), `metadata`. Kein Write-Zugriff nötig.

### 7.3 Search Tool — Perplexity Sonar

**Tool-Definition für den LLM:**

```
Name: search
Beschreibung: Recherchiert ein Thema im Web und gibt eine synthetisierte
              Antwort mit nummerierten Quellenangaben zurück.
              Nutze dies für: aktuelle News, Trends, Technologie-Vergleiche,
              Hintergrundinformationen für Posts.
Parameter:
  query    (string)  Suchanfrage in natürlicher Sprache
  recency  (string)  "hour" | "day" | "week" | "month" — default: aus config
  model    (string)  "sonar" | "sonar-pro" | "sonar-reasoning" — default: aus config
```

**Perplexity API:**

```
POST https://api.perplexity.ai/chat/completions
Content-Type: application/json
Authorization: Bearer {api_key}

{
  "model": "sonar-pro",
  "messages": [{"role": "user", "content": "{query}"}],
  "search_recency_filter": "week",
  "return_citations": true
}
```

Die Response ist OpenAI-kompatibel. Zusätzlich enthält sie ein `citations`-Array mit den Quell-URLs.

**SearchResult-Typ:**

```go
type SearchResult struct {
    Answer    string     // synthetisierte Antwort
    Citations []Citation // [{Index, URL, Title}]
    Model     string     // genutztes Sonar-Modell
    Query     string     // ursprüngliche Anfrage
}

type Citation struct {
    Index int
    URL   string
    Title string
}
```

**Sonar-Modelle im Überblick:**

| Modell                | Geschwindigkeit | Qualität | Empfehlung              |
|-----------------------|-----------------|----------|-------------------------|
| `sonar`               | schnell         | gut      | Einfache Fakten-Checks  |
| `sonar-pro`           | mittel          | sehr gut | **Standard (empfohlen)**|
| `sonar-reasoning`     | langsam         | hoch     | Komplexe Analysen       |
| `sonar-reasoning-pro` | sehr langsam    | höchste  | Tiefe Recherchen        |

**Pluggable Backend Interface:**

```go
type SearchBackend interface {
    Search(ctx context.Context, req SearchRequest) (SearchResult, error)
    Name() string
}

// Implementierungen:
// - PerplexityBackend  (primär)
// - BraveBackend       (Fallback, liefert raw results)
// - SerpAPIBackend     (Fallback)
```

Ist kein Perplexity-Key konfiguriert, greift der Agent automatisch auf den konfigurierten Fallback zurück — oder deaktiviert das Search-Tool und informiert den User.

---

## 8. TUI (bubbletea)

### Architektur (Elm-Modell)

```go
type Model struct {
    // State
    history    []agent.Message
    streaming  bool
    inputValue string

    // bubbletea components
    viewport   viewport.Model   // scrollbarer Chat-Verlauf
    textarea   textarea.Model   // Eingabefeld
    spinner    spinner.Model    // Lade-Indikator

    // Channels
    streamCh   chan agent.Chunk

    // Config
    cfg        config.Config
    provider   provider.Provider
    agent      *agent.Agent
    width, height int
}
```

### Messages (bubbletea)

```go
type UserSentMsg     struct{ Text string }
type AgentChunkMsg   struct{ Chunk agent.Chunk }
type AgentDoneMsg    struct{}
type ToolCallMsg     struct{ Name, Args string }
type ToolResultMsg   struct{ Name, Result string }
type ActionRequiredMsg struct{ Prompt string; Options []string }
type ActionChosenMsg  struct{ Choice string }
type WindowSizeMsg   struct{ Width, Height int }
```

### Keyboard-Shortcuts

| Shortcut      | Aktion                                         |
|---------------|------------------------------------------------|
| `Enter`       | Nachricht senden                               |
| `Ctrl+C`      | Laufenden Agent-Call abbrechen                 |
| `Ctrl+P`      | Geplante Posts anzeigen                        |
| `Ctrl+R`      | GitHub-Repo-Aktivität anzeigen                 |
| `Ctrl+S`      | Direkt eine Suche starten                      |
| `Ctrl+L`      | Chat-Verlauf leeren                            |
| `PgUp/PgDn`   | Im Viewport scrollen                           |
| `↑`           | Letzte eigene Eingabe wiederholen              |
| `?`           | Hilfe anzeigen                                 |
| `Ctrl+Q`      | Beenden                                        |

### Screens / Views

Die TUI kennt keine separaten "Screens" im Router-Sinne — alles läuft im selben Chat-Viewport. Unterschiedliche Inhalte werden durch spezielle Message-Typen gerendert:

- **Normaler Chat:** Agent-Text mit grüner Markierung
- **Tool-Call:** eingerückter Block in Amber mit `▶ TOOL — name`
- **Post-Vorschau:** umrahmter Block mit Plattform / Zeitpunkt / Zeichenzahl
- **Aktions-Prompt:** lila Block mit Tastatur-Optionen (`j / e / p / z / n`)
- **Quellenangaben:** eingerückte Liste nach dem Agent-Text

### Status-Bar (Header)

```
dispatch   ● claude/sonnet-4-6   ● postiz   ● github   ● perplexity
```

Punkte färben sich grün (verbunden), gelb (optional/nicht konfiguriert) oder rot (Fehler).

---

## 9. Implementierungsphasen

### Phase 1 — Grundgerüst

**Ziel:** Das Projekt baut, startet und zeigt eine leere TUI.

- [ ] `go.mod` anlegen, Abhängigkeiten hinzufügen
- [ ] Cobra-CLI mit `run`- und `--setup`-Commands
- [ ] Viper-Config laden + validieren
- [ ] Minimale bubbletea-App: Eingabefeld, Viewport, Status-Bar
- [ ] lipgloss Styles-Datei anlegen (Farben zentral definieren)
- [ ] `--setup`-Wizard (Terminal-Dialog, Speicherung in config.toml)

### Phase 2 — Provider-Interface & erste LLM-Verbindung

**Ziel:** Einfaches Frage-Antwort ohne Tools, gestreamt.

- [ ] `Provider`-Interface definieren
- [ ] Anthropic-Adapter implementieren + Streaming
- [ ] Agent Core: einfache Schleife ohne Tool-Use
- [ ] Streaming-Chunks an bubbletea-Viewport weiterleiten
- [ ] OpenAI-Adapter (+ OpenRouter + xAI als Varianten)
- [ ] Gemini-Adapter

### Phase 3 — GitHub Tool

**Ziel:** Agent kann Repo-Daten abrufen.

- [ ] Tool-Definition für den LLM (internes Format)
- [ ] `converter.go`: internes Format → Provider-spezifisches Tool-Format
- [ ] Tool-Use-Schleife im Agent Core
- [ ] GitHub REST Client: `get_recent_commits`, `get_activity`, `get_file`
- [ ] TUI: Tool-Call-Rendering (`▶ TOOL — github`)

### Phase 4 — Search Tool (Perplexity)

**Ziel:** Agent kann recherchieren und Quellen anzeigen.

- [ ] `SearchBackend`-Interface
- [ ] Perplexity-Backend: API-Call + Citation-Parsing
- [ ] Brave-Backend als Fallback
- [ ] TUI: Quellenangaben-Rendering
- [ ] Graceful degradation wenn kein Search-Key konfiguriert

### Phase 5 — Postiz Tool

**Ziel:** Agent kann Posts vorschlagen, zeigen und planen.

- [ ] Postiz REST Client: `create_post`, `list_posts`, `list_channels`
- [ ] Post-Vorschau-Rendering in der TUI
- [ ] Bestätigungs-Flow: `ActionRequiredMsg` → Tastendruck → `ActionChosenMsg`
- [ ] Postiz-Bestätigung mit Post-ID anzeigen

### Phase 6 — Schliff & Open-Source-Vorbereitung

**Ziel:** Produktionsreif, für externe Nutzer nutzbar.

- [ ] Fehlerbehandlung durchgängig verbessern (keine Panics)
- [ ] Verbindungstest beim Start (`--check`)
- [ ] `--version` mit Build-Infos (ldflags)
- [ ] `README.md` mit Quickstart, Screenshots, Config-Doku
- [ ] `config.toml.example` finalisieren
- [ ] `.gitignore` sicherstellen (keine Keys commitbar)
- [ ] GitHub Actions: Build + Test für linux/darwin/windows
- [ ] Release-Binaries via `goreleaser`

---

## 10. Open-Source-Checkliste

### Sicherheit

- [ ] `config.toml` ist in `.gitignore` — **niemals** commitbar
- [ ] Kein Hardcoding von Keys, URLs oder persönlichen Daten
- [ ] Keys werden in Logs/TUI niemals im Klartext angezeigt (masking)
- [ ] GitHub-Token: minimale Scopes dokumentiert (`repo` read-only)
- [ ] `config.toml.example` enthält ausschließlich Platzhalter

### Dokumentation

- [ ] `README.md`: Was ist dispatch? Quickstart. Config. Screenshots.
- [ ] `docs/wireframes.html`: TUI-Design-Referenz
- [ ] `CONTRIBUTING.md`: Wie füge ich einen neuen Provider / ein neues Tool hinzu?
- [ ] Inline-Kommentare für alle Interfaces und nicht-offensichtliche Logik
- [ ] `config.toml.example`: jede Zeile kommentiert

### Code-Qualität

- [ ] `golangci-lint` im CI (`.golangci.yml` im Repo)
- [ ] Alle Interfaces aus internen Paketen sind testbar (kein direktes HTTP in Business-Logik)
- [ ] Mindestens ein Test pro Tool (mit Mock-HTTP-Server)
- [ ] `go vet` und `staticcheck` im CI

### Release

- [ ] Semantische Versionierung (`v0.1.0`, `v0.2.0`, ...)
- [ ] `CHANGELOG.md` pflegen
- [ ] `goreleaser` für Binaries: linux/amd64, linux/arm64, darwin/arm64, windows/amd64
- [ ] Homebrew-Tap langfristig (nice-to-have)

---

## Referenzen

| Dienst        | Dokumentation                                              |
|---------------|------------------------------------------------------------|
| bubbletea     | https://github.com/charmbracelet/bubbletea                 |
| lipgloss      | https://github.com/charmbracelet/lipgloss                  |
| Anthropic API | https://docs.anthropic.com/en/api/getting-started          |
| OpenAI API    | https://platform.openai.com/docs/api-reference             |
| Gemini API    | https://ai.google.dev/api/generate-content                 |
| OpenRouter    | https://openrouter.ai/docs                                 |
| xAI API       | https://docs.x.ai/api                                      |
| Perplexity    | https://docs.perplexity.ai/api-reference/chat-completions  |
| Postiz API    | https://docs.postiz.app/api-reference                      |
| GitHub API    | https://docs.github.com/en/rest                            |
| Viper         | https://github.com/spf13/viper                             |
| Cobra         | https://github.com/spf13/cobra                             |
| goreleaser    | https://goreleaser.com/intro/                              |
