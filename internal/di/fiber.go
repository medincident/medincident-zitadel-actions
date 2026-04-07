package di

import (
	"context"
	"errors"
	"time"

	"github.com/go-redsync/redsync/v4"
	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	goredislib "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/handler"
	"github.com/medincident/medincident-zitadel-actions/internal/middleware"
)

// fiberWrapper holds *fiber.App and implements do.ShutdownerWithContextAndError
// so samber/do gracefully shuts down the HTTP server.
type fiberWrapper struct {
	app *fiber.App
}

func (w *fiberWrapper) Shutdown(ctx context.Context) error {
	return w.app.ShutdownWithContext(ctx)
}

// ProvideFiberWrapper is a samber/do provider for *fiberWrapper.
func ProvideFiberWrapper(injector do.Injector) (*fiberWrapper, error) {
	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		return nil, err
	}
	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return nil, err
	}
	nc, err := do.Invoke[*nats.Conn](injector)
	if err != nil {
		return nil, err
	}
	js, err := do.Invoke[jetstream.JetStream](injector)
	if err != nil {
		return nil, err
	}
	rc, err := do.Invoke[*goredislib.Client](injector)
	if err != nil {
		return nil, err
	}
	rs, err := do.Invoke[*redsync.Redsync](injector)
	if err != nil {
		return nil, err
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BodyLimit:    64 * 1024,
		ErrorHandler: errorHandler(logger),
	})

	app.Use(fiberzerolog.New(fiberzerolog.Config{Logger: logger}))

	// Health check (no middleware).
	app.Get("/health", handler.HealthCheck(nc, rc))

	// POST routes with ContentType + HMAC middleware.
	post := app.Group("", middleware.ContentType(), middleware.HMACVerify(cfg.SigningKey, cfg.SigningKeyTolerance.Duration()))

	post.Post("/debug", handler.PostDebugWebhook(logger))
	post.Post("/events/user/human/added", handler.PostHumanUserAdded(logger, js, rs, cfg))
	post.Post("/events/user/human/profile/changed", handler.PostHumanUserProfileChanged(logger, js, rs, cfg))

	return &fiberWrapper{app: app}, nil
}

// errorHandler returns a Fiber ErrorHandler that distinguishes between
// client errors (*fiber.Error) and internal errors (oops / unknown).
func errorHandler(logger *zerolog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return c.Status(fe.Code).JSON(fiber.Map{
				"error": fe.Message,
			})
		}

		// Log internal errors with oops context for debugging.
		if oopsErr, ok := oops.AsOops(err); ok {
			logger.Error().
				Str("method", c.Method()).
				Str("path", c.Path()).
				Any("oops_code", oopsErr.Code()).
				Str("oops_domain", oopsErr.Domain()).
				Any("oops_context", oopsErr.Context()).
				Str("stacktrace", oopsErr.Stacktrace()).
				Err(err).
				Msg("internal error")
		} else {
			logger.Error().
				Str("method", c.Method()).
				Str("path", c.Path()).
				Err(err).
				Msg("internal error")
		}

		// Never leak internals to the client.
		return c.SendStatus(fiber.StatusInternalServerError)
	}
}

// ProvideFiberApp is a samber/do provider for *fiber.App.
// It delegates to *fiberWrapper, which owns the lifecycle.
func ProvideFiberApp(injector do.Injector) (*fiber.App, error) {
	w, err := do.Invoke[*fiberWrapper](injector)
	if err != nil {
		return nil, err
	}
	return w.app, nil
}
