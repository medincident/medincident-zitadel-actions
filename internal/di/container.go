package di

import (
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/app"
	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/httpserver"
)

// NewContainer creates a fully wired samber/do injector from the given config.
func NewContainer(cfg *config.Config) do.Injector {
	injector := do.New()

	do.Provide(injector, func(_ do.Injector) (*config.Config, error) {
		return cfg, nil
	})
	do.Provide(injector, func(_ do.Injector) (*config.ZerologConfig, error) {
		return &cfg.Zerolog, nil
	})
	do.Provide(injector, ProvideZerolog)
	do.Provide(injector, provideApp)

	return injector
}

func provideApp(i do.Injector) (*app.App, error) {
	logger, err := do.Invoke[*zerolog.Logger](i)
	if err != nil {
		return nil, err
	}

	fiberApp, err := httpserver.New(
		httpserver.WithLogger(logger),
	)
	if err != nil {
		return nil, err
	}

	return app.New(fiberApp)
}
