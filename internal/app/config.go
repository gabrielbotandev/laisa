package app

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultSystemPrompt = `You are a helpful local AI assistant running on the user's laptop.
Be concise, practical, and direct.
For code questions, give useful explanations and runnable examples when appropriate.
You do not have live internet access.
`

// Config holds user settings persisted in YAML.
type Config struct {
	DefaultModel  string `yaml:"default_model"`
	DefaultDevice string `yaml:"default_device"`
	MaxTokens     int    `yaml:"max_tokens"`
	SystemPrompt  string `yaml:"system_prompt"`
}

// DefaultConfig returns factory defaults.
func DefaultConfig() Config {
	return Config{
		DefaultModel:  "",
		DefaultDevice: "CPU",
		MaxTokens:     1000,
		SystemPrompt:  defaultSystemPrompt,
	}
}

// LoadOrCreate reads config or writes defaults on first run.
func LoadOrCreate() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.DefaultDevice == "" {
		cfg.DefaultDevice = "CPU"
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 1000
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = defaultSystemPrompt
	}

	return cfg, nil
}

// Save writes the config to disk.
func (c Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := EnsureDirs(); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// RunOptions are effective settings for a single run.
type RunOptions struct {
	Model       string
	Device      string
	MaxTokens   int
	SystemPrompt string
}

// EffectiveRunOptions merges config with CLI overrides (flags win).
func EffectiveRunOptions(cfg Config, model, device string, maxTokens int, maxTokensSet bool) RunOptions {
	opts := RunOptions{
		Model:        cfg.DefaultModel,
		Device:       cfg.DefaultDevice,
		MaxTokens:    cfg.MaxTokens,
		SystemPrompt: cfg.SystemPrompt,
	}
	if model != "" {
		opts.Model = model
	}
	if device != "" {
		opts.Device = strings.ToUpper(device)
	}
	if maxTokensSet && maxTokens > 0 {
		opts.MaxTokens = maxTokens
	}
	return opts
}

// FormatHuman returns a readable config summary.
func (c Config) FormatHuman() (string, error) {
	path, err := ConfigPath()
	if err != nil {
		return "", err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Config: %s\n\n%s", path, string(data)), nil
}
