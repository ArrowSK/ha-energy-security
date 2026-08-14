# Energy Security Monitor

Energy Security Monitor turns public energy-system data into a cautious, inspectable assessment of national energy security. Home Assistant remains the primary integration, and the same core can now also run as a standalone Docker or Railway web service.

The normal Home Assistant setup is deliberately simple: add the repository, install the app, start it, and open the Ingress dashboard. By default it uses the country configured for Home Assistant's HOME location. A two-letter ISO country code can be supplied as an override. Standalone deployments use an explicit two-letter country code through environment variables because there is no Home Assistant HOME configuration to resolve.

The project has no account, telemetry service, licence server, remote configuration service, MQTT requirement, paid API requirement, or runtime dependency on this repository. Country profiles, provider logic, scoring rules, dashboard assets, a small versioned electricity reference table and fallback behaviour are shipped with the application binary. If this repository becomes unavailable after installation, a running deployment keeps working with its bundled logic and local cache.

> **Release status:** 0.2.0 adds standalone Docker/Railway deployment while preserving the Home Assistant runtime. Both deployment modes compile the same Go command and `internal/` core. Hungary is the full reference profile. Broader European coverage is graded rather than pretending that every country publishes equivalent data.

## What it measures

Depending on country support and source availability, the app can use:

- electricity generation and live load when available;
- a clearly labelled embedded annual-demand reference when fresh generation exists but live load is absent;
- generation mix, including nuclear, solar, wind, hydro and thermal generation;
- cross-border electricity trading where exposed by the selected feed;
- gas storage/system measurements or a clearly labelled lower-frequency national stock proxy;
- emergency oil-stock days-equivalent for EU profiles;
- hydrological conditions where a country adapter exists; Hungary includes Budapest/Paks Danube evidence and water temperature where published;
- seven-day heat, cold, wind and precipitation stress around the configured location;
- generation diversity as a structural resilience signal;
- source freshness, provider failures, fallbacks and confidence.

The provider model is intentionally extensible: reservoir storage, cooling-water constraints, plant outages, interconnector headroom, LNG send-out, pipeline flows, demand forecasts, pumped storage and batteries can be added without replacing the scoring/dashboard contract.

## Scores and confidence

The dashboard exposes three horizons:

- **Current** — operational conditions now and roughly the next 48 hours.
- **7-day outlook** — current conditions with near-term weather stress.
- **Strategic resilience** — slower-moving structural signals such as generation diversity, storage/stocks, oil evidence and hydrology.

The headline combines those horizons and is always accompanied by a separate **confidence** value.

Missing data never becomes zero. Stale data does not contribute indefinitely. If a source fails, the app tries the next configured provider. If no fresh fallback exists, a last-known-good observation is retained only inside its defined freshness window. Missing domains reduce confidence instead of manufacturing a crisis.

When live electricity load is absent but fresh generation is present, the app can use an embedded recent annual-demand reference for the selected country. It converts annual demand into an annual-average MW value, explicitly labels that derivation in the electricity description, sharply reduces confidence and composite weight, and never treats the reference as current or peak demand. Fresh live load always wins.

The scoring model is deterministic. No LLM, news sentiment model or remote scoring service is in the decision path.

## Home Assistant default

Normal Home Assistant installation requires no credentials:

```yaml
country: auto
refresh_minutes: 30
enable_ha_entities: true
enable_weather: true
```

Advanced users can optionally provide a GIE AGSI key to improve supported gas-storage coverage and an ENTSO-E Transparency Platform token for configured electricity fallback use. These are optional enhancements, not prerequisites.

All normal Home Assistant options can also be changed from **Dashboard → menu → Setup**. Secret values are never returned from the app backend to the browser; blank secret fields preserve existing values unless the explicit clear control is selected.

## One core, two deployment wrappers

The Home Assistant app and standalone service do not maintain separate provider or scoring implementations:

