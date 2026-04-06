package di

import (
	"context"
	"time"

	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/handler"
)

// fiberWrapper holds *fiber.App and implements do.ShutdownerWithContextAndError
// so samber/do gracefully shuts down the HTTP server.
type fiberWrapper struct {
	app *fiber.App
}

func (w *fiberWrapper) Shutdown(ctx context.Context) error {
	return w.app.ShutdownWithContext(ctx)
}

// ProvideFiberApp is a samber/do provider for *fiberWrapper.
func ProvideFiberApp(injector do.Injector) (*fiberWrapper, error) {
	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return nil, err
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BodyLimit:    64 * 1024,
	})

	app.Use(fiberzerolog.New(fiberzerolog.Config{Logger: logger}))

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Post("/events", handler.PostAnyEvent(logger))
	app.Post("/events/user/human/added", handler.PostHumanUserAdded(logger))
	app.Post("/events/user/human/profile/changed", handler.PostHumanUserProfileChanged(logger))
	app.Post("/requests", handler.PostAnyRequest(logger))
	app.Post("/responses", handler.PostAnyResponse(logger))

	return &fiberWrapper{app: app}, nil
}
