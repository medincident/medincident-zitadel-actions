package di

import (
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
)

// ProvidePublisher is a samber/do provider for *publish.Publisher.
func ProvidePublisher(injector do.Injector) (*publish.Publisher, error) {
	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return nil, err
	}
	js, err := do.Invoke[jetstream.JetStream](injector)
	if err != nil {
		return nil, err
	}
	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		return nil, err
	}

	return publish.NewPublisher(logger, js, cfg.Publish), nil
}
