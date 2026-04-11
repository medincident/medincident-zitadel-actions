package di

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// Error codes emitted by this DI init. Declared at file level so each emit site
// is grep-local; string values carry the component name so interceptors can
// distinguish per-component telemetry.
const ErrCodeRedisConnectFailed = "connect_failed"

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
		_ = client.Close()
		return nil, oops.In("redis").Code(ErrCodeRedisConnectFailed).With("address", cfg.Redis.Address).Wrap(err)
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
