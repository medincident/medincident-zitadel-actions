package middleware_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/medincident/medincident-zitadel-actions/internal/middleware"
)

const testSigningKey = "test-secret-key-for-hmac"

func setupApp(signingKey string) *fiber.App {
	app := fiber.New()
	app.Use(middleware.HMACVerify(signingKey))
	app.Post("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func TestHMACVerify_ValidSignature(t *testing.T) {
	app := setupApp(testSigningKey)

	body := []byte(`{"event_type":"user.human.added"}`)
	sig := middleware.ComputeSignatureHeader(time.Now(), body, testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", sig)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestHMACVerify_MissingHeader(t *testing.T) {
	app := setupApp(testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHMACVerify_InvalidSignature(t *testing.T) {
	app := setupApp(testSigningKey)

	body := []byte(`{"event_type":"user.human.added"}`)
	sig := middleware.ComputeSignatureHeader(time.Now(), body, "wrong-key")

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", sig)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHMACVerify_ExpiredTimestamp(t *testing.T) {
	app := setupApp(testSigningKey)

	body := []byte(`{"event_type":"user.human.added"}`)
	sig := middleware.ComputeSignatureHeader(time.Now().Add(-10*time.Minute), body, testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", sig)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHMACVerify_MalformedHeader(t *testing.T) {
	app := setupApp(testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", "garbage")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHMACVerify_EmptyKeySkips(t *testing.T) {
	app := setupApp("")

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestHMACVerify_TamperedBody(t *testing.T) {
	app := setupApp(testSigningKey)

	originalBody := []byte(`{"event_type":"user.human.added"}`)
	sig := middleware.ComputeSignatureHeader(time.Now(), originalBody, testSigningKey)

	tamperedBody := []byte(`{"event_type":"user.human.deleted"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(string(tamperedBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", sig)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHMACVerify_FutureTimestamp(t *testing.T) {
	app := setupApp(testSigningKey)

	body := []byte(`{}`)
	// 4 minutes in the future — within tolerance
	sig := middleware.ComputeSignatureHeader(time.Now().Add(4*time.Minute), body, testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", sig)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestHMACVerify_FutureTimestampBeyondTolerance(t *testing.T) {
	app := setupApp(testSigningKey)

	body := []byte(`{}`)
	// 10 minutes in the future — beyond tolerance
	sig := middleware.ComputeSignatureHeader(time.Now().Add(10*time.Minute), body, testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ZITADEL-Signature", sig)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// Verify response body does not leak internal details.
func TestHMACVerify_NoInternalLeakOnReject(t *testing.T) {
	app := setupApp(testSigningKey)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	assert.NotContains(t, bodyStr, testSigningKey)
	assert.NotContains(t, bodyStr, "hmac")
	assert.NotContains(t, bodyStr, "signature")
}
