package di

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// redisClientWrapper owns the *redis.Client lifecycle so samber/do can
// Close it on injector shutdown.
type redisClientWrapper struct {
	client *redis.Client
}

func (w *redisClientWrapper) Shutdown(_ context.Context) error {
	return w.client.Close()
}

func provideRedisClientWrapper(injector do.Injector) (*redisClientWrapper, error) {
	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		return nil, err
	}
	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	logger.Info().Str("address", cfg.Redis.Address).Msg("redis client configured")

	return &redisClientWrapper{client: client}, nil
}

func provideRedisClient(injector do.Injector) (*redis.Client, error) {
	w, err := do.Invoke[*redisClientWrapper](injector)
	if err != nil {
		return nil, err
	}
	return w.client, nil
}
