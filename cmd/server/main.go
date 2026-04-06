package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	injector := di.NewContainer(cfg)

	app := di.MustStart(injector)
	logger := do.MustInvoke[*zerolog.Logger](injector)

	logger.Info().Str("addr", ":8080").Msg("starting server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan error, 1)

	go func() {
		done <- app.Listen(":8080")
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error().Err(err).Msg("server error")
			return err
		}
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if report := injector.ShutdownWithContext(ctx); !report.Succeed {
			logger.Error().Err(report).Msg("shutdown error")
		}
	}

	logger.Info().Msg("server stopped")

	return nil
}
