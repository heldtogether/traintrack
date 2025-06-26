package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	DefaultConfigPath = filepath.Join(os.Getenv("HOME"), ".traintrack", "instance-config.json")
)

type InstanceConfig struct {
	URL string `json:"url"`
}

func SaveConfig(path string, conf *InstanceConfig) error {
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func LoadConfig(path string) (*InstanceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var stored InstanceConfig
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	return &stored, nil
}
