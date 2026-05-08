package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsWhenConfigMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Fatalf("default provider = %q", cfg.LLM.Provider)
	}
	if cfg.Search.Model != "sonar-pro" {
		t.Fatalf("default search model = %q", cfg.Search.Model)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("LLM_API_KEY", "llm-env")
	t.Setenv("GITHUB_TOKEN", "gh-env")
	t.Setenv("POSTIZ_API_KEY", "pz-env")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LLM.APIKey != "llm-env" || cfg.GitHub.Token != "gh-env" || cfg.Postiz.APIKey != "pz-env" {
		t.Fatalf("env overrides not applied: %#v", cfg)
	}
}

func TestValidateReportsMissingKeys(t *testing.T) {
	err := Default().Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{"LLM_API_KEY", "POSTIZ_API_KEY", "GITHUB_TOKEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("validation error missing %q: %s", want, err)
		}
	}
}

func TestValidateAllowsPostizKeyInMCPURL(t *testing.T) {
	cfg := Default()
	cfg.LLM.APIKey = "llm"
	cfg.GitHub.Token = "github"
	cfg.Postiz.BaseURL = "https://postiz.example/api/mcp/postiz-key"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestWriteAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg := Default()
	cfg.LLM.APIKey = "llm"
	cfg.Postiz.APIKey = "postiz"
	cfg.GitHub.Token = "github"
	cfg.Search.APIKey = "search"

	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %v", info.Mode().Perm())
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.LLM.APIKey != "llm" || loaded.Search.APIKey != "search" {
		t.Fatalf("roundtrip mismatch: %#v", loaded)
	}
}

func TestDefaultPathUsesDotConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}

	want := filepath.Join(home, ".config", appName, defaultCfgFile)
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}
