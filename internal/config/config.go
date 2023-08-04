package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RateIT struct {
	Upstream string `yaml:"upstream"`
}

type Telegram struct {
	APIKey string `yaml:"api_key"`
}

type History struct {
	Limit       int    `yaml:"limit"`
	StorageFile string `yaml:"storage_file"`
}

type Exchange struct {
	Name  string `yaml:"name"`
	Slug  string `yaml:"slug"`
	Route string `yaml:"route"`
}

type Config struct {
	Debug     bool       `yaml:"debug"`
	RateIT    RateIT     `yaml:"rate_it"`
	Telegram  Telegram   `yaml:"telegram"`
	History   History    `yaml:"history"`
	Exchanges []Exchange `yaml:"exchanges"`
}

func LoadConfig(configs ...string) (*Config, error) {
	out := &Config{
		Debug: false,
		RateIT: RateIT{
			Upstream: "http://127.0.0.1:3000",
		},
		Telegram: Telegram{
			APIKey: os.Getenv("TG_TOKEN"),
		},
		History: History{
			Limit: 48,
		},
		Exchanges: []Exchange{
			{
				Name:  "Contact (RU -> THB)",
				Slug:  "contact",
				Route: "contact/ru-th",
			},
			{
				Name:  "Korona (RU -> THB)",
				Slug:  "korona",
				Route: "korona/ru-th",
			},
		},
	}

	if len(configs) == 0 {
		return out, nil
	}

	for _, cfgPath := range configs {
		err := func() error {
			f, err := os.Open(cfgPath)
			if err != nil {
				return fmt.Errorf("unable to open config file: %w", err)
			}
			defer func() { _ = f.Close() }()

			if err := yaml.NewDecoder(f).Decode(&out); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			return nil
		}()
		if err != nil {
			return nil, fmt.Errorf("unable to load config %q: %w", cfgPath, err)
		}
	}

	return out, nil
}
