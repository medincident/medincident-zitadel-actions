package httpserver

import (
	"time"

	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	"github.com/medincident/medincident-zitadel-actions/internal/httpserver/events"
)

type serverConfig struct {
	logger   *zerolog.Logger
	fiberCfg fiber.Config
}

func defaultServerConfig() *serverConfig {
	return &serverConfig{
		fiberCfg: fiber.Config{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			BodyLimit:    64 * 1024,
		},
	}
}

// Option configures the HTTP server.
type Option func(*serverConfig) error

// WithLogger attaches a zerolog logger for request logging middleware.
func WithLogger(logger *zerolog.Logger) Option {
	return func(cfg *serverConfig) error {
		cfg.logger = logger
		return nil
	}
}

// WithTimeout sets the read and write timeouts on the underlying fiber server.
// Defaults: ReadTimeout 10s, WriteTimeout 10s (set by defaultServerConfig).
func WithTimeout(readTimeout, writeTimeout time.Duration) Option {
	return func(cfg *serverConfig) error {
		cfg.fiberCfg.ReadTimeout = readTimeout
		cfg.fiberCfg.WriteTimeout = writeTimeout
		return nil
	}
}

// WithBodyLimit sets the maximum allowed size of a request body in bytes.
// Default: 65536 (64KB), appropriate for Zitadel webhook payloads.
func WithBodyLimit(limitBytes int) Option {
	return func(cfg *serverConfig) error {
		cfg.fiberCfg.BodyLimit = limitBytes
		return nil
	}
}

// New creates a *fiber.App with all routes registered.
func New(opts ...Option) (*fiber.App, error) {
	cfg := defaultServerConfig()
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	app := fiber.New(cfg.fiberCfg)

	if cfg.logger != nil {
		app.Use(fiberzerolog.New(fiberzerolog.Config{
			Logger: cfg.logger,
		}))
	}

	events.SetupRoutes(app, cfg.logger)

	return app, nil
}
