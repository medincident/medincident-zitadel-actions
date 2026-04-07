package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration with YAML string unmarshaling support.
// Accepts Go duration strings like "5m", "200ms", "1h30m".
type Duration time.Duration

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", s, err)
		}
		*d = Duration(parsed)
		return nil
	}

	// Fallback: numeric value as nanoseconds.
	var ns int64
	if err := value.Decode(&ns); err != nil {
		return fmt.Errorf("cannot decode duration: expected string or integer")
	}
	*d = Duration(ns)
	return nil
}
