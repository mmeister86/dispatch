package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	appName        = "dispatch"
	defaultCfgFile = "config.toml"
)

type Config struct {
	LLM    LLMConfig    `mapstructure:"llm"`
	Postiz PostizConfig `mapstructure:"postiz"`
	GitHub GitHubConfig `mapstructure:"github"`
	Search SearchConfig `mapstructure:"search"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
}

type PostizConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type GitHubConfig struct {
	Token string `mapstructure:"token"`
}

type SearchConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Recency  string `mapstructure:"recency"`
}

func Default() Config {
	return Config{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-6",
		},
		Postiz: PostizConfig{
			BaseURL: "https://app.postiz.com",
		},
		Search: SearchConfig{
			Provider: "perplexity",
			Model:    "sonar-pro",
			Recency:  "week",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	v := newViper(path)
	applyDefaults(v, cfg)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return Config{}, err
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Write(path string, cfg Config) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(renderTOML(cfg)), 0o600)
}

func DefaultPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName, defaultCfgFile), nil
}

func DisplayPath(path string) string {
	if path != "" {
		return path
	}
	defaultPath, err := DefaultPath()
	if err != nil {
		return filepath.Join("~", ".config", appName, defaultCfgFile)
	}
	return defaultPath
}

func (c Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.LLM.APIKey) == "" {
		missing = append(missing, "LLM_API_KEY oder llm.api_key")
	}
	if strings.TrimSpace(c.LLM.Provider) == "" {
		missing = append(missing, "llm.provider")
	}
	if strings.TrimSpace(c.LLM.Model) == "" {
		missing = append(missing, "llm.model")
	}
	if strings.TrimSpace(c.Postiz.APIKey) == "" {
		missing = append(missing, "POSTIZ_API_KEY oder postiz.api_key")
	}
	if strings.TrimSpace(c.GitHub.Token) == "" {
		missing = append(missing, "GITHUB_TOKEN oder github.token")
	}

	if len(missing) == 0 {
		return nil
	}

	lines := []string{"Konfiguration unvollstaendig:"}
	for _, key := range missing {
		lines = append(lines, fmt.Sprintf("✗ %s fehlt.", key))
	}
	lines = append(lines, "Setze die Env-Variablen oder starte `dispatch --setup`.")
	return errors.New(strings.Join(lines, "\n"))
}

func newViper(path string) *viper.Viper {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindEnv(v, "llm.provider")
	bindEnv(v, "llm.model")
	bindEnv(v, "llm.api_key", "LLM_API_KEY")
	bindEnv(v, "postiz.api_key", "POSTIZ_API_KEY")
	bindEnv(v, "postiz.base_url", "POSTIZ_BASE_URL")
	bindEnv(v, "github.token", "GITHUB_TOKEN")
	bindEnv(v, "search.provider", "SEARCH_PROVIDER")
	bindEnv(v, "search.api_key", "SEARCH_API_KEY")
	bindEnv(v, "search.model", "SEARCH_MODEL")
	bindEnv(v, "search.recency", "SEARCH_RECENCY")

	if path != "" {
		v.SetConfigFile(path)
		return v
	}

	v.SetConfigName(strings.TrimSuffix(defaultCfgFile, filepath.Ext(defaultCfgFile)))
	if dir, err := defaultConfigDir(); err == nil {
		v.AddConfigPath(filepath.Join(dir, appName))
	}
	v.AddConfigPath(".")
	return v
}

func defaultConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func bindEnv(v *viper.Viper, key string, aliases ...string) {
	if len(aliases) == 0 {
		_ = v.BindEnv(key)
		return
	}
	args := append([]string{key}, aliases...)
	_ = v.BindEnv(args...)
}

func applyDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("llm.provider", cfg.LLM.Provider)
	v.SetDefault("llm.model", cfg.LLM.Model)
	v.SetDefault("postiz.base_url", cfg.Postiz.BaseURL)
	v.SetDefault("search.provider", cfg.Search.Provider)
	v.SetDefault("search.model", cfg.Search.Model)
	v.SetDefault("search.recency", cfg.Search.Recency)
}

func renderTOML(cfg Config) string {
	return fmt.Sprintf(`[llm]
provider = %q
model = %q
api_key = %q

[postiz]
api_key = %q
base_url = %q

[github]
token = %q

[search]
provider = %q
api_key = %q
model = %q
recency = %q
`, cfg.LLM.Provider, cfg.LLM.Model, cfg.LLM.APIKey,
		cfg.Postiz.APIKey, cfg.Postiz.BaseURL,
		cfg.GitHub.Token,
		cfg.Search.Provider, cfg.Search.APIKey, cfg.Search.Model, cfg.Search.Recency)
}
