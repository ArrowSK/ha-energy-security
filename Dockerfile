FROM golang:1.23-alpine3.22 AS builder
WORKDIR /src
COPY energy_security/go.mod ./
COPY energy_security/cmd ./cmd
COPY energy_security/internal ./internal
COPY energy_security/config.yaml ./config.yaml
RUN VERSION="$(sed -n 's/^version: "\([^"]*\)"/\1/p' config.yaml)" \
 && test -n "$VERSION" \
 && CGO_ENABLED=0 go test ./... \
 && CGO_ENABLED=0 go vet ./... \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/energy-security ./cmd/energy-security

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S energy-security \
 && adduser -S -G energy-security energy-security \
 && mkdir -p /data \
 && chown energy-security:energy-security /data
COPY --from=builder /out/energy-security /usr/local/bin/energy-security
USER energy-security
ENV ENERGY_SECURITY_MODE=standalone \
    ENERGY_SECURITY_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-8080}/healthz" || exit 1
ENTRYPOINT ["/usr/local/bin/energy-security"]
