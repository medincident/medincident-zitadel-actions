package di

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredislib "github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
)

func ProvideRedsync(injector do.Injector) (*redsync.Redsync, error) {
	client, err := do.Invoke[*goredislib.Client](injector)
	if err != nil {
		return nil, err
	}

	pool := goredis.NewPool(client)
	return redsync.New(pool), nil
}