- shared binary entry point: `energy_security/cmd/energy-security`;
- shared core: `energy_security/internal/`;
- Home Assistant container wrapper: `energy_security/Dockerfile` plus Supervisor options/Ingress behaviour;
- standalone Docker/Railway wrapper: repository-root `Dockerfile` plus environment configuration.

`energy_security/config.yaml` remains the release-version source of truth. The root standalone Dockerfile reads that Home Assistant manifest during its build and compiles the same version into the standalone binary. CI verifies that the standalone image's `--version` exactly matches the manifest. A provider/scoring/dashboard fix is therefore made once in the shared core and is inherited by both deployment modes.

## Self-healing behaviour

Each domain has an ordered provider chain. Collection and recovery happen locally:

1. query the preferred supported provider;
2. on failure, try the next provider;
3. after repeated failures, temporarily open a local circuit for the failing provider;
4. keep valid last-known-good observations in `/data`;
5. probe the preferred provider again after cooldown;
6. automatically return to it when it recovers.

Cache writes use a temporary file and atomic rename. Historical dashboard points are bounded. No project server coordinates this process and the app never downloads parser code while running. Both container variants expose `/healthz`; the Home Assistant image also keeps its native Docker `HEALTHCHECK` independently of provider health.

## Dashboard and Home Assistant states

The built-in dashboard uses an Android-style sticky bottom navigator for **Overview, Domains, Signals, Trend and Diagnostics**, keeps manual Refresh as a compact title-bar icon, and places configuration under the title-bar menu. In Home Assistant it is served through Ingress. In standalone mode the same embedded dashboard is served directly over HTTP.

**Domains** has a strict hierarchy. Only **Electricity, Gas, Oil reserves, Hydrology and Weather stress** are shown as top-level scored security domains. Supporting evidence is nested underneath the relevant domain rather than being presented as another score. Nuclear and renewable generation therefore belong under Electricity; storage details belong under Gas; reserve measurements belong under Oil; and river/weather observations stay under their respective domains.

Supporting-indicator sections are collapsed by default and expand on click. A section that the user opens stays open during normal dashboard refreshes. For Electricity, generation components show MW plus a percentage share. When live load exists the denominator is current load, matching the Nuclear presentation; if live load is unavailable the UI explicitly switches to percentage of current generation. Cross-border balance shows percentage of current load when possible. Generation diversity remains a normalized structural score, not a generation share.

The older standalone **Generation mix** card is no longer shown because it duplicated the same electricity evidence already available under Electricity supporting indicators.

Diagnostics separates **Sources** from **Measurements**. Measurement rows are grouped into collapsible domain/provider sections, start collapsed, and use larger typography on mobile. A group manually opened by the user remains open through normal refreshes.

The frontend has no external JavaScript, font, analytics or CDN dependency; its assets are embedded in the Go binary.

With `enable_ha_entities: true`, the Home Assistant app also publishes state-machine sensors through Home Assistant's internal REST API, including:

- `sensor.energy_security_score`
- `sensor.energy_security_confidence`
- `sensor.energy_security_status`
- `sensor.energy_security_electricity`
- `sensor.energy_security_gas`
- `sensor.energy_security_oil`
- `sensor.energy_security_water`
- `sensor.energy_security_weather`
- selected generation/load/nuclear/renewable/storage measurements when available;
- `sensor.energy_security_gas_national_stock` and `sensor.energy_security_gas_stock_index` when the Eurostat monthly gas fallback is active.

Standalone mode deliberately disables Home Assistant entity publication. The embedded electricity reference is scoring metadata and does not fabricate a live `electricity_load_mw` state.

No MQTT broker or companion cloud service is required.

## Current source strategy

The source order is: established machine-readable feeds first, optional official APIs second, public national pages where no stable unauthenticated structured feed exists, then explicitly labelled local fallbacks/cache where their semantics remain defensible.

| Domain | Primary / fallback sources | Credentials | Current coverage |
|---|---|---:|---|
| Electricity | Energy-Charts → optional ENTSO-E → embedded annual-demand reference when only live load is missing | None / optional | Broad Europe |
| Gas | FGSZ (HU) → optional GIE AGSI → Eurostat monthly gas stocks | None / optional | HU full chain; EU strategic fallback |
| Oil | Eurostat emergency-stock dataset | None | EU profiles |
| Water | HYDROINFO | None | Hungary |
| Weather stress | Open-Meteo | None | Configured coordinates |

