package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LogLimit    int    `json:"log_limit"`
	RebaseLimit int    `json:"rebase_limit"`
	SplitPane   bool   `json:"split_pane"`
	Editor      string `json:"editor"`
}

func Default() Config {
	return Config{
		LogLimit:    100,
		RebaseLimit: 15,
		SplitPane:   true,
		Editor:      "",
	}
}

// Load reads the config file, returning defaults for any missing values.
func Load() Config {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

// Save writes cfg to the config file, creating the directory if needed.
func Save(cfg Config) error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Path returns the config file path, respecting CGIT_CONFIG env var.
func Path() string {
	if p := os.Getenv("CGIT_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cgit", "config.json")
}
