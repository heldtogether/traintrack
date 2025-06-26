package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var (
	DefaultConfigPath = filepath.Join(os.Getenv("HOME"), ".traintrack", "oauth-client-config.json")
	ProjectConfigPath = filepath.Join(".traintrack", "oauth-client-config.json")
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
	// 1. Check environment variables
	authName := os.Getenv("TRAINTRACK_AUTH_NAME")
	clientID := os.Getenv("TRAINTRACK_CLIENT_ID")
	authURL := os.Getenv("TRAINTRACK_AUTH_URL")

	if clientID != "" && authURL != "" && authName != "" {
		return &OAuthProviderConfig{
			Name:     authName,
			ClientID: clientID,
			AuthURL:  authURL,
		}, nil
	}

	// 2. Check project-level config
	if conf, err := loadFromFile(ProjectConfigPath); err == nil {
		return conf, nil
	}

	// 3. Check home-level config
	if conf, err := loadFromFile(DefaultConfigPath); err == nil {
		return conf, nil
	}

	return nil, errors.New("no valid config found in environment variables or config files")
}

func loadFromFile(path string) (*OAuthProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf OAuthProviderConfig
	if err := json.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