The gas fallback deliberately distinguishes actual storage fill from the keyless Eurostat monthly stock proxy. The electricity fallback similarly distinguishes a derived annual-average load from live demand. Neither lower-fidelity fallback is allowed to masquerade as the primary measurement.

See [Data sources](docs/DATA_SOURCES.md), [Electricity reference](docs/ELECTRICITY_REFERENCE.md) and [Support matrix](docs/SUPPORT_MATRIX.md) for exact limitations.

## Home Assistant installation

Add this repository to Home Assistant's App Store repositories, then install **Energy Security Monitor**.

The Home Assistant app intentionally omits a runtime `image` setting. Supervisor builds the app locally during installation from the repository rather than requiring a separately managed project container registry. CI builds supported architectures independently for validation. Once installed, normal operation does not contact this repository.

The Home Assistant app opens through Ingress and exposes no user-facing external port.

## Docker and Railway

A standalone deployment is also available without changing the Home Assistant package.

For Docker Compose:

```sh
cp .env.example .env
# edit .env, then:
docker compose up -d --build
```

For Railway, create a service from this repository's `main` branch and set at least `ENERGY_SECURITY_COUNTRY` to a two-letter country code. The included `railway.toml` selects the root Dockerfile, configures `/healthz`, and uses an on-failure restart policy. Railway's injected `PORT` is used automatically.

Standalone Setup values are environment-controlled and therefore read-only from the dashboard; change Docker/Railway variables and restart/redeploy. A `/data` volume is recommended if cached observations and dashboard history should survive container replacement.

See the full [Standalone Docker and Railway guide](docs/STANDALONE.md).

## Resource budget

The runtime is one Go process plus a small container base. There is no Python interpreter, Node process, browser engine, database server or server-side chart renderer.

For the Home Assistant build, the release gate is **under 50 MB steady-state RSS**. The engineering target is **35 MB or less** during normal operation. Build-time memory is separate. See [Memory budget](docs/MEMORY_BUDGET.md) for the measurement method.

## Privacy and independence

At runtime the application sends requests only to data providers needed for the configured country. Home Assistant mode additionally uses Home Assistant's internal API when state publication is enabled and Home Assistant Supervisor when the user explicitly saves dashboard Setup. Standalone mode does neither.

It does not send usage data to ArrowSK, GitHub or any project-operated service. There is no telemetry endpoint, analytics SDK, remote configuration backend, project API, licence server, GitHub polling, or automatic code/parser download.

Standalone deployments do not add an authentication layer. If the dashboard should not be public, keep it on a private network or place it behind an authenticated reverse proxy. Secrets belong in environment variables and are never returned by the configuration API.

## Documentation

- [Home Assistant app guide](energy_security/DOCS.md)
- [Standalone Docker and Railway](docs/STANDALONE.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Scoring model](docs/SCORING.md)
- [Data sources](docs/DATA_SOURCES.md)
- [Embedded electricity reference](docs/ELECTRICITY_REFERENCE.md)
- [Support matrix](docs/SUPPORT_MATRIX.md)
- [Provider health and fallbacks](docs/PROVIDER_HEALTH.md)
- [Privacy](docs/PRIVACY.md)
- [Memory budget](docs/MEMORY_BUDGET.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Security policy](SECURITY.md)
- [Contributing](CONTRIBUTING.md)

## Licence

Copyright 2026 ArrowSK.

The software is licensed under **PolyForm Noncommercial License 1.0.0**. This is a source-available noncommercial licence and is not an OSI-approved open-source licence. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

Most runtime provider data remains subject to the respective provider's terms. The embedded electricity reference is derived from Ember yearly demand data under CC BY 4.0 and is attributed separately in [Third-party software and data](THIRD_PARTY_LICENSES.md).
