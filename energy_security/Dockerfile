FROM golang:1.23-alpine3.22 AS builder
ARG BUILD_VERSION=dev
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go test ./... \
 && CGO_ENABLED=0 go vet ./... \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${BUILD_VERSION:-dev}" -o /out/energy-security ./cmd/energy-security

FROM ghcr.io/home-assistant/base:3.24
ARG BUILD_VERSION=0.1.2
ARG BUILD_ARCH
LABEL io.hass.version="${BUILD_VERSION}" \
      io.hass.type="app" \
      io.hass.arch="${BUILD_ARCH}" \
      org.opencontainers.image.title="Energy Security Monitor" \
      org.opencontainers.image.description="Local national energy-security assessment for Home Assistant" \
      org.opencontainers.image.source="https://github.com/ArrowSK/ha-energy-security" \
      org.opencontainers.image.licenses="PolyForm-Noncommercial-1.0.0"
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /out/energy-security /usr/local/bin/energy-security
COPY run.sh /run.sh
RUN chmod 0755 /run.sh /usr/local/bin/energy-security \
 && /usr/local/bin/energy-security --self-test
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 CMD curl --fail --silent --show-error http://127.0.0.1:8099/healthz >/dev/null || exit 1
CMD ["/run.sh"]
