package config

import (
	"os"
	"time"

	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

// Config is the root application configuration.
type Config struct {
	Address string        `yaml:"address"`
	Zerolog ZerologConfig `yaml:"zerolog"`
}

func defaultConfig() Config {
	return Config{
		Address: ":8080",
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
