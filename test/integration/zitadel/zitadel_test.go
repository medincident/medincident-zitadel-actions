//go:build integration

package zitadel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-redsync/redsync/v4"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/events/v1"
	sessionsv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/sessions/v1"
	usersv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/users/v1"
	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/handler"
	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/middleware"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// Package-level state shared across all tests.
var (
	zitadelBaseURL string
	pat            string
	servicePort    int

	debugCh              = make(chan []byte, 10)
	userAddedCh          = make(chan []byte, 10)
	profileChangedCh     = make(chan []byte, 10)
	emailChangedCh       = make(chan []byte, 10)
	emailVerifiedCh      = make(chan []byte, 10)
	sessionAddedCh       = make(chan []byte, 10)
	sessionUserCheckedCh = make(chan []byte, 10)

	natsConn  *nats.Conn
	js        jetstream.JetStream
	redisAddr string

	cleanupFuncs []func()
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)

	if err := setup(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
		cancel()
		runCleanup()
		os.Exit(1)
	}
	cancel()

	code := m.Run()
	runCleanup()
	os.Exit(code)
}

func runCleanup() {
	for i := len(cleanupFuncs) - 1; i >= 0; i-- {
		cleanupFuncs[i]()
	}
}

func setup(ctx context.Context) error {
	// 1. Docker network
	dockerNet, err := tcnetwork.New(ctx)
	if err != nil {
		return fmt.Errorf("create docker network: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		_ = dockerNet.Remove(context.Background())
	})

	// 2. PostgreSQL
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "zitadel",
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
			},
			Networks: []string{dockerNet.Name},
			NetworkAliases: map[string][]string{
				dockerNet.Name: {"db"},
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
				wait.ForListeningPort("5432/tcp"),
			).WithDeadline(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start postgres: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		_ = pgContainer.Terminate(context.Background())
	})

	// 3. NATS with JetStream
	natsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2-alpine",
			Cmd:          []string{"-js"},
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor: wait.ForListeningPort("4222/tcp").
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start nats: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		_ = natsContainer.Terminate(context.Background())
	})

	natsMappedPort, err := natsContainer.MappedPort(ctx, "4222/tcp")
	if err != nil {
		return fmt.Errorf("get nats mapped port: %w", err)
	}
	natsHost, err := natsContainer.Host(ctx)
	if err != nil {
		return fmt.Errorf("get nats host: %w", err)
	}
	natsURL := fmt.Sprintf("nats://%s:%s", natsHost, natsMappedPort.Port())

	natsConn, err = nats.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("connect to nats: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		natsConn.Close()
	})

	js, err = jetstream.New(natsConn)
	if err != nil {
		return fmt.Errorf("create jetstream context: %w", err)
	}

	// Create the stream that handlers publish to.
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:       "zitadel",
		Subjects:   []string{"zitadel.>"},
		Duplicates: 24 * time.Hour,
	})
	if err != nil {
		return fmt.Errorf("create jetstream stream: %w", err)
	}

	fmt.Printf("NATS ready at %s (JetStream enabled, stream=zitadel)\n", natsURL)

	// 3b. Redis
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForListeningPort("6379/tcp").
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start redis: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		_ = redisContainer.Terminate(context.Background())
	})

	redisMappedPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return fmt.Errorf("get redis mapped port: %w", err)
	}
	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		return fmt.Errorf("get redis host: %w", err)
	}
	redisAddr = fmt.Sprintf("%s:%s", redisHost, redisMappedPort.Port())
	fmt.Printf("Redis ready at %s\n", redisAddr)

	// 4. Our service (in-process)
	if err := startService(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	// 5. Zitadel
	zitadelContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "ghcr.io/zitadel/zitadel:v4.13.1",
			Cmd: []string{
				"start-from-init",
				"--masterkey", "MasterkeyMustBeExact32CharsXXXXX",
				"--tlsMode", "disabled",
				"--config", "/zitadel-config.yaml",
				"--steps", "/zitadel-steps.yaml",
			},
			ExposedPorts: []string{"8080/tcp"},
			Env: map[string]string{
				"ZITADEL_EXTERNALSECURE": "false",
				"ZITADEL_EXTERNALDOMAIN": "localhost",
			},
			Networks: []string{dockerNet.Name},
			Files: []testcontainers.ContainerFile{
				{
					HostFilePath:      "data/zitadel-config.yaml",
					ContainerFilePath: "/zitadel-config.yaml",
					FileMode:          0o644,
				},
				{
					HostFilePath:      "data/zitadel-steps.yaml",
					ContainerFilePath: "/zitadel-steps.yaml",
					FileMode:          0o644,
				},
			},
			HostAccessPorts: []int{servicePort},
			WaitingFor: wait.ForHTTP("/debug/healthz").
				WithPort("8080/tcp").
				WithStartupTimeout(360 * time.Second).
				WithPollInterval(5 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start zitadel: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		_ = zitadelContainer.Terminate(context.Background())
	})

	// Zitadel base URL
	mappedPort, err := zitadelContainer.MappedPort(ctx, "8080/tcp")
	if err != nil {
		return fmt.Errorf("get zitadel mapped port: %w", err)
	}
	host, err := zitadelContainer.Host(ctx)
	if err != nil {
		return fmt.Errorf("get zitadel host: %w", err)
	}
	zitadelBaseURL = fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	// 6. Extract PAT from Zitadel logs
	pat, err = extractPAT(ctx, zitadelContainer)
	if err != nil {
		return fmt.Errorf("extract PAT: %w", err)
	}
	fmt.Printf("Zitadel ready at %s (PAT length: %d)\n", zitadelBaseURL, len(pat))

	return nil
}

