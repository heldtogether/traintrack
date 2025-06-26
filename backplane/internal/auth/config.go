package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	DefaultConfigPath = filepath.Join(os.Getenv("HOME"), ".traintrack", "oauth-client-config.json")
)

type OAuthProviderConfig struct {
	Name     string `json:"name"`
	ClientID string `json:"client_id"`
	AuthURL  string `json:"auth_url"`
}

func SaveConfig(path string, conf *OAuthProviderConfig) error {
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func LoadConfig(path string) (*OAuthProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var stored OAuthProviderConfig
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	return &stored, nil
}
