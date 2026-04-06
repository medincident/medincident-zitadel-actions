package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	// signatureHeader is the HTTP header Zitadel uses for webhook signatures.
	signatureHeader = "ZITADEL-Signature"

	// defaultTolerance is the maximum allowed age of a signed request (replay protection).
	defaultTolerance = 5 * time.Minute
)

// HMACVerify returns a Fiber middleware that validates the Zitadel webhook
// HMAC-SHA256 signature. If signingKey is empty, the middleware is a no-op
// (useful for development).
func HMACVerify(signingKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if signingKey == "" {
			return c.Next()
		}

		header := c.Get(signatureHeader)
		if header == "" {
			return fiber.ErrUnauthorized
		}

		ts, signatures, err := parseSignatureHeader(header)
		if err != nil || len(signatures) == 0 {
			return fiber.ErrUnauthorized
		}

		// Replay protection.
		age := time.Since(time.Unix(ts, 0))
		if age < 0 {
			age = -age
		}
		if age > defaultTolerance {
			return fiber.ErrUnauthorized
		}

		// Compute expected signature.
		body := c.Body()
		mac := hmac.New(sha256.New, []byte(signingKey))
		fmt.Fprintf(mac, "%d", ts)
		mac.Write([]byte("."))
		mac.Write(body)
		expected := mac.Sum(nil)

		// Constant-time comparison against all v1 signatures.
		for _, sig := range signatures {
			decoded, err := hex.DecodeString(sig)
			if err != nil {
				continue
			}
			if hmac.Equal(expected, decoded) {
				return c.Next()
			}
		}

		return fiber.ErrUnauthorized
	}
}

// parseSignatureHeader parses "t=123,v1=abc,v1=def" into timestamp and signature list.
func parseSignatureHeader(header string) (ts int64, signatures []string, err error) {
	tsFound := false

	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			parsed, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return 0, nil, err
			}
			if parsed < 0 {
				return 0, nil, fmt.Errorf("timestamp out of range: %d", parsed)
			}
			ts = parsed
			tsFound = true
		case "v1":
			if kv[1] != "" {
				signatures = append(signatures, kv[1])
			}
		}
	}

	if !tsFound {
		return 0, nil, fmt.Errorf("missing timestamp")
	}

	return ts, signatures, nil
}

// ComputeSignatureHeader builds a ZITADEL-Signature header value for testing.
func ComputeSignatureHeader(t time.Time, payload []byte, signingKey string) string {
	mac := hmac.New(sha256.New, []byte(signingKey))
	fmt.Fprintf(mac, "%d", t.Unix())
	mac.Write([]byte("."))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", t.Unix(), sig)
}
