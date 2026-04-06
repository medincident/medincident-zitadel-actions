package di

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// NewContainer creates a fully wired samber/do injector from the given config.
func NewContainer(cfg *config.Config) do.Injector {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.Provide(injector, ProvideLoggerWrapper)
	do.Provide(injector, ProvideZerolog)
	do.Provide(injector, ProvideFiberApp)

	return injector
}

// MustStart invokes *fiber.App from the container, triggering the full dependency graph.
func MustStart(injector do.Injector) *fiber.App {
	return do.MustInvoke[*fiber.App](injector)
}
