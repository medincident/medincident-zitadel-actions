package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/di"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Read(*configPath)
	if err != nil {
		return err
	}

	injector, err := di.NewContainer(cfg)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if report := injector.ShutdownWithContext(ctx); !report.Succeed {
			fmt.Fprintf(os.Stderr, "shutdown error: %v\n", report)
		}
	}()

	app, err := do.Invoke[*fiber.App](injector)
	if err != nil {
		return err
	}

	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return err
	}

	logger.Info().Str("addr", cfg.Address).Msg("starting server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan error, 1)

	go func() {
		done <- app.Listen(cfg.Address)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error().Err(err).Msg("server error")
			return err
		}
	case <-quit:
		logger.Info().Msg("shutting down")
	}

	return nil
}
