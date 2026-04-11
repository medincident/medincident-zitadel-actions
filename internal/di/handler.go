package di

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/handler"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
)

func ProvideEventHandler(injector do.Injector) (*handler.EventHandler, error) {
	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return nil, err
	}
	pub, err := do.Invoke[*publish.Publisher](injector)
	if err != nil {
		return nil, err
	}
	rs, err := do.Invoke[*redsync.Redsync](injector)
	if err != nil {
		return nil, err
	}
	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		return nil, err
	}

	return handler.NewEventHandler(logger, pub, rs, cfg), nil
}