func startService() error {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	rc := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	cleanupFuncs = append(cleanupFuncs, func() {
		_ = rc.Close()
	})
	pool := goredis.NewPool(rc)
	rs := redsync.New(pool)

	cfg := &config.Config{
		Redis: config.RedisConfig{
			LockPrefix: "test:lock:",
			LockExpiry: config.Duration(30 * time.Second),
		},
		Publish: config.PublishConfig{
			MaxRetries:     3,
			InitialBackoff: config.Duration(100 * time.Millisecond),
			MaxBackoff:     config.Duration(1 * time.Second),
			MaxElapsedTime: config.Duration(5 * time.Second),
		},
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BodyLimit:    64 * 1024,
	})

	// Body-capturing middleware — must be registered before routes.
	app.Use(func(c fiber.Ctx) error {
		body := slices.Clone(c.Body())
		err := c.Next()
		if err == nil {
			switch c.Path() {
			case "/debug":
				trySend(debugCh, body)
			case "/events/user/human/added":
				trySend(userAddedCh, body)
			case "/events/user/human/profile/changed":
				trySend(profileChangedCh, body)
			case "/events/user/human/email/changed":
				trySend(emailChangedCh, body)
			case "/events/user/human/email/verified":
				trySend(emailVerifiedCh, body)
			case "/events/session/added":
				trySend(sessionAddedCh, body)
			case "/events/session/user/checked":
				trySend(sessionUserCheckedCh, body)
			}
		}
		return err
	})

	// Health check (no middleware).
	app.Get("/health", handler.HealthCheck(natsConn, rc))

	// POST routes with ContentType middleware, no HMAC (empty key = no-op).
	post := app.Group("", middleware.ContentType(), middleware.HMACVerify("", 5*time.Minute))

	pub := publish.NewPublisher(&logger, js, cfg.Publish)
	eh := handler.NewEventHandler(&logger, pub, rs, cfg)

	post.Post("/debug", handler.PostDebugWebhook(&logger))
	post.Post("/events/user/human/added", eh.PostHumanUserAdded())
	post.Post("/events/user/human/profile/changed", eh.PostHumanUserProfileChanged())
	post.Post("/events/user/human/email/changed", eh.PostHumanUserEmailChanged())
	post.Post("/events/user/human/email/verified", eh.PostHumanUserEmailVerified())
	post.Post("/events/session/added", eh.PostSessionAdded())
	post.Post("/events/session/user/checked", eh.PostSessionUserChecked())

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	servicePort = ln.Addr().(*net.TCPAddr).Port

	go func() {
		_ = app.Listener(ln)
	}()

	cleanupFuncs = append(cleanupFuncs, func() {
		_ = app.Shutdown()
	})

	fmt.Printf("Service listening on 127.0.0.1:%d\n", servicePort)
	return nil
}

