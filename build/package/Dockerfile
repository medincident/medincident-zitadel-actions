# Stage 1: Build
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 app && adduser -u 1000 -G app -D app \
    && mkdir -p /var/log/medincident-zitadel-actions /etc/medincident-zitadel-actions \
    && chown app:app /var/log/medincident-zitadel-actions /etc/medincident-zitadel-actions

COPY --from=builder /build/server /usr/local/bin/server

USER app

EXPOSE 8080

VOLUME ["/var/log/medincident-zitadel-actions", "/etc/medincident-zitadel-actions"]

ENTRYPOINT ["server"]
CMD ["-config", "/etc/medincident-zitadel-actions/config.yaml"]
