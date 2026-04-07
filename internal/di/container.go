package di

import (
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// NewContainer creates a fully wired samber/do injector from the given config.
func NewContainer(cfg *config.Config) do.Injector {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.Provide(injector, ProvideLoggerWrapper)
	do.Provide(injector, ProvideZerolog)
	do.Provide(injector, ProvideRedisClientWrapper)
	do.Provide(injector, ProvideRedisClient)
	do.Provide(injector, ProvideRedsync)
	do.Provide(injector, ProvideNatsConnWrapper)
	do.Provide(injector, ProvideNatsConn)
	do.Provide(injector, ProvideJetStream)
	do.Provide(injector, ProvidePublisher)
	do.Provide(injector, ProvideEventHandler)
	do.Provide(injector, ProvideFiberWrapper)
	do.Provide(injector, ProvideFiberApp)

	return injector
}