func trySend(ch chan<- []byte, data []byte) {
	select {
	case ch <- data:
	default:
	}
}

func extractPAT(ctx context.Context, c testcontainers.Container) (string, error) {
	reader, err := c.Logs(ctx)
	if err != nil {
		return "", fmt.Errorf("get logs: %w", err)
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}

	// Strip docker multiplexing headers (non-printable chars).
	clean := strings.Map(func(r rune) rune {
		if r >= 32 || r == '\n' || r == '\r' {
			return r
		}
		return -1
	}, string(raw))

	lines := strings.Split(clean, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), "serviceaccount") {
			for j := i + 1; j < len(lines); j++ {
				candidate := strings.TrimSpace(lines[j])
				if candidate != "" && len(candidate) > 10 {
					return candidate, nil
				}
			}
		}
	}

	return "", fmt.Errorf("PAT not found in container logs (%d lines)", len(lines))
}

// --- Zitadel API helpers ---

func zitadelAPI(path string, body any) (map[string]any, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, zitadelBaseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, respBody)
	}

	if errCode, ok := result["code"]; ok {
		return result, fmt.Errorf("zitadel API error (code=%v): %s", errCode, respBody)
	}

	return result, nil
}

func createTarget(name, endpoint string) (string, error) {
	resp, err := zitadelAPI("/zitadel.action.v2beta.ActionService/CreateTarget", map[string]any{
		"name":        name,
		"restWebhook": map[string]any{"interruptOnError": false},
		"endpoint":    endpoint,
		"timeout":     "10s",
	})
	if err != nil {
		return "", fmt.Errorf("create target %q: %w", name, err)
	}
	if id, ok := resp["id"].(string); ok && id != "" {
		return id, nil
	}
	if d, ok := resp["details"].(map[string]any); ok {
		if id, ok := d["id"].(string); ok && id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("no target ID in response: %v", resp)
}

func setExecution(condition map[string]any, targetID string) error {
	_, err := zitadelAPI("/zitadel.action.v2beta.ActionService/SetExecution", map[string]any{
		"condition": condition,
		"targets":   []string{targetID},
	})
	return err
}

func createUser(givenName, familyName, email string) (string, error) {
	resp, err := zitadelAPI("/zitadel.user.v2.UserService/AddHumanUser", map[string]any{
		"username": fmt.Sprintf("test-%d", time.Now().UnixNano()),
		"profile": map[string]any{
			"givenName":  givenName,
			"familyName": familyName,
		},
		"email": map[string]any{
			"email":      email,
			"isVerified": true,
		},
		"password": map[string]any{
			"password":       "TestPassword1!",
			"changeRequired": false,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}
	if id, ok := resp["userId"].(string); ok && id != "" {
		return id, nil
	}
	if id, ok := resp["user_id"].(string); ok && id != "" {
		return id, nil
	}
	return "", fmt.Errorf("no user ID in response: %v", resp)
}

func updateProfile(userID, givenName, familyName string) error {
	_, err := zitadelAPI("/zitadel.user.v2.UserService/UpdateHumanUser", map[string]any{
		"userId": userID,
		"profile": map[string]any{
			"givenName":  givenName,
			"familyName": familyName,
		},
	})
	return err
}

func updateEmail(userID, newEmail string) (string, error) {
	resp, err := zitadelAPI("/zitadel.user.v2.UserService/UpdateHumanUser", map[string]any{
		"userId": userID,
		"email": map[string]any{
			"email":      newEmail,
			"returnCode": map[string]any{},
		},
	})
	if err != nil {
		return "", fmt.Errorf("update email: %w", err)
	}

	// Zitadel v4.13.1 returns the code as "emailCode" when returnCode is set.
	if vc, ok := resp["emailCode"].(string); ok && vc != "" {
		return vc, nil
	}
	if vc, ok := resp["verificationCode"].(string); ok && vc != "" {
		return vc, nil
	}

	return "", fmt.Errorf("no verification code in response: %v", resp)
}

func verifyEmail(userID, code string) error {
	_, err := zitadelAPI("/zitadel.user.v2.UserService/VerifyEmail", map[string]any{
		"userId":           userID,
		"verificationCode": code,
	})
	return err
}

func createSession(userID string) error {
	resp, err := zitadelAPI("/v2/sessions", map[string]any{
		"checks": map[string]any{
			"user": map[string]any{
				"userId": userID,
			},
		},
		"userAgent": map[string]any{
			"fingerprintId": "test-fingerprint",
			"ip":            "127.0.0.1",
			"description":   "integration-test-agent",
		},
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if _, ok := resp["sessionId"].(string); ok {
		return nil
	}
	return fmt.Errorf("no sessionId in response: %v", resp)
}

func waitForBody(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	select {
	case body := <-ch:
		return body
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for webhook")
		return nil
	}
}

func drainChannel(ch chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

const natsFetchTimeout = 30 * time.Second

// fetchNATSMessage creates an ephemeral consumer on the given subject, fetches one message,
// and unmarshals it into an eventsv1.Envelope.
func fetchNATSMessage(t *testing.T, subject string) *eventsv1.Envelope {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), natsFetchTimeout)
	defer cancel()

	cons, err := js.CreateConsumer(ctx, "zitadel", jetstream.ConsumerConfig{
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverLastPerSubjectPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	require.NoError(t, err, "create NATS consumer for %s", subject)

	msgs, err := cons.Fetch(1, jetstream.FetchMaxWait(natsFetchTimeout))
	require.NoError(t, err, "fetch NATS message from %s", subject)

	var envelope *eventsv1.Envelope
	for msg := range msgs.Messages() {
		envelope = new(eventsv1.Envelope)
		require.NoError(t, proto.Unmarshal(msg.Data(), envelope), "unmarshal NATS envelope")
		_ = msg.Ack()
	}

	require.NoError(t, msgs.Error(), "NATS fetch error for %s", subject)
	require.NotNil(t, envelope, "no NATS message received on %s within %s", subject, natsFetchTimeout)

	return envelope
}

// --- Test Cases ---

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", servicePort), http.NoBody)
	require.NoError(t, err, "build GET /health request")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "GET /health")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "ok", result["status"])
}

func TestUserHumanAdded(t *testing.T) {
	drainChannel(userAddedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/user/human/added", servicePort)

	tid, err := createTarget("added-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.added"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	_, err = createUser("Test", "User", email)
	require.NoError(t, err, "create user")

	// Verify webhook received.
	body := waitForBody(t, userAddedCh)

	var envelope zitadel.Envelope[zitadel.UserHumanAdded]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "Test", envelope.EventPayload.FirstName)
	assert.Equal(t, "User", envelope.EventPayload.LastName)
	assert.Equal(t, email, envelope.EventPayload.Email)
	assert.Equal(t, "user.human.added", envelope.EventType)

	// Verify NATS message published.
	natsEnvelope := fetchNATSMessage(t, "zitadel.users.v1.human.added")

	assert.Equal(t, "user", natsEnvelope.GetAggregateType())
	assert.NotNil(t, natsEnvelope.GetCreatedAt())
	assert.NotEmpty(t, natsEnvelope.GetAggregateId())
	assert.Equal(t, "user.human.added", natsEnvelope.GetEventType())

	// Unpack the payload and verify UserHumanAdded fields.
	var payload usersv1.UserHumanAdded
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&payload), "unmarshal UserHumanAdded from NATS payload")
	assert.Equal(t, "Test", payload.GetFirstName())
	assert.Equal(t, "User", payload.GetLastName())
	assert.Equal(t, email, payload.GetEmail())
}

func TestUserHumanProfileChanged(t *testing.T) {
	drainChannel(profileChangedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/user/human/profile/changed", servicePort)

	tid, err := createTarget("profile-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.profile.changed"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	userID, err := createUser("Original", "Name", email)
	require.NoError(t, err, "create user")

	time.Sleep(time.Second)

	err = updateProfile(userID, "Updated", "Profile")
	require.NoError(t, err, "update profile")

	// Verify webhook received.
	body := waitForBody(t, profileChangedCh)

	var envelope zitadel.Envelope[zitadel.UserHumanProfileChanged]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "Updated", envelope.EventPayload.FirstName)
	assert.Equal(t, "Profile", envelope.EventPayload.LastName)

	// Verify NATS message published.
	natsEnvelope := fetchNATSMessage(t, "zitadel.users.v1.human.profile.changed")

	assert.Equal(t, "user", natsEnvelope.GetAggregateType())
	assert.NotNil(t, natsEnvelope.GetCreatedAt())
	assert.NotEmpty(t, natsEnvelope.GetAggregateId())
	assert.Equal(t, "user.human.profile.changed", natsEnvelope.GetEventType())

	// Unpack the payload and verify UserHumanProfileChanged fields.
	var profileChanged usersv1.UserHumanProfileChanged
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&profileChanged), "unmarshal UserHumanProfileChanged from NATS payload")
	assert.Equal(t, "Updated", profileChanged.GetFirstName())
	assert.Equal(t, "Profile", profileChanged.GetLastName())
}

func TestProfileChangedFieldMaskOnlyNickName(t *testing.T) {
	drainChannel(profileChangedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/user/human/profile/changed", servicePort)

	tid, err := createTarget("nick-only-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.profile.changed"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	userID, err := createUser("Nick", "Test", email)
	require.NoError(t, err, "create user")

	time.Sleep(time.Second)

	// Zitadel's UpdateHumanUser API validates the embedded profile as a
	// whole — passing only nickName is rejected for missing givenName.
	// We pass first/last unchanged so Zitadel's diff engine emits a
	// profile.changed event whose payload contains only the nickName
	// delta (and therefore whose FieldMask covers only nick_name).
	_, err = zitadelAPI("/zitadel.user.v2.UserService/UpdateHumanUser", map[string]any{
		"userId": userID,
		"profile": map[string]any{
			"givenName":  "Nick",
			"familyName": "Test",
			"nickName":   "only-nick",
		},
	})
	require.NoError(t, err, "update nick name")

	// Wait for webhook.
	waitForBody(t, profileChangedCh)

	// Fetch NATS message and verify FieldMask.
	natsEnvelope := fetchNATSMessage(t, "zitadel.users.v1.human.profile.changed")

	var profileChanged usersv1.UserHumanProfileChanged
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&profileChanged), "unmarshal UserHumanProfileChanged from NATS payload")

	paths := profileChanged.GetUpdatedFields().GetPaths()
	assert.Equal(t, []string{"nick_name"}, paths, "FieldMask should contain only nick_name")
	assert.Equal(t, "only-nick", profileChanged.GetNickName())
}

func TestUserHumanEmailChanged(t *testing.T) {
	drainChannel(emailChangedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/user/human/email/changed", servicePort)

	tid, err := createTarget("email-changed-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.email.changed"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	userID, err := createUser("Email", "Test", email)
	require.NoError(t, err, "create user")

	time.Sleep(time.Second)

	newEmail := fmt.Sprintf("changed-%s@example.com", suffix)
	_, err = updateEmail(userID, newEmail)
	require.NoError(t, err, "update email")

	// Verify webhook received.
	body := waitForBody(t, emailChangedCh)

	var envelope zitadel.Envelope[zitadel.UserHumanEmailChanged]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, newEmail, envelope.EventPayload.Email)
	assert.Equal(t, "user.human.email.changed", envelope.EventType)

	// Verify NATS message published.
	natsEnvelope := fetchNATSMessage(t, "zitadel.users.v1.human.email.changed")

	assert.Equal(t, "user", natsEnvelope.GetAggregateType())
	assert.NotNil(t, natsEnvelope.GetCreatedAt())
	assert.NotEmpty(t, natsEnvelope.GetAggregateId())
	assert.Equal(t, "user.human.email.changed", natsEnvelope.GetEventType())

	// Unpack the payload and verify UserHumanEmailChanged fields.
	var emailChanged usersv1.UserHumanEmailChanged
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&emailChanged), "unmarshal UserHumanEmailChanged from NATS payload")
	assert.Equal(t, newEmail, emailChanged.GetEmail())
}

