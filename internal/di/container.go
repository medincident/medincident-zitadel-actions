// Package di wires the service's dependency graph via samber/do/v2.
//
// The container is built once from a loaded *config.Config. Providers
// are lifecycle-aware: anything that holds external resources (NATS,
// Redis, HTTP server, log file handles) is held by a private wrapper
// that implements the do Shutdowner protocol. Consumers depend on the
// real type (e.g. *nats.Conn, *fiber.App), not the wrapper.
//
// Provider functions are all unexported — NewContainer is the only
// public entry point. Consumers resolve dependencies via do.Invoke on
// the returned injector, keyed by the real type.
package di

import (
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// NewContainer wires every provider and returns a ready-to-invoke
// injector. Provider order here is irrelevant — samber/do resolves
// dependencies lazily on Invoke.
func NewContainer(cfg *config.Config) (do.Injector, error) {
	injector := do.New()

	do.ProvideValue(injector, cfg)

	do.Provide(injector, provideLoggerWrapper)
	do.Provide(injector, provideZerolog)
	do.Provide(injector, provideRedisClientWrapper)
	do.Provide(injector, provideRedisClient)
	do.Provide(injector, provideRedsync)
	do.Provide(injector, provideNatsConnWrapper)
	do.Provide(injector, provideNatsConn)
	do.Provide(injector, provideJetStream)
	do.Provide(injector, providePublisher)
	do.Provide(injector, provideEventHandler)
	do.Provide(injector, provideFiberWrapper)
	do.Provide(injector, provideFiberApp)

	return injector, nil
}
