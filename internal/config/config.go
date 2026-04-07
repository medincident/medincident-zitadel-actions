package config

import (
	"os"
	"time"

	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

// Config is the root application configuration.
type Config struct {
	Address             string        `yaml:"address"`
	SigningKey          string        `yaml:"signing_key"`
	SigningKeyTolerance Duration      `yaml:"signing_key_tolerance"`
	Nats                NatsConfig    `yaml:"nats"`
	Redis               RedisConfig   `yaml:"redis"`
	Publish             PublishConfig `yaml:"publish"`
	Zerolog             ZerologConfig `yaml:"zerolog"`
}

// NatsConfig holds NATS connection settings.
type NatsConfig struct {
	URL           string   `yaml:"url"`
	MaxReconnects int      `yaml:"max_reconnects"`
	ReconnectWait Duration `yaml:"reconnect_wait"`
}

// RedisConfig holds Redis connection and distributed lock settings.
type RedisConfig struct {
	Address    string   `yaml:"address"`
	Password   string   `yaml:"password"`
	DB         int      `yaml:"db"`
	LockPrefix string   `yaml:"lock_prefix"`
	LockExpiry Duration `yaml:"lock_expiry"`
}

// PublishConfig holds NATS JetStream publish retry settings.
type PublishConfig struct {
	MaxRetries     uint     `yaml:"max_retries"`
	InitialBackoff Duration `yaml:"initial_backoff"`
	MaxBackoff     Duration `yaml:"max_backoff"`
}

func defaultConfig() Config {
	return Config{
		Address:             ":8080",
		SigningKeyTolerance: Duration(5 * time.Minute),
		Nats: NatsConfig{
			URL:           "nats://localhost:4222",
			MaxReconnects: -1,
			ReconnectWait: Duration(2 * time.Second),
		},
		Redis: RedisConfig{
			Address:    "localhost:6379",
			LockPrefix: "medincident:lock:",
			LockExpiry: Duration(30 * time.Second),
		},
		Publish: PublishConfig{
			MaxRetries:     5,
			InitialBackoff: Duration(200 * time.Millisecond),
			MaxBackoff:     Duration(5 * time.Second),
		},
		Zerolog: ZerologConfig{
			Level:      "info",
			Timestamp:  true,
			TimeFormat: time.RFC3339,
			Outputs: []ZerologOutputConfig{
				{
					Type:       ZerologOutputTypeConsole,
					Target:     ZerologConsoleTargetStdout,
					Pretty:     true,
					TimeFormat: "15:04:05",
					PartsOrder: []string{"time", "level", "caller", "message"},
				},
			},
		},
	}
}

// Read loads a YAML config file from path and expands ${VAR} / $VAR
// references in its content using the current process environment.
func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, oops.
			In("config").
			Code("read_failed").
			With("path", path).
			Wrap(err)
	}

	expanded := os.ExpandEnv(string(data))

	cfg := defaultConfig()
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, oops.
			In("config").
			Code("unmarshal_failed").
			With("path", path).
			Wrap(err)
	}

	return &cfg, nil
}