func TestUserHumanEmailVerified(t *testing.T) {
	drainChannel(emailVerifiedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/user/human/email/verified", servicePort)

	tid, err := createTarget("email-verified-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.email.verified"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	userID, err := createUser("Verify", "Test", email)
	require.NoError(t, err, "create user")

	time.Sleep(time.Second)

	// Change email to make it unverified, capturing the verification code.
	newEmail := fmt.Sprintf("verify-%s@example.com", suffix)
	code, err := updateEmail(userID, newEmail)
	require.NoError(t, err, "update email")
	require.NotEmpty(t, code, "verification code should be returned")

	time.Sleep(time.Second)

	// Verify the email — this triggers user.human.email.verified.
	err = verifyEmail(userID, code)
	require.NoError(t, err, "verify email")

	// Verify webhook received.
	body := waitForBody(t, emailVerifiedCh)

	var envelope zitadel.Envelope[zitadel.UserHumanEmailVerified]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "user.human.email.verified", envelope.EventType)

	// Verify NATS message published.
	natsEnvelope := fetchNATSMessage(t, "zitadel.users.v1.human.email.verified")

	assert.Equal(t, "user", natsEnvelope.GetAggregateType())
	assert.NotNil(t, natsEnvelope.GetCreatedAt())
	assert.NotEmpty(t, natsEnvelope.GetAggregateId())
	assert.Equal(t, "user.human.email.verified", natsEnvelope.GetEventType())

	// Unpack the payload — UserHumanEmailVerified has no fields, just verify it deserializes.
	var emailVerified usersv1.UserHumanEmailVerified
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&emailVerified), "unmarshal UserHumanEmailVerified from NATS payload")
}

