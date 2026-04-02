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

	"github.com/medincident/medincident-zitadel-actions/internal/app"
	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/di"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Read(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		os.Exit(1)
	}

	injector := di.NewContainer(cfg)

	application, err := do.Invoke[*app.App](injector)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	logger := do.MustInvoke[*zerolog.Logger](injector)
	logger.Info().Str("addr", ":8080").Msg("starting server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		report := injector.ShutdownWithContext(ctx)
		if !report.Succeed {
			logger.Error().Err(report).Msg("shutdown error")
		}
	}()

	if err := application.Run(); err != nil {
		logger.Error().Err(err).Msg("server error")
		os.Exit(1)
	}

	logger.Info().Msg("server stopped")
}
