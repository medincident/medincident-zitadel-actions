# Agent Instructions — medincident-zitadel-actions

HTTP gateway: **Zitadel Actions v2 → NATS JetStream**, written in Go 1.26.1.

---

## Project layout

```
cmd/server/main.go                          — entry point
internal/
  app/                                      — App struct (fiber wrapper, graceful shutdown)
  config/                                   — YAML config types + reader
  di/                                       — samber/do providers (container.go, zerolog.go)
  httpserver/                               — fiber.App factory + route registration
  httpserver/events/                        — HTTP handlers for Zitadel webhook events
  log/                                      — zerolog builder (multi-output, per-output level filter)
  zitadel/actions/events/                   — Envelope, Unmarshal[T], event payload structs
api/proto/                                  — buf.yaml, buf.gen.yaml; .proto files go here
pkg/                                        — buf-generated Go code (import from here)
configs/config.example.yaml                — annotated config reference
```

---

## Key libraries

| Library | Purpose |
|---|---|
| `github.com/gofiber/fiber/v3` | HTTP server |
| `github.com/rs/zerolog` | Structured JSON logging |
| `github.com/samber/do/v2` | Dependency injection container |
| `github.com/samber/oops` | Structured errors with stack traces |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/bufbuild/buf` (go tool) | Protobuf tooling |
| `google.golang.org/protobuf/cmd/protoc-gen-go` (go tool) | Proto → Go codegen |
| `github.com/abice/go-enum` (go tool) | Go enum codegen |

NATS (`nats.go`) is **not yet in go.mod** — add it when implementing the JetStream integration.

---

## Tooling

```bash
task generate   # buf generate + go generate (runs from api/proto/)
task fmt        # buf format -w
task lint       # buf lint
```

No Makefile exists yet. Do not add `generate`, `fmt`, or `lint` targets to a Makefile — those belong in Taskfile.yml only.

---

## Configuration

Config is a YAML file passed via `-config <path>` (default: `config.yaml`).
`${VAR}` and `$VAR` references are expanded from the process environment before parsing.
See `configs/config.example.yaml` for the full annotated reference.

---

## Dependency injection pattern

1. `di.NewContainer(cfg)` registers all `do.Provide` calls.
2. `do.Invoke[*app.App](injector)` bootstraps the full dependency graph.
3. External deps that need cleanup implement `Shutdown(ctx context.Context) error` — samber/do calls them on `injector.ShutdownWithContext(ctx)`.

When adding a new infrastructure component (e.g. NATS client):
- Create `internal/<component>/` with a struct wrapping the client.
- Add a `Shutdown(ctx context.Context) error` method.
- Register a `do.Provide` in `di/container.go`.

---

## HTTP handler pattern

Handlers live in `internal/httpserver/events/`. Each handler is a private constructor returning `fiber.Handler`:

```go
func makeFooHandler(logger *zerolog.Logger) fiber.Handler {
    return func(c fiber.Ctx) error { ... }
}
```

Register in `events.SetupRoutes`. Pass only what the handler needs (logger, publisher, etc.) — no god objects.

---

## Error handling

Use `samber/oops` for all error construction:

```go
return oops.In("component").Code("snake_case_code").With("key", val).Wrap(err)
// or
return oops.In("component").Code("snake_case_code").Errorf("message")
```

Never use `fmt.Errorf` with `%w` — oops handles wrapping and stack traces.

---

## Logging

Get the logger from the DI container:

```go
logger := do.MustInvoke[*zerolog.Logger](injector)
```

Log structured fields, not formatted strings:

```go
logger.Info().Str("user_id", id).Str("event_type", t).Msg("received event")
```

---

## Zitadel event envelope

Every Zitadel Actions v2 POST body deserialises into `zitadel/actions/events.Envelope`.
The `event_payload` field is base64-encoded JSON — use the generic `Unmarshal[T]` to decode it:

```go
envelope := new(events.Envelope)
c.Bind().Body(envelope)

payload := new(events.UserHumanAdded)
events.Unmarshal(envelope, payload)
```

---

## Protobuf

Proto source files belong in `api/proto/medincident/zitadel/v1/`.
Generated Go code lands in `pkg/medincident/zitadel/v1/` (set by `buf.gen.yaml`).
After editing `.proto` files run `task generate`.

---

## What is not done yet (TODO.md)

- NATS JetStream integration — `internal/nats/`, publisher wired into handlers
- Protobuf message definitions and codegen
- Webhook HMAC signature verification middleware
- Rate limiting middleware
- Handlers for `/user/profile`, `/user/email`, `/user/idp`
- Tests (unit: `Envelope.Unmarshal`; integration: handlers)
- Makefile (`build` / `run` targets)
- TLS termination strategy