func TestSessionAdded(t *testing.T) {
	drainChannel(sessionAddedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/session/added", servicePort)

	tid, err := createTarget("session-added-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "session.added"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	userID, err := createUser("Session", "Test", email)
	require.NoError(t, err, "create user")

	time.Sleep(time.Second)

	err = createSession(userID)
	require.NoError(t, err, "create session")

	// Verify webhook received.
	body := waitForBody(t, sessionAddedCh)

	var envelope zitadel.Envelope[zitadel.SessionAdded]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "session.added", envelope.EventType)
	assert.Equal(t, "session", envelope.AggregateType)

	// Verify NATS message published.
	natsEnvelope := fetchNATSMessage(t, "zitadel.sessions.v1.added")

	assert.Equal(t, "session", natsEnvelope.GetAggregateType())
	assert.NotNil(t, natsEnvelope.GetCreatedAt())
	assert.NotEmpty(t, natsEnvelope.GetAggregateId())
	assert.Equal(t, "session.added", natsEnvelope.GetEventType())

	var sessionAdded sessionsv1.SessionAdded
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&sessionAdded), "unmarshal SessionAdded from NATS payload")
	ua := sessionAdded.GetUserAgent()
	require.NotNil(t, ua, "UserAgent should be present")
	assert.Equal(t, "test-fingerprint", ua.GetFingerprintId())
	assert.Equal(t, "127.0.0.1", ua.GetIp())
	assert.Equal(t, "integration-test-agent", ua.GetDescription())
}

