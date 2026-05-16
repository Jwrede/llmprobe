package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Defaults  Defaults   `yaml:"defaults"`
	Providers []Provider `yaml:"providers"`
}

type Defaults struct {
	Prompt      string   `yaml:"prompt"`
	MaxTokens   int      `yaml:"max_tokens"`
	Timeout     Duration `yaml:"timeout"`
	Concurrency int      `yaml:"concurrency"`
}

type Provider struct {
	Name       string  `yaml:"name"`
	Label      string  `yaml:"label"`
	APIKey     string  `yaml:"api_key"`
	BaseURL    string  `yaml:"base_url"`
	Models     []Model `yaml:"models"`
	APIVersion string  `yaml:"api_version"`
	Region     string  `yaml:"region"`
	AccessKey  string  `yaml:"access_key"`
	SecretKey  string  `yaml:"secret_key"`
}

func (p Provider) DisplayName() string {
	if p.Label != "" {
		return p.Label
	}
	return p.Name
}

type Model struct {
	Name       string     `yaml:"name"`
	Prompt     string     `yaml:"prompt"`
	MaxTokens  int        `yaml:"max_tokens"`
	Thresholds Thresholds `yaml:"thresholds"`
}

type Thresholds struct {
	MaxTTFT       Duration `yaml:"max_ttft"`
	MaxLatency    Duration `yaml:"max_latency"`
	MinTokensPerS float64  `yaml:"min_tokens_per_sec"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	expandEnvInKeys(&cfg)
	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func expandEnvInKeys(cfg *Config) {
	for i := range cfg.Providers {
		cfg.Providers[i].APIKey = os.ExpandEnv(cfg.Providers[i].APIKey)
		cfg.Providers[i].AccessKey = os.ExpandEnv(cfg.Providers[i].AccessKey)
		cfg.Providers[i].SecretKey = os.ExpandEnv(cfg.Providers[i].SecretKey)
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Defaults.Prompt == "" {
		cfg.Defaults.Prompt = "Hello"
	}
	if cfg.Defaults.MaxTokens == 0 {
		cfg.Defaults.MaxTokens = 20
	}
	if cfg.Defaults.Timeout.Duration == 0 {
		cfg.Defaults.Timeout.Duration = 30 * time.Second
	}
	if cfg.Defaults.Concurrency == 0 {
		cfg.Defaults.Concurrency = 5
	}

	for i := range cfg.Providers {
		for j := range cfg.Providers[i].Models {
			m := &cfg.Providers[i].Models[j]
			if m.Prompt == "" {
				m.Prompt = cfg.Defaults.Prompt
			}
			if m.MaxTokens == 0 {
				m.MaxTokens = cfg.Defaults.MaxTokens
			}
		}
	}
}

func validate(cfg *Config) error {
	if len(cfg.Providers) == 0 {
		return fmt.Errorf("config: no providers configured")
	}
	known := map[string]bool{"openai": true, "anthropic": true, "google": true, "azure": true, "bedrock": true}
	seen := make(map[string]bool)
	for _, p := range cfg.Providers {
		if !known[p.Name] {
			return fmt.Errorf("config: unknown provider %q (supported: openai, anthropic, google, azure, bedrock)", p.Name)
		}
		displayName := p.DisplayName()
		if seen[displayName] {
			return fmt.Errorf("config: duplicate provider name %q (use the label field to distinguish)", displayName)
		}
		seen[displayName] = true
		if len(p.Models) == 0 {
			return fmt.Errorf("config: provider %q has no models", displayName)
		}
		switch p.Name {
		case "bedrock":
			if p.AccessKey == "" || p.SecretKey == "" {
				return fmt.Errorf("config: provider %q needs access_key and secret_key", displayName)
			}
			if p.Region == "" {
				return fmt.Errorf("config: provider %q needs region", displayName)
			}
		case "azure":
			if p.APIKey == "" {
				return fmt.Errorf("config: provider %q has no api_key (set the env var)", displayName)
			}
			if p.BaseURL == "" {
				return fmt.Errorf("config: provider %q needs base_url (e.g. https://your-resource.openai.azure.com)", displayName)
			}
		default:
			if p.APIKey == "" {
				return fmt.Errorf("config: provider %q has no api_key (set the env var)", displayName)
			}
		}
	}
	return nil
}
