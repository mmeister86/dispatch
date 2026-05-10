package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/agent"
	"github.com/matthias/dispatch/internal/provider"
	"github.com/matthias/dispatch/internal/session"
	"github.com/matthias/dispatch/internal/tools"
	"github.com/matthias/dispatch/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	cfgFile string
	setup   bool

	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "dispatch",
		Short:         "KI-gestuetzte Social Media-Planung im Terminal",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if setup {
				return runSetup(cmd, cfgFile)
			}
			return runTUI(cfgFile)
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "Pfad zu config.toml")
	root.Flags().BoolVar(&setup, "setup", false, "gefuehrte Konfiguration starten")

	root.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "TUI starten",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cfgFile)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Konfiguration pruefen",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Konfiguration ist vollstaendig.")
			return nil
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Version anzeigen",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "dispatch %s (%s, %s)\n", version, commit, date)
		},
	})

	return root
}

func runTUI(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	var warnings []string
	var chatAgent *agent.Agent
	if cfg.LLM.APIKey == "" {
		warnings = append(warnings, "LLM API Key fehlt. Starte `dispatch --setup` oder setze LLM_API_KEY.")
	} else {
		p, err := provider.NewFromConfig(cfg)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else if p != nil {
			chatAgent = agent.NewWithTools(p, tools.NewRegistry(cfg), systemPrompt, 80)
		}
	}

	p := tea.NewProgram(tui.NewWithSessionStore(cfg, chatAgent, warnings, session.NewStore("")), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

const systemPrompt = `Du bist dispatch, ein Assistent fuer Developer, der bei der Social-Media-Planung hilft.
Du antwortest auf Deutsch, ausser der User schreibt in einer anderen Sprache.
Du hast Tools fuer GitHub-Kontext, Web-Recherche und Postiz-Planung.
Nutze GitHub fuer Repo-Aktivitaet, Search fuer aktuelle Themen und Postiz fuer Posts.
Bevor du einen Post planst, zeige immer eine Vorschau und bitte um Bestaetigung.
Wenn du einen X/Twitter-Thread planst, zeige jeden Tweet nummeriert in der Vorschau.
Rufe create_post fuer X/Twitter-Threads mit thread_parts als Array der einzelnen Tweets auf, nicht als mehrere einzelne Posts.
Rufe create_post nur auf, wenn der User nach der Vorschau eindeutig zugestimmt hat, und setze dann confirmed=true.
Rufe delete_post, set_post_status und connect_post_release nur nach eindeutiger User-Bestaetigung mit confirmed=true auf.`

const customModelValue = "__custom_model__"

var errSetupAborted = errors.New("Einrichtung abgebrochen; keine Konfiguration gespeichert")

type setupChoice struct {
	Label string
	Value string
}

var providerChoices = []setupChoice{
	{Label: "Anthropic", Value: "anthropic"},
	{Label: "OpenAI", Value: "openai"},
	{Label: "Gemini", Value: "gemini"},
	{Label: "OpenRouter", Value: "openrouter"},
	{Label: "xAI", Value: "xai"},
}

var modelChoicesByProvider = map[string][]setupChoice{
	"anthropic": {
		{Label: "Claude Sonnet 4.6", Value: "claude-sonnet-4-6"},
		{Label: "Claude Opus 4.7", Value: "claude-opus-4-7"},
		{Label: "Claude Haiku 4.5", Value: "claude-haiku-4-5-20251001"},
	},
	"openai": {
		{Label: "GPT-5.4 mini", Value: "gpt-5.4-mini"},
		{Label: "GPT-5.5", Value: "gpt-5.5"},
		{Label: "GPT-5.4", Value: "gpt-5.4"},
		{Label: "GPT-5.4 nano", Value: "gpt-5.4-nano"},
		{Label: "GPT-5", Value: "gpt-5"},
	},
	"gemini": {
		{Label: "Gemini 2.5 Flash", Value: "gemini-2.5-flash"},
		{Label: "Gemini 2.5 Pro", Value: "gemini-2.5-pro"},
		{Label: "Gemini 2.5 Flash-Lite", Value: "gemini-2.5-flash-lite"},
		{Label: "Gemini 3.1 Pro Preview", Value: "gemini-3.1-pro-preview"},
		{Label: "Gemini 3 Flash Preview", Value: "gemini-3-flash-preview"},
	},
	"openrouter": {
		{Label: "OpenRouter Auto", Value: "openrouter/auto"},
		{Label: "Claude Sonnet 4.6", Value: "anthropic/claude-sonnet-4.6"},
		{Label: "Gemini 3.1 Pro Preview", Value: "google/gemini-3.1-pro-preview"},
		{Label: "Gemini 2.5 Flash-Lite", Value: "google/gemini-2.5-flash-lite"},
		{Label: "DeepSeek R1", Value: "deepseek/deepseek-r1"},
	},
	"xai": {
		{Label: "Grok 4.3", Value: "grok-4.3"},
		{Label: "Grok 4.20 Reasoning", Value: "grok-4.20-0309-reasoning"},
		{Label: "Grok 4.20 Non-Reasoning", Value: "grok-4.20-0309-non-reasoning"},
		{Label: "Grok 4.20 Multi-Agent", Value: "grok-4.20-multi-agent-0309"},
	},
}

func runSetup(cmd *cobra.Command, path string) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	cfg := config.Default()
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, "dispatch Einrichtung")
	fmt.Fprintf(out, "Config-Ziel: %s\n\n", config.DisplayPath(path))

	provider, err := askChoice(reader, out, "LLM Provider", providerChoices, cfg.LLM.Provider)
	if err != nil {
		return err
	}
	cfg.LLM.Provider = provider

	modelChoices := append([]setupChoice(nil), modelChoicesByProvider[cfg.LLM.Provider]...)
	modelChoices = append(modelChoices, setupChoice{Label: "Custom model...", Value: customModelValue})
	model, err := askChoice(reader, out, fmt.Sprintf("LLM Modell fuer %s", cfg.LLM.Provider), modelChoices, cfg.LLM.Model)
	if err != nil {
		return err
	}
	if model == customModelValue {
		model, err = ask(reader, out, "Custom LLM Modell", modelChoices[0].Value)
		if err != nil {
			return err
		}
	}
	cfg.LLM.Model = model

	apiKey, err := askSecret(reader, cmd, "LLM API Key")
	if err != nil {
		return err
	}
	cfg.LLM.APIKey = apiKey

	postizURL, err := ask(reader, cmd.OutOrStdout(), "Postiz MCP Base URL", cfg.Postiz.BaseURL)
	if err != nil {
		return err
	}
	cfg.Postiz.BaseURL = postizURL

	postizKey, err := askSecret(reader, cmd, "Postiz MCP API Key (optional)")
	if err != nil {
		return err
	}
	cfg.Postiz.APIKey = postizKey

	githubToken, err := askSecret(reader, cmd, "GitHub Token (optional)")
	if err != nil {
		return err
	}
	cfg.GitHub.Token = githubToken

	searchKey, err := askSecret(reader, cmd, "Perplexity/Search API Key (optional)")
	if err != nil {
		return err
	}
	cfg.Search.APIKey = searchKey

	printSetupSummary(out, cfg, config.DisplayPath(path))
	save, err := askConfirm(reader, out, "Konfiguration speichern?", true)
	if err != nil {
		return err
	}
	if !save {
		return errSetupAborted
	}

	if err := config.Write(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n✓ Konfiguration gespeichert: %s\n", config.DisplayPath(path))
	fmt.Fprintln(out, "Naechster Schritt: dispatch check")
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(out, "\nHinweis:\n%s\n", err)
	}
	return nil
}

