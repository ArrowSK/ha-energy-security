# Energy Security Monitor for Home Assistant

Energy Security Monitor is a self-contained Home Assistant app that turns public energy-system data into a cautious, inspectable assessment of national energy security.

The normal setup is deliberately simple: add the repository, install the app, start it, and open the Ingress dashboard. By default it uses the country configured for Home Assistant's HOME location. A two-letter ISO country code can be supplied as an override.

The app has no project account, telemetry service, licence server, remote configuration service, MQTT requirement, paid API requirement, or runtime dependency on this repository. Country profiles, provider logic, scoring rules, dashboard assets and fallback behaviour are shipped with the installed app. If this repository becomes unavailable after installation, the running app keeps working with its bundled logic and local cache.

> **Release status:** 0.1.0 is an experimental first public release. Hungary is the full reference profile. Broader European electricity and oil coverage is available, but country support is graded rather than pretending that every country publishes equivalent data.

## What it measures

The first release covers the parts of energy security that can be obtained reliably without asking users to connect a collection of external services:

- electricity generation and load;
- generation mix, including nuclear, solar, wind, hydro and thermal generation where available;
- cross-border electricity trading where exposed by the selected feed;
- gas storage and gas-system measurements where a country adapter exists;
- emergency oil-stock data from Eurostat for EU profiles, interpreted conservatively;
- hydrological conditions where a country adapter exists; Hungary includes Budapest and Paks Danube level, discharge and water temperature when published;
- seven-day heat, cold, wind and precipitation stress around HOME;
- generation diversity as a structural resilience signal;
- source freshness, provider failures, fallbacks and confidence.

The provider model can later add reservoir storage, cooling-water constraints, plant outages, interconnector headroom, LNG send-out, pipeline flows, demand forecasts, pumped storage and batteries without changing the dashboard or scoring contract.

## Scores and confidence

The dashboard exposes three horizons:

- **Current** — operational conditions now and roughly the next 48 hours.
- **7-day outlook** — current conditions with near-term weather stress.
- **Strategic resilience** — slower-moving structural signals such as generation diversity, storage, oil data and hydrology.

The headline combines those horizons, but is always accompanied by a separate **confidence** value.

Missing data never becomes zero. Stale data does not contribute indefinitely. If a source fails, the app tries the next configured provider. If no fresh fallback exists, a last-known-good observation is retained only inside its defined freshness window. Missing domains reduce confidence instead of manufacturing a crisis.

The scoring model is deterministic. No LLM, news sentiment model or remote scoring service is in the decision path.

## Zero-configuration default

Normal installation requires no credentials:

```yaml
country: auto
refresh_minutes: 30
enable_ha_entities: true
enable_weather: true
```

Advanced users can optionally provide a GIE AGSI key to improve supported gas-storage coverage and an ENTSO-E Transparency Platform token for configured electricity fallback use. These are optional enhancements, not prerequisites.

## Self-healing behaviour

Each domain has an ordered provider chain. The collector records provider health locally and applies bounded recovery behaviour:

1. query the preferred supported provider;
2. on failure, try the next provider;
3. after repeated failures, temporarily open a circuit for the failing provider;
4. keep valid last-known-good observations in `/data`;
5. probe the preferred provider again after its cooldown;
6. automatically return to it when it recovers.

Cache writes use a temporary file and atomic rename. Historical dashboard points are bounded to seven days. No project server coordinates this process.

## Dashboard and Home Assistant states

The built-in dashboard is served through Home Assistant Ingress and is progressive rather than a wall of gauges. Its main view shows the national headline, confidence, the three scoring horizons, domain scores, current stress signals, electricity generation mix and recent history. **Diagnostics** exposes provider state, source age, measurement quality, fallback errors and stale state.

The frontend has no external JavaScript, font, analytics or CDN dependency; its assets are embedded in the Go binary.

With `enable_ha_entities: true`, the app also publishes a compact set of state-machine sensors through Home Assistant's internal REST API, including:

- `sensor.energy_security_score`
- `sensor.energy_security_confidence`
- `sensor.energy_security_status`
- `sensor.energy_security_electricity`
- `sensor.energy_security_gas`
- `sensor.energy_security_oil`
- `sensor.energy_security_water`
- `sensor.energy_security_weather`
- selected generation, load, nuclear, renewable and storage measurements when available.

No MQTT broker or companion cloud service is required.

## Current source strategy

The source order is: established machine-readable feeds first, optional official APIs second, public national pages where no stable unauthenticated feed exists, then local last-known-good state.

| Domain | Primary / optional source | Credentials | 0.1.0 coverage |
|---|---|---:|---|
| Electricity | Energy-Charts / optional ENTSO-E fallback | None / optional | Broad Europe |
| Gas | FGSZ Hungary / optional GIE AGSI | None / optional | Hungary full; AGSI enhancement |
| Oil | Eurostat emergency-stock dataset | None | EU profiles, conservative parsing |
| Water | HYDROINFO | None | Hungary |
| Weather stress | Open-Meteo | None | HOME coordinates |

See [Data sources](docs/DATA_SOURCES.md) and [Support matrix](docs/SUPPORT_MATRIX.md) for exact limitations.

## Installation

Add this repository to Home Assistant's App Store repositories, then install **Energy Security Monitor**.

Version 0.1.0 intentionally omits the Home Assistant `image` setting. Supervisor therefore builds the app locally from the public repository during installation rather than requiring access to a separately managed container-registry package. CI independently builds both `aarch64` and `amd64` container variants to validate the Dockerfile and architecture support, but does not publish a runtime package. After installation, normal operation does not contact this repository.

The app opens through Home Assistant Ingress and exposes no external port.

## Resource budget

The runtime is one statically linked Go process plus the Home Assistant base image. There is no Python interpreter, Node process, browser engine, database server or server-side chart renderer.

The release gate is **under 50 MB steady-state RSS**. The engineering target is **35 MB or less** during normal operation. Build-time memory is separate from this runtime budget. See [Memory budget](docs/MEMORY_BUDGET.md) for the measurement method.

## Privacy and independence

At runtime the app sends requests only to data providers needed for the configured country and, when enabled, Home Assistant's internal API. It does not send usage data to ArrowSK, GitHub or any project-operated service.

There is no telemetry endpoint, analytics SDK, remote configuration backend, project API, licence server, GitHub polling, or automatic code/parser download.

## Documentation

- [App configuration](energy_security/DOCS.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Scoring model](docs/SCORING.md)
- [Data sources](docs/DATA_SOURCES.md)
- [Support matrix](docs/SUPPORT_MATRIX.md)
- [Privacy](docs/PRIVACY.md)
- [Memory budget](docs/MEMORY_BUDGET.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Security policy](SECURITY.md)
- [Contributing](CONTRIBUTING.md)

## Licence

Copyright 2026 ArrowSK.

The software is licensed under **PolyForm Noncommercial License 1.0.0**. This is a source-available noncommercial licence and is not an OSI-approved open-source licence. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

External data remains subject to the terms of its respective provider. No third-party dataset is bundled into the executable.
