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
	signatureHeader = "ZITADEL-Signature"
)

// HMACVerify validates the Zitadel webhook signature header
// (ZITADEL-Signature: t=<unix>,v1=<hex>,...). The MAC input is
// "<t>.<body>" keyed by signingKey, compared in constant time against
// every v1 signature in the header. Requests older than tolerance (or
// in the future by more than tolerance) are rejected to mitigate
// replay. If signingKey is empty the middleware becomes a no-op, which
// is intended only for local development.
//
// See https://zitadel.com/docs/guides/integrate/actions/testing-request-signature
func HMACVerify(signingKey string, tolerance time.Duration) fiber.Handler {
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

		age := time.Since(time.Unix(ts, 0))
		if age < 0 {
			age = -age
		}
		if age > tolerance {
			return fiber.ErrUnauthorized
		}

		body := c.Body()
		mac := hmac.New(sha256.New, []byte(signingKey))
		fmt.Fprintf(mac, "%d", ts)
		mac.Write([]byte("."))
		mac.Write(body)
		expected := mac.Sum(nil)

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

// parseSignatureHeader parses "t=<unix>,v1=<hex>,v1=<hex>,..." into the
// timestamp and the list of v1 signatures. Unknown keys are ignored so
// future scheme versions do not break the parser.
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

// ComputeSignatureHeader builds a ZITADEL-Signature header value. It is
// only used by tests; production never signs outbound requests.
func ComputeSignatureHeader(t time.Time, payload []byte, signingKey string) string {
	mac := hmac.New(sha256.New, []byte(signingKey))
	fmt.Fprintf(mac, "%d", t.Unix())
	mac.Write([]byte("."))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", t.Unix(), sig)
}
