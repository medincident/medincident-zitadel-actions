package config

import (
	"errors"
	"os"
	"sort"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

const (
	ErrCodeConfigReadFailed      = "read_failed"
	ErrCodeConfigUnmarshalFailed = "unmarshal_failed"
	ErrCodeConfigValidateFailed  = "validate_failed"
)

// Config is the root application configuration.
type Config struct {
	Address             string        `yaml:"address" validate:"required,hostname_port"`
	SigningKey          string        `yaml:"signing_key"`
	SigningKeyTolerance time.Duration `yaml:"signing_key_tolerance" validate:"required,min=1s,max=1h"`
	Nats                NatsConfig    `yaml:"nats" validate:"required"`
	Redis               RedisConfig   `yaml:"redis" validate:"required"`
	Publish             PublishConfig `yaml:"publish" validate:"required"`
	Zerolog             ZerologConfig `yaml:"zerolog" validate:"required"`
}

// NatsConfig holds NATS connection settings.
type NatsConfig struct {
	URL           string        `yaml:"url" validate:"required,url"`
	MaxReconnects int           `yaml:"max_reconnects" validate:"min=-1"`
	ReconnectWait time.Duration `yaml:"reconnect_wait" validate:"required,min=100ms"`
}

// RedisConfig holds Redis connection and distributed lock settings.
type RedisConfig struct {
	Address    string        `yaml:"address" validate:"required,hostname_port"`
	Password   string        `yaml:"password"`
	DB         int           `yaml:"db" validate:"min=0,max=15"`
	LockPrefix string        `yaml:"lock_prefix" validate:"required"`
	LockExpiry time.Duration `yaml:"lock_expiry" validate:"required,min=1s,max=5m"`
}

// PublishConfig holds NATS JetStream publish retry settings.
type PublishConfig struct {
	MaxRetries     uint          `yaml:"max_retries" validate:"min=0,max=100"`
	InitialBackoff time.Duration `yaml:"initial_backoff" validate:"required,min=1ms"`
	MaxBackoff     time.Duration `yaml:"max_backoff" validate:"required,min=1ms,gtefield=InitialBackoff"`
	MaxElapsedTime time.Duration `yaml:"max_elapsed_time" validate:"required,min=10ms,gtefield=MaxBackoff"`
}

func defaultConfig() Config {
	return Config{
		Address:             ":8080",
		SigningKeyTolerance: 5 * time.Minute,
		Nats: NatsConfig{
			URL:           "nats://localhost:4222",
			MaxReconnects: -1,
			ReconnectWait: 2 * time.Second,
		},
		Redis: RedisConfig{
			Address:    "localhost:6379",
			LockPrefix: "medincident:lock:",
			LockExpiry: 30 * time.Second,
		},
		Publish: PublishConfig{
			MaxRetries:     5,
			InitialBackoff: 200 * time.Millisecond,
			MaxBackoff:     5 * time.Second,
			MaxElapsedTime: 8 * time.Second,
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

// newValidator returns a configured validator instance. A new one is
// created per Read call rather than stored as package state so
// there's no hidden global to reason about.
func newValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}

// formatValidationErrors walks validator.ValidationErrors and returns a
// sorted []string like "Nats.URL: required", "Redis.LockExpiry: min=1s".
// If err is not validator.ValidationErrors, returns []string{err.Error()}.
func formatValidationErrors(err error) []string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []string{err.Error()}
	}

	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		// StructNamespace looks like "Config.Nats.URL" — strip the root type prefix.
		ns := fe.StructNamespace()
		dot := len("Config.")
		if len(ns) > dot {
			ns = ns[dot:]
		}

		tag := fe.Tag()
		param := fe.Param()
		if param != "" {
			tag = tag + "=" + param
		}

		msgs = append(msgs, ns+": "+tag)
	}

	sort.Strings(msgs)
	return msgs
}

// Read loads a YAML config file from path and expands ${VAR} / $VAR
// references in its content using the current process environment.
func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, oops.
			In("config").
			Code(ErrCodeConfigReadFailed).
			With("path", path).
			Wrap(err)
	}

	expanded := os.ExpandEnv(string(data))

	cfg := defaultConfig()
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, oops.
			In("config").
			Code(ErrCodeConfigUnmarshalFailed).
			With("path", path).
			Wrap(err)
	}

	if err := newValidator().Struct(&cfg); err != nil {
		return nil, oops.
			In("config").
			Code(ErrCodeConfigValidateFailed).
			With("path", path).
			With("violations", formatValidationErrors(err)).
			Wrap(err)
	}

	return &cfg, nil
}
