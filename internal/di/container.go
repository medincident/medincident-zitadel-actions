// Package di wires the service's dependency graph via samber/do/v2.
//
// The container is built once from a loaded *config.Config. Providers are
// lifecycle-aware: anything that holds external resources (NATS, Redis, HTTP
// server, log file handles) implements the do shutdowner protocol so
// injector.ShutdownWithContext cleans up on exit.
package di

import (
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// NewContainer wires all providers and returns a ready-to-invoke injector.
func NewContainer(cfg *config.Config) (do.Injector, error) {
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

	return injector, nil
}
