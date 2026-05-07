package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matthias/dispatch/config"
	"github.com/matthias/dispatch/internal/agent"
	"github.com/matthias/dispatch/internal/provider"
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

	p := tea.NewProgram(tui.New(cfg, chatAgent, warnings), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

const systemPrompt = `Du bist dispatch, ein Assistent fuer Developer, der bei der Social-Media-Planung hilft.
Du antwortest auf Deutsch, ausser der User schreibt in einer anderen Sprache.
Du hast Tools fuer GitHub-Kontext, Web-Recherche und Postiz-Planung.
Nutze GitHub fuer Repo-Aktivitaet, Search fuer aktuelle Themen und Postiz fuer Posts.
Bevor du einen Post planst, zeige immer eine Vorschau und bitte um Bestaetigung.
Rufe create_post nur auf, wenn der User nach der Vorschau eindeutig zugestimmt hat, und setze dann confirmed=true.`

func runSetup(cmd *cobra.Command, path string) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	cfg := config.Default()

	fmt.Fprintln(cmd.OutOrStdout(), "dispatch Einrichtung")
	fmt.Fprintf(cmd.OutOrStdout(), "Config-Ziel: %s\n\n", config.DisplayPath(path))

	provider, err := ask(reader, cmd.OutOrStdout(), "LLM Provider [anthropic/openai/gemini/openrouter/xai]", cfg.LLM.Provider)
	if err != nil {
		return err
	}
	cfg.LLM.Provider = provider

	model, err := ask(reader, cmd.OutOrStdout(), "LLM Modell", cfg.LLM.Model)
	if err != nil {
		return err
	}
	cfg.LLM.Model = model

	apiKey, err := askSecret(reader, cmd, "LLM API Key")
	if err != nil {
		return err
	}
	cfg.LLM.APIKey = apiKey

	postizURL, err := ask(reader, cmd.OutOrStdout(), "Postiz Base URL", cfg.Postiz.BaseURL)
	if err != nil {
		return err
	}
	cfg.Postiz.BaseURL = postizURL

	postizKey, err := askSecret(reader, cmd, "Postiz API Key (optional)")
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

	if err := config.Write(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Konfiguration gespeichert: %s\n", config.DisplayPath(path))
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "\nHinweis:\n%s\n", err)
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

	bytes, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(cmd.OutOrStdout())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
