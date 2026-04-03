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

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/medincident/medincident-zitadel-actions/internal/httpserver/events"
	"github.com/medincident/medincident-zitadel-actions/internal/httpserver/requests"
	"github.com/medincident/medincident-zitadel-actions/internal/httpserver/responses"
	zitadelevents "github.com/medincident/medincident-zitadel-actions/internal/zitadel/actions/events"
)

// Package-level state shared across all tests.
var (
	zitadelBaseURL string
	pat            string
	servicePort    int

	eventCh          = make(chan []byte, 10)
	requestCh        = make(chan []byte, 10)
	responseCh       = make(chan []byte, 10)
	userAddedCh      = make(chan []byte, 10)
	profileChangedCh = make(chan []byte, 10)

	cleanupFuncs []func()
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()

	if err := setup(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
		runCleanup()
		os.Exit(1)
	}

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

	// 3. Our service (in-process)
	if err := startService(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	// 4. Zitadel
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

	// 5. Extract PAT from Zitadel logs
	pat, err = extractPAT(ctx, zitadelContainer)
	if err != nil {
		return fmt.Errorf("extract PAT: %w", err)
	}
	fmt.Printf("Zitadel ready at %s (PAT length: %d)\n", zitadelBaseURL, len(pat))

	return nil
}

func startService() error {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

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
			case "/event":
				trySend(eventCh, body)
			case "/request":
				trySend(requestCh, body)
			case "/response":
				trySend(responseCh, body)
			case "/user/human/added":
				trySend(userAddedCh, body)
			case "/user/human/profile/changed":
				trySend(profileChangedCh, body)
			}
		}
		return err
	})

	// Health check.
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Register real production handlers.
	events.SetupRoutes(app, &logger)
	requests.SetupRoutes(app, &logger)
	responses.SetupRoutes(app, &logger)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
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

func zitadelAPI(method, path string, body any) (map[string]any, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, zitadelBaseURL+path, r)
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
	resp, err := zitadelAPI("POST", "/zitadel.action.v2beta.ActionService/CreateTarget", map[string]any{
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
	_, err := zitadelAPI("POST", "/zitadel.action.v2beta.ActionService/SetExecution", map[string]any{
		"condition": condition,
		"targets":   []string{targetID},
	})
	return err
}

func createUser(givenName, familyName, email string) (string, error) {
	resp, err := zitadelAPI("POST", "/zitadel.user.v2.UserService/AddHumanUser", map[string]any{
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
	_, err := zitadelAPI("POST", "/zitadel.user.v2.UserService/UpdateHumanUser", map[string]any{
		"userId": userID,
		"profile": map[string]any{
			"givenName":  givenName,
			"familyName": familyName,
		},
	})
	return err
}

func waitForBody(t *testing.T, ch <-chan []byte, timeout time.Duration) []byte {
	t.Helper()
	select {
	case body := <-ch:
		return body
	case <-time.After(timeout):
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

// --- Test Cases ---

func TestUserHumanAdded(t *testing.T) {
	drainChannel(userAddedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/user/human/added", servicePort)

	tid, err := createTarget("added-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"event": "user.human.added"}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	_, err = createUser("Test", "User", email)
	require.NoError(t, err, "create user")

	body := waitForBody(t, userAddedCh, 30*time.Second)

	var envelope zitadelevents.Envelope[zitadelevents.UserHumanAdded]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "Test", envelope.EventPayload.FirstName)
	assert.Equal(t, "User", envelope.EventPayload.LastName)
	assert.Equal(t, email, envelope.EventPayload.Email)
	assert.Equal(t, "user.human.added", envelope.EventType)
}

func TestUserHumanProfileChanged(t *testing.T) {
	drainChannel(profileChangedCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/user/human/profile/changed", servicePort)

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

	body := waitForBody(t, profileChangedCh, 30*time.Second)

	var envelope zitadelevents.Envelope[zitadelevents.UserHumanProfileChanged]
	require.NoError(t, json.Unmarshal(body, &envelope), "unmarshal envelope")

	assert.Equal(t, "Updated", envelope.EventPayload.FirstName)
	assert.Equal(t, "Profile", envelope.EventPayload.LastName)
}

func TestCatchAllEvent(t *testing.T) {
	drainChannel(eventCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/event", servicePort)

	tid, err := createTarget("event-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"event": map[string]any{"all": true}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	_, err = createUser("CatchAll", "Event", email)
	require.NoError(t, err, "create user")

	body := waitForBody(t, eventCh, 30*time.Second)

	assert.True(t, json.Valid(body), "body should be valid JSON")
	var raw map[string]any
	require.NoError(t, json.Unmarshal(body, &raw))
	assert.Contains(t, raw, "event_type")
}

func TestCatchAllRequest(t *testing.T) {
	drainChannel(requestCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/request", servicePort)

	tid, err := createTarget("request-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"request": map[string]any{"all": true}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	_, err = createUser("CatchAll", "Request", email)
	require.NoError(t, err, "create user")

	body := waitForBody(t, requestCh, 30*time.Second)

	assert.True(t, json.Valid(body), "body should be valid JSON")
}

func TestCatchAllResponse(t *testing.T) {
	drainChannel(responseCh)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	endpoint := fmt.Sprintf("http://host.testcontainers.internal:%d/response", servicePort)

	tid, err := createTarget("response-"+suffix, endpoint)
	require.NoError(t, err, "create target")

	err = setExecution(map[string]any{"response": map[string]any{"all": true}}, tid)
	require.NoError(t, err, "set execution")

	time.Sleep(3 * time.Second)

	email := fmt.Sprintf("test-%s@example.com", suffix)
	_, err = createUser("CatchAll", "Response", email)
	require.NoError(t, err, "create user")

	body := waitForBody(t, responseCh, 30*time.Second)

	assert.True(t, json.Valid(body), "body should be valid JSON")
}