func TestSessionUserChecked(t *testing.T) {
	drainChannel(sessionUserCheckedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/events/session/user/checked", servicePort)

	tid, err := createTarget("session-user-checked-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "session.user.checked"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	userID, err := createUser("SessionUser", "Test", email)
	require.NoError(t, err, "create user")

	time.Sleep(time.Second)

	err = createSession(userID)
	require.NoError(t, err, "create session")

	// Verify webhook received.
	body := waitForBody(t, sessionUserCheckedCh)

	var envelope zitadel.Envelope[zitadel.SessionUserChecked]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "session.user.checked", envelope.EventType)
	assert.Equal(t, userID, envelope.EventPayload.UserID)

	// Verify NATS message published.
	natsEnvelope := fetchNATSMessage(t, "zitadel.sessions.v1.user.checked")

	assert.Equal(t, "session", natsEnvelope.GetAggregateType())
	assert.NotNil(t, natsEnvelope.GetCreatedAt())
	assert.NotEmpty(t, natsEnvelope.GetAggregateId())
	assert.Equal(t, "session.user.checked", natsEnvelope.GetEventType())

	var userChecked sessionsv1.SessionUserChecked
	require.NoError(t, natsEnvelope.GetPayload().UnmarshalTo(&userChecked), "unmarshal SessionUserChecked from NATS payload")
	assert.Equal(t, userID, userChecked.GetUserId())
}

func TestDebugWebhook(t *testing.T) {
	drainChannel(debugCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/debug", servicePort)

	tid, err := createTarget("debug-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.added"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	_, err = createUser("Debug", "Event", email)
	require.NoError(t, err, "create user")

	body := waitForBody(t, debugCh)

	assert.True(t, json.Valid(body), "body should be valid JSON")
	var raw map[string]any
	require.NoError(t, json.Unmarshal(body, &raw))
	assert.Contains(t, raw, "event_type")
	assert.Equal(t, "user.human.added", raw["event_type"])
}

func TestDedupByMsgID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cfg := config.PublishConfig{
		MaxRetries:     1,
		InitialBackoff: config.Duration(50 * time.Millisecond),
		MaxBackoff:     config.Duration(200 * time.Millisecond),
		MaxElapsedTime: config.Duration(1 * time.Second),
	}
	pub := publish.NewPublisher(&logger, js, cfg)

	// Use a subject no other test publishes to so the fetch count
	// reflects only this test's two publishes.
	subject := "zitadel.users.v1.dedup.test"
	env := zitadel.Envelope[zitadel.UserHumanEmailChanged]{
		AggregateID:   "dedup-agg",
		AggregateType: "user",
		ResourceOwner: "org-1",
		InstanceID:    "inst-1",
		Version:       "v1",
		Sequence:      999,
		EventType:     "user.human.email.changed",
		CreatedAt:     time.Now().UTC(),
		UserID:        "user-1",
		EventPayload:  zitadel.UserHumanEmailChanged{Email: "dedup@example.com"},
	}

	payload, err := mapper.BuildUserHumanEmailChanged(&env)
	require.NoError(t, err)

	meta := publish.FromZitadelEnvelope(&env)

	require.NoError(t, pub.PublishZitadelEvent(ctx, subject, meta, payload), "first publish")
	require.NoError(t, pub.PublishZitadelEvent(ctx, subject, meta, payload), "second publish")

	cons, err := js.CreateConsumer(ctx, "zitadel", jetstream.ConsumerConfig{
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	require.NoError(t, err)

	msgs, err := cons.Fetch(2, jetstream.FetchMaxWait(2*time.Second))
	require.NoError(t, err)

	count := 0
	for m := range msgs.Messages() {
		count++
		_ = m.Ack()
	}
	require.NoError(t, msgs.Error())
	assert.Equal(t, 1, count, "JetStream should keep only one message for duplicate Nats-Msg-Id")
}
