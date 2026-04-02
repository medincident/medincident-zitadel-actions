package app

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"
)

type appConfig struct {
	addr string
}

func defaultAppConfig() appConfig {
	return appConfig{
		addr: ":8080",
	}
}

// Option configures the application.
type Option func(*appConfig)

// WithAddr sets the listen address (default: ":8080").
func WithAddr(addr string) Option {
	return func(cfg *appConfig) { cfg.addr = addr }
}

// App wraps a Fiber server and implements do.ShutdownerWithContextAndError.
type App struct {
	fiber *fiber.App
	cfg   appConfig
}

// New creates an App from an already-configured *fiber.App.
func New(fiberApp *fiber.App, opts ...Option) (*App, error) {
	if fiberApp == nil {
		return nil, oops.
			In("app").
			Code("nil_fiber_app").
			Hint("Pass a *fiber.App created via httpserver.New()").
			Errorf("fiber.App must not be nil")
	}

	cfg := defaultAppConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &App{fiber: fiberApp, cfg: cfg}, nil
}

// Run starts the HTTP server and blocks until it stops.
func (a *App) Run() error {
	return oops.
		In("app").
		Code("listen_failed").
		With("addr", a.cfg.addr).
		Wrap(a.fiber.Listen(a.cfg.addr))
}

// Shutdown gracefully stops the server (implements do.ShutdownerWithContextAndError).
func (a *App) Shutdown(ctx context.Context) error {
	return oops.
		In("app").
		Code("shutdown_failed").
		Wrap(a.fiber.ShutdownWithContext(ctx))
}
