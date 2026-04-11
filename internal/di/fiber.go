package di

import (
	"context"
	"errors"
	"time"

	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go"
	goredislib "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/handler"
	"github.com/medincident/medincident-zitadel-actions/internal/middleware"
)

// fiberWrapper owns the Fiber App lifecycle. It implements
// do.ShutdownerWithContextAndError so samber/do drains in-flight
// requests via ShutdownWithContext on injector shutdown.
type fiberWrapper struct {
	app *fiber.App
}

func (w *fiberWrapper) Shutdown(ctx context.Context) error {
	return w.app.ShutdownWithContext(ctx)
}

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
	rc, err := do.Invoke[*goredislib.Client](injector)
	if err != nil {
		return nil, err
	}
	eh, err := do.Invoke[*handler.EventHandler](injector)
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

	app.Get("/health", handler.HealthCheck(nc, rc))

	post := app.Group("", middleware.ContentType(), middleware.HMACVerify(cfg.SigningKey, cfg.SigningKeyTolerance))

	post.Post("/debug", handler.PostDebugWebhook(logger))
	post.Post("/events/user/human/added", eh.PostHumanUserAdded())
	post.Post("/events/user/human/profile/changed", eh.PostHumanUserProfileChanged())
	post.Post("/events/user/human/email/changed", eh.PostHumanUserEmailChanged())
	post.Post("/events/user/human/email/verified", eh.PostHumanUserEmailVerified())
	post.Post("/events/session/added", eh.PostSessionAdded())
	post.Post("/events/session/user/checked", eh.PostSessionUserChecked())

	return &fiberWrapper{app: app}, nil
}

// errorHandler returns a Fiber ErrorHandler that routes *fiber.Error
// to the client with its declared status and body while logging every
// other error (including oops-wrapped internal failures) and replying
// with a bare 500. Client responses never expose internal error text.
func errorHandler(logger *zerolog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return c.Status(fe.Code).JSON(fiber.Map{
				"error": fe.Message,
			})
		}

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

		return c.SendStatus(fiber.StatusInternalServerError)
	}
}

func ProvideFiberApp(injector do.Injector) (*fiber.App, error) {
	w, err := do.Invoke[*fiberWrapper](injector)
	if err != nil {
		return nil, err
	}
	return w.app, nil
}
