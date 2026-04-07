package di

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// redisClientWrapper holds *redis.Client and implements do.ShutdownerWithContextAndError.
type redisClientWrapper struct {
	client *redis.Client
}

func (w *redisClientWrapper) Shutdown(_ context.Context) error {
	return w.client.Close()
}

// ProvideRedisClientWrapper is a samber/do provider for *redisClientWrapper.
func ProvideRedisClientWrapper(injector do.Injector) (*redisClientWrapper, error) {
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

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, oops.In("redis").Code("connect_failed").With("address", cfg.Redis.Address).Wrap(err)
	}

	logger.Info().Str("address", cfg.Redis.Address).Msg("connected to Redis")

	return &redisClientWrapper{client: client}, nil
}

// ProvideRedisClient is a samber/do provider for *redis.Client.
func ProvideRedisClient(injector do.Injector) (*redis.Client, error) {
	w, err := do.Invoke[*redisClientWrapper](injector)
	if err != nil {
		return nil, err
	}
	return w.client, nil
}