func ask(reader *bufio.Reader, out io.Writer, label, fallback string) (string, error) {
	fmt.Fprintf(out, "%s [%s]: ", label, fallback)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	return value, nil
}

func askChoice(reader *bufio.Reader, out io.Writer, label string, choices []setupChoice, fallback string) (string, error) {
	defaultIndex := choiceIndex(choices, fallback)
	if defaultIndex == -1 {
		defaultIndex = 0
	}

	for {
		fmt.Fprintf(out, "%s:\n", label)
		for i, choice := range choices {
			recommended := ""
			if i == defaultIndex {
				recommended = " (empfohlen)"
			}
			fmt.Fprintf(out, "  %d) %s - %s%s\n", i+1, choice.Label, choice.Value, recommended)
		}
		fmt.Fprintf(out, "Auswahl [%d]: ", defaultIndex+1)

		value, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return choices[defaultIndex].Value, nil
		}
		for i, choice := range choices {
			if value == fmt.Sprintf("%d", i+1) || strings.EqualFold(value, choice.Value) {
				return choice.Value, nil
			}
		}
		fmt.Fprintf(out, "Ungueltige Auswahl. Bitte 1-%d eingeben.\n\n", len(choices))
	}
}

func choiceIndex(choices []setupChoice, value string) int {
	for i, choice := range choices {
		if choice.Value == value {
			return i
		}
	}
	return -1
}

func askConfirm(reader *bufio.Reader, out io.Writer, label string, fallback bool) (bool, error) {
	suffix := "Y/n"
	if !fallback {
		suffix = "y/N"
	}
	for {
		fmt.Fprintf(out, "%s [%s]: ", label, suffix)
		value, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "":
			return fallback, nil
		case "y", "yes", "j", "ja":
			return true, nil
		case "n", "no", "nein":
			return false, nil
		default:
			fmt.Fprintln(out, "Bitte mit y oder n antworten.")
		}
	}
}

func printSetupSummary(out io.Writer, cfg config.Config, path string) {
	fmt.Fprintln(out, "\nZusammenfassung")
	fmt.Fprintf(out, "LLM: %s / %s\n", cfg.LLM.Provider, cfg.LLM.Model)
	fmt.Fprintf(out, "LLM API Key: %s\n", configuredStatus(cfg.LLM.APIKey))
	fmt.Fprintf(out, "Postiz: %s (%s)\n", configuredStatus(cfg.Postiz.APIKey), cfg.Postiz.BaseURL)
	fmt.Fprintf(out, "GitHub: %s\n", configuredStatus(cfg.GitHub.Token))
	fmt.Fprintf(out, "Search: %s\n", configuredStatus(cfg.Search.APIKey))
	fmt.Fprintf(out, "Config-Ziel: %s\n\n", path)
}

func configuredStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return "uebersprungen"
	}
	return "konfiguriert"
}

func askSecret(reader *bufio.Reader, cmd *cobra.Command, label string) (string, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label)
	file, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		value, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(value), nil
	}

	oldState, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = term.Restore(int(file.Fd()), oldState)
	}()

	value, err := readMaskedSecret(reader, cmd.OutOrStdout())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func readMaskedSecret(reader *bufio.Reader, out io.Writer) (string, error) {
	var value []byte
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return string(value), nil
			}
			return "", err
		}

		switch b {
		case '\r', '\n':
			fmt.Fprintln(out)
			return string(value), nil
		case 3:
			fmt.Fprintln(out)
			return "", errors.New("Eingabe abgebrochen")
		case '\b', 0x7f:
			if len(value) > 0 {
				value = value[:len(value)-1]
				fmt.Fprint(out, "\b \b")
			}
		default:
			value = append(value, b)
			fmt.Fprint(out, "*")
		}
	}
}
