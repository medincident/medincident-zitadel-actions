# Agent Instructions — medincident-zitadel-actions

HTTP gateway: **Zitadel Actions v2 → NATS JetStream**, written in Go 1.26.2.

---

## Project layout

```
cmd/server/main.go                          — entry point, graceful shutdown
api/proto/                                  — protobuf source files (buf v2 module)
  buf.yaml                                  — buf module config (lint, breaking)
  events/v1/                                — event envelope proto
  sessions/v1/                              — session event protos
  users/v1/                                 — user event + type protos
pkg/                                        — buf-generated Go code (import from here)
  events/v1/                                — event envelope pb.go
  sessions/v1/                              — session event pb.go
  users/v1/                                 — user event + type pb.go
buf.gen.yaml                                — buf codegen config (local input, managed mode)
buf.gen.docs.yaml                           — buf doc generation config
internal/
  config/                                   — YAML config types + reader
  di/                                       — samber/do providers (container, zerolog, fiber, nats, redis)
  handler/                                  — HTTP handlers for all Zitadel webhook endpoints
  mapper/                                   — Zitadel → proto event mappers
  middleware/                               — Fiber middleware (ContentType, HMAC)
  publish/                                  — NATS JetStream publishing helper
  zitadel/                                  — Envelope[T], event payload structs
configs/config.example.yaml                — annotated config reference
test/integration/zitadel/                   — integration tests (testcontainers-go)
  data/                                     — Zitadel config & steps YAML for tests
```

---

## Key libraries

| Library | Purpose |
|---|---|
| `github.com/gofiber/fiber/v3` | HTTP server |
| `github.com/rs/zerolog` | Structured JSON logging |
| `github.com/samber/do/v2` | Dependency injection container |
| `github.com/samber/oops` | Structured errors with stack traces |
| `github.com/nats-io/nats.go` | NATS client + JetStream |
| `github.com/google/uuid` | UUID generation for event IDs |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/bufbuild/buf` (go tool) | Protobuf tooling |
| `google.golang.org/protobuf/cmd/protoc-gen-go` (go tool) | Proto → Go codegen |
| `github.com/pseudomuto/protoc-gen-doc` (go tool) | Proto → Markdown doc generation |
| `github.com/abice/go-enum` (go tool) | Go enum codegen |
| `github.com/testcontainers/testcontainers-go` | Integration test containers |
| `github.com/cenkalti/backoff/v4` | Exponential backoff retry |
| `github.com/go-redsync/redsync/v4` | Distributed mutex (Redlock) |
| `github.com/redis/go-redis/v9` | Redis client |
| `github.com/stretchr/testify` | Test assertions |

---

## Tooling

```bash
task gen               # buf generate Go code + docs from local api/proto/
task gen:check         # regenerate and fail if pkg/ or docs/proto/ drift from committed code (CI gate)
task proto:fmt         # format proto files (buf format)
task proto:fmt:check   # verify proto files are formatted (CI gate)
task proto:lint        # lint proto files (buf lint)
task proto:breaking    # breaking change detection vs main
task fmt               # format Go code (gofmt + goimports via golangci-lint)
task fmt:check         # verify Go code is formatted (CI gate, dry-run)
task lint              # golangci-lint
task test:integration  # integration tests via testcontainers (requires Docker)
```

---

## Configuration

Config is a YAML file passed via `-config <path>` (default: `config.yaml`).
`${VAR}` and `$VAR` references are expanded from the process environment before parsing.
See `configs/config.example.yaml` for the full annotated reference.

---

## Dependency injection pattern

1. `di.NewContainer(cfg)` registers all `do.Provide` calls.
2. `do.Invoke[*fiber.App](injector)` bootstraps the full dependency graph.
3. External deps that need cleanup implement `Shutdown(ctx context.Context) error` — samber/do calls them on `injector.ShutdownWithContext(ctx)`.

When adding a new infrastructure component:
- For simple clients, a thin DI provider in `di/` with a wrapper struct for lifecycle is sufficient (see `di/nats.go`).
- Register a `do.Provide` in `di/container.go`.

---

## HTTP handler pattern

Handlers live in `internal/handler/`. Each handler is a public constructor returning `fiber.Handler`:

```go
func PostFoo(logger *zerolog.Logger, js jetstream.JetStream) fiber.Handler {
    return func(c fiber.Ctx) error { ... }
}
```

Register in `di/fiber.go`. Pass only what the handler needs (logger, JetStream, etc.) — no god objects.

---

## Error handling

Use `samber/oops` for internal errors in handlers and publishers:

```go
return oops.In("component").Code("snake_case_code").With("key", val).Wrap(err)
```

Use `*fiber.Error` for expected client errors in middleware (401, 422).

Fiber `ErrorHandler` in `di/fiber.go` distinguishes between the two: `*fiber.Error` returns the HTTP status, oops errors log with stack trace and return 500.

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

Every Zitadel Actions v2 POST body deserialises into the generic `zitadel.Envelope[T]`.
The `event_payload` field is a JSON object that is directly unmarshaled into `T`:

```go
envelope := new(zitadel.Envelope[zitadel.UserHumanAdded])
c.Bind().Body(envelope)

// Access the typed payload directly:
envelope.EventPayload.FirstName
```

---

## Protobuf

Proto source lives in `api/proto/` with a buf v2 module (`api/proto/buf.yaml`).
`buf.gen.yaml` in the project root uses managed mode to inject `go_package` and generates Go code into `pkg/`.
Generated packages: `pkg/events/v1/`, `pkg/users/v1/`, and `pkg/sessions/v1/`.
Proto packages: `events.v1`, `users.v1`, `sessions.v1` (no prefix).
Proto files contain no `option go_package` — buf managed mode injects it via `go_package_prefix`.
Run `task gen` to regenerate. Run `task proto:lint` to lint.

---

## Event mapping pipeline

1. Zitadel webhook → handler binds `Envelope[T]`
2. Handler calls a thin `mapper.Build*` function that returns a single typed proto message (no splitting, no value remapping)
3. Handler calls `publish.PublishZitadelEvent` → wraps the payload in `zitadel.events.v1.Envelope`, marshals it, and publishes on the mapped NATS subject with a `Nats-Msg-Id` dedup header set via `jetstream.WithMsgID`
4. NATS subject scheme: `zitadel.<aggregate_plural>.v1.<event_tail>` with the Zitadel event-type dots preserved — e.g. `zitadel.users.v1.human.profile.changed`, `zitadel.sessions.v1.user.checked`

Profile changes are **not** split — a single `UserHumanProfileChanged` payload carries all changed fields plus a `google.protobuf.FieldMask updated_fields` listing which fields were actually in the webhook body. Consumers check the mask to know what changed.

Dedup key: `{instance_id}:{aggregate_type}:{aggregate_id}:{sequence}`. Publisher-side retries are idempotent because JetStream dedups by this key within the stream's duplicate window (recommend 24h; the stream itself is provisioned by operators, not by this service).

---

## Integration tests

Tests live in `test/integration/zitadel/` behind `//go:build integration`.
`TestMain` starts a shared stack via testcontainers-go (PostgreSQL + NATS + Zitadel v4.13.1) and an in-process Fiber service.
Zitadel Actions v2 targets fire real webhooks to the service; tests verify payloads via channels and NATS message assertions.

Run: `task test:integration` (requires Docker).

---

## What is not done yet (TODO)

- Handler for `/user/idp`
- Prometheus metrics endpoint
- Request ID middleware
- Unit tests for `Envelope[T]`
- Makefile (`build` / `run` targets)
- TLS termination strategy
