# Contributing

Contributions are welcome for noncommercial use under the repository licence.

The most useful contributions are source adapters, parser fixtures, country profiles, scoring tests, documentation corrections and dashboard accessibility improvements.

## Ground rules

1. Keep the normal installation zero-configuration.
2. Do not add a mandatory paid API, project backend, telemetry service or runtime GitHub dependency.
3. Prefer official machine-readable data over HTML scraping. HTML adapters are acceptable only where there is no practical keyless structured source and must fail safely when parsing is ambiguous.
4. Never interpret missing data as zero.
5. Every new scored metric needs a freshness rule, quality value, unit validation and a test explaining its effect on confidence/score.
6. Do not add heavy runtime frameworks. The steady-state RSS release limit is 50 MB.
7. Provider tests must use fixtures/fakes. CI must not make scheduled or live data-provider requests.

## Local checks

From `energy_security/`:

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/energy-security
./energy-security --self-test
```

The Dockerfile repeats tests and self-test during image construction.

## Adding a country

Country profiles are embedded from `energy_security/internal/country/data/countries.json`. Add only capabilities that have a tested provider. A country with electricity-only coverage should say so; do not label it full merely because a country code is recognised.

## Adding a provider

Implement the `provider.Provider` interface and place it in an ordered provider group. A provider must:

- use bounded HTTP timeouts and response-size limits;
- return source and observation timestamps separately;
- set an explicit data-quality value;
- reject ambiguous or structurally invalid responses;
- expose no credential in errors, URLs returned to the dashboard, or cache attributes;
- have a deterministic fixture/unit test for its parsing logic when practical.
