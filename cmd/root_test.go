package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthias/dispatch/config"
	"github.com/spf13/cobra"
)

func TestRunSetupUsesDefaultProviderAndModelSelections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	input := strings.Join([]string{
		"",                    // default provider
		"",                    // default model
		"llm-key",             // LLM API key
		"https://postiz.test", // Postiz URL
		"postiz-key",          // Postiz key
		"github-token",        // GitHub token
		"search-key",          // Search key
		"",                    // confirm save
		"",
	}, "\n")

	out, err := runSetupWithInput(path, input)
	if err != nil {
		t.Fatalf("runSetup returned error: %v\nOutput:\n%s", err, out)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Fatalf("provider = %q", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "claude-sonnet-4-6" {
		t.Fatalf("model = %q", cfg.LLM.Model)
	}
	for _, want := range []string{
		"LLM Provider",
		"1) Anthropic",
		"LLM Modell fuer anthropic",
		"Zusammenfassung",
		"LLM: anthropic / claude-sonnet-4-6",
		"GitHub: konfiguriert",
		"Naechster Schritt: dispatch check",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("setup output missing %q:\n%s", want, out)
		}
	}
}

func TestRunSetupRetriesInvalidSelectionAndAcceptsCustomModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	input := strings.Join([]string{
		"9",               // invalid provider
		"2",               // OpenAI
		"9",               // invalid model
		"6",               // custom model
		"my-openai-model", // custom model value
		"llm-key",
		"https://postiz.test",
		"postiz-key",
		"github-token",
		"search-key",
		"y",
		"",
	}, "\n")

	out, err := runSetupWithInput(path, input)
	if err != nil {
		t.Fatalf("runSetup returned error: %v\nOutput:\n%s", err, out)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LLM.Provider != "openai" {
		t.Fatalf("provider = %q", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "my-openai-model" {
		t.Fatalf("model = %q", cfg.LLM.Model)
	}
	if strings.Count(out, "Ungueltige Auswahl") < 2 {
		t.Fatalf("expected provider and model retry messages:\n%s", out)
	}
}

func TestRunSetupCanAbortBeforeWritingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	input := strings.Join([]string{
		"3", // Gemini
		"2", // Gemini 2.5 Pro
		"llm-key",
		"https://postiz.test",
		"postiz-key",
		"github-token",
		"search-key",
		"n",
		"",
	}, "\n")

	out, err := runSetupWithInput(path, input)
	if err == nil {
		t.Fatalf("expected abort error\nOutput:\n%s", out)
	}
	if !strings.Contains(err.Error(), "abgebrochen") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("config file stat error = %v, want not exist", statErr)
	}
}

func TestRunSetupKeepsExistingValuesWhenPromptsAreSkipped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	existing := config.Default()
	existing.LLM.Provider = "openai"
	existing.LLM.Model = "gpt-5.4-mini"
	existing.LLM.APIKey = "existing-llm"
	existing.Postiz.BaseURL = "https://postiz.existing"
	existing.Postiz.APIKey = "existing-postiz"
	existing.GitHub.Token = "existing-github"
	existing.Search.APIKey = "existing-search"
	if err := config.Write(path, existing); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	input := strings.Join([]string{
		"", // keep provider
		"", // keep model
		"", // keep LLM API key
		"", // keep Postiz URL
		"", // keep Postiz key
		"", // keep GitHub token
		"", // keep Search key
		"", // confirm save
		"",
	}, "\n")

	out, err := runSetupWithInput(path, input)
	if err != nil {
		t.Fatalf("runSetup returned error: %v\nOutput:\n%s", err, out)
	}

	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.LLM.Provider != existing.LLM.Provider ||
		got.LLM.Model != existing.LLM.Model ||
		got.LLM.APIKey != existing.LLM.APIKey ||
		got.Postiz.BaseURL != existing.Postiz.BaseURL ||
		got.Postiz.APIKey != existing.Postiz.APIKey ||
		got.GitHub.Token != existing.GitHub.Token ||
		got.Search.APIKey != existing.Search.APIKey {
		t.Fatalf("existing values not preserved:\ngot:  %#v\nwant: %#v", got, existing)
	}
}

func TestAskSecretKeepsNonTTYLineFallback(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  secret-from-pipe  \n"))
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader("  secret-from-pipe  \n"))
	cmd.SetOut(&out)

	got, err := askSecret(reader, cmd, "LLM API Key")
	if err != nil {
		t.Fatalf("askSecret returned error: %v", err)
	}
	if got != "secret-from-pipe" {
		t.Fatalf("secret = %q", got)
	}
	if !strings.Contains(out.String(), "LLM API Key: ") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestReadMaskedSecretEchoesStarsForPastedInput(t *testing.T) {
	var out bytes.Buffer
	got, err := readMaskedSecret(bufio.NewReader(strings.NewReader("pasted-value\n")), &out)
	if err != nil {
		t.Fatalf("readMaskedSecret returned error: %v", err)
	}
	if got != "pasted-value" {
		t.Fatalf("secret = %q", got)
	}
	if strings.Count(out.String(), "*") != len("pasted-value") {
		t.Fatalf("masked output = %q", out.String())
	}
	if !strings.HasSuffix(out.String(), "\r\n") {
		t.Fatalf("masked output should end with CRLF in raw mode: %q", out.String())
	}
}

func TestReadMaskedSecretHandlesBackspace(t *testing.T) {
	var out bytes.Buffer
	got, err := readMaskedSecret(bufio.NewReader(strings.NewReader("ab\x7fc\n")), &out)
	if err != nil {
		t.Fatalf("readMaskedSecret returned error: %v", err)
	}
	if got != "ac" {
		t.Fatalf("secret = %q", got)
	}
	if !strings.Contains(out.String(), "\b \b") {
		t.Fatalf("masked output missing erase sequence: %q", out.String())
	}
}

func TestReadMaskedSecretAllowsEmptyEnter(t *testing.T) {
	var out bytes.Buffer
	got, err := readMaskedSecret(bufio.NewReader(strings.NewReader("\n")), &out)
	if err != nil {
		t.Fatalf("readMaskedSecret returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("secret = %q", got)
	}
}

func TestReadMaskedSecretInterruptsOnCtrlC(t *testing.T) {
	var out bytes.Buffer
	_, err := readMaskedSecret(bufio.NewReader(strings.NewReader("abc\x03")), &out)
	if err == nil {
		t.Fatal("expected interrupt error")
	}
}

func runSetupWithInput(path, input string) (string, error) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(input))
	cmd.SetOut(&out)
	err := runSetup(cmd, path)
	return out.String(), err
}
