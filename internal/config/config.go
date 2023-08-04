package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type RateIT struct {
	Upstream string `yaml:"upstream"`
}

type Telegram struct {
	APIKey string `yaml:"api_key"`
}

type History struct {
	StorageFile string `yaml:"storage_file"`
}

type Exchange struct {
	Name  string `yaml:"name"`
	Slug  string `yaml:"slug"`
	Route string `yaml:"route"`
}

type HistoryLimits struct {
	Overall int `yaml:"overall"`
	Short   int `yaml:"short"`
	Long    int `yaml:"long"`
}

type Limits struct {
	History HistoryLimits `yaml:"history"`
}

type Notification struct {
	Threshold float64 `yaml:"threshold"`
	ChatID    int     `yaml:"chat_id"`
}

type Notifier struct {
	CheckPeriod   time.Duration  `yaml:"check_period"`
	Notifications []Notification `yaml:"notifications"`
}

type Config struct {
	Debug     bool       `yaml:"debug"`
	RateIT    RateIT     `yaml:"rate_it"`
	Telegram  Telegram   `yaml:"telegram"`
	Notifier  Notifier   `yaml:"notifier"`
	History   History    `yaml:"history"`
	Exchanges []Exchange `yaml:"exchanges"`
	Limits    Limits     `yaml:"limits"`
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
		Limits: Limits{
			History: HistoryLimits{
				Overall: 1000,
				Short:   72,
				Long:    0,
			},
		},
		Notifier: Notifier{
			CheckPeriod: 10 * time.Minute,
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
