package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultClipboardSearchInterval = 100 * time.Millisecond

type Config struct {
	Repo      RepoConfig               `yaml:"repo"`
	Docker    DockerConfig             `yaml:"docker"`
	Time      TimeConfig               `yaml:"time"`
	Clipboard ClipboardConfig          `yaml:"clipboard"`
	Projects  map[string]ProjectConfig `yaml:"proj"`
}

type RepoConfig struct {
	Roots []string `yaml:"roots"`
}

type DockerConfig struct {
	Registries map[string]string `yaml:"registries"`
}

type TimeConfig struct {
	Zones []string `yaml:"zones"`
}

type ClipboardConfig struct {
	BaseURL        string        `yaml:"base_url"`
	Token          string        `yaml:"token"`
	SearchInterval time.Duration `yaml:"search_interval"`
}

type ProjectConfig struct {
	Match string   `yaml:"match"`
	Logs  []string `yaml:"logs"`
}

func DefaultPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "leo-cli", "config.yaml"), nil
}

func Ensure(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte("repo:\n  roots:\n    - ~/work\ntime:\n  zones:\n    - +9\n    - +0\nclipboard:\n  base_url: http://127.0.0.1:8080\n  token: \"\"\n  search_interval: 100ms\n"), 0o600)
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{Clipboard: ClipboardConfig{SearchInterval: DefaultClipboardSearchInterval}}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Clipboard.SearchInterval <= 0 {
		cfg.Clipboard.SearchInterval = DefaultClipboardSearchInterval
	}
	return cfg, nil
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	expanded := os.ExpandEnv(path)
	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}

	if !filepath.IsAbs(expanded) {
		abs, err := filepath.Abs(expanded)
		if err != nil {
			return "", err
		}
		expanded = abs
	}

	return filepath.Clean(expanded), nil
}

func ExpandedRepoRoots(cfg Config) ([]string, error) {
	roots := make([]string, 0, len(cfg.Repo.Roots))
	for _, root := range cfg.Repo.Roots {
		expanded, err := ExpandPath(root)
		if err != nil {
			return nil, err
		}
		roots = append(roots, expanded)
	}
	return roots, nil
}
