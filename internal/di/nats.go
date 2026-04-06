package di

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// natsConnWrapper holds *nats.Conn and implements do.ShutdownerWithContextAndError.
type natsConnWrapper struct {
	conn *nats.Conn
}

func (w *natsConnWrapper) Shutdown(_ context.Context) error {
	w.conn.Close()
	return nil
}

// ProvideNatsConnWrapper is a samber/do provider for *natsConnWrapper.
func ProvideNatsConnWrapper(injector do.Injector) (*natsConnWrapper, error) {
	cfg := do.MustInvoke[*config.Config](injector)
	logger := do.MustInvoke[*zerolog.Logger](injector)

	nc, err := nats.Connect(cfg.Nats.URL)
	if err != nil {
		return nil, err
	}

	logger.Info().Str("url", cfg.Nats.URL).Msg("connected to NATS")

	return &natsConnWrapper{conn: nc}, nil
}

// ProvideNatsConn is a samber/do provider for *nats.Conn.
func ProvideNatsConn(injector do.Injector) (*nats.Conn, error) {
	w, err := do.Invoke[*natsConnWrapper](injector)
	if err != nil {
		return nil, err
	}
	return w.conn, nil
}

// ProvideJetStream is a samber/do provider for jetstream.JetStream.
func ProvideJetStream(injector do.Injector) (jetstream.JetStream, error) {
	nc, err := do.Invoke[*nats.Conn](injector)
	if err != nil {
		return nil, err
	}
	return jetstream.New(nc)
}
