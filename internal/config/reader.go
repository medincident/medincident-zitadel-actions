package config

import (
	"os"

	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

// Config is the root application configuration.
type Config struct {
	Zerolog ZerologConfig `yaml:"zerolog"`
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

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, oops.
			In("config").
			Code("unmarshal_failed").
			With("path", path).
			Wrap(err)
	}

	return &cfg, nil
}
