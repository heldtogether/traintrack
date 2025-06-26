package cmd

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/heldtogether/traintrack/internal/auth"
)

var (
	DefaultConfigPath = filepath.Join(os.Getenv("HOME"), ".traintrack", "instance-config.json")
)

type InstanceConfig struct {
	URL         string    `json:"url"`
	LastFetched time.Time `json:"last_fetched"`
}

func (c *InstanceConfig) refreshAuthConfig() *InstanceConfig {
	if time.Since(c.LastFetched) >= 4*time.Hour {
		base, err := url.Parse(c.URL)
		if err != nil {
			log.Fatalf("invalid base URL in config: %s", err)
		}
		base.Path = path.Join(base.Path, ".well-known", "oauth-client-config")
		url := base.String()

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("unable to refresh oauth client config: %s", err.Error())
			return c
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf(
				"unable to refresh oauth client config: %d - %s",
				resp.StatusCode,
				http.StatusText(resp.StatusCode),
			)
			return c
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("unable to refresh oauth client config: %s", err.Error())
			return c
		}

		var data auth.OAuthProviderConfig
		if err := json.Unmarshal(body, &data); err != nil {
			log.Printf("unable to refresh oauth client config: %s", err.Error())
			return c
		}
		auth.SaveConfig(auth.DefaultConfigPath, &data)

		c.LastFetched = time.Now()
		err = SaveConfig(DefaultConfigPath, c)
		if err != nil {
			log.Printf("unable to refresh oauth client config: %s", err.Error())
			return c
		}
	}
	return c
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

	config := stored.refreshAuthConfig()

	return config, nil
}
