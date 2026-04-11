package di

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// Error codes emitted by this DI init. Declared at file level so each emit site
// is grep-local; string values carry the component name so interceptors can
// distinguish per-component telemetry.
const ErrCodeNATSConnectFailed = "nats_connect_failed"

// natsConnWrapper holds *nats.Conn and implements do.ShutdownerWithContextAndError.
type natsConnWrapper struct {
	conn *nats.Conn
}

func (w *natsConnWrapper) Shutdown(_ context.Context) error {
	return w.conn.Drain()
}

// ProvideNatsConnWrapper is a samber/do provider for *natsConnWrapper.
func ProvideNatsConnWrapper(injector do.Injector) (*natsConnWrapper, error) {
	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		return nil, err
	}
	logger, err := do.Invoke[*zerolog.Logger](injector)
	if err != nil {
		return nil, err
	}

	nc, err := nats.Connect(cfg.Nats.URL,
		nats.MaxReconnects(cfg.Nats.MaxReconnects),
		nats.ReconnectWait(cfg.Nats.ReconnectWait.Duration()),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn().Err(err).Msg("nats disconnected")
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info().Msg("nats reconnected")
		}),
	)
	if err != nil {
		return nil, oops.In("nats").Code(ErrCodeNATSConnectFailed).With("url", cfg.Nats.URL).Wrap(err)
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
