package obsidian

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	VaultPath          string   `json:"vault_path"`
	FileExtensions     []string `json:"file_extensions"`
	IgnoredDirectories []string `json:"ignored_directories"`
}

func LoadConfig(configPath string) (*Config, error) {
	// Default config
	config := &Config{
		VaultPath:          filepath.Join("/", "mnt", "c", "Users", "chtur", "OneDrive", "Documents", "Obsidian", "MyVault"),
		FileExtensions:     []string{".md", ".txt"},
		IgnoredDirectories: []string{".obsidian", ".git", ".trash", "node_modules"},
	}

	// Try to load from file
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}

		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	return config, nil
}

func (c *Config) SaveConfig(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
