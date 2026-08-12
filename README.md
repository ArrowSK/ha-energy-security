# Energy Security Monitor for Home Assistant

Energy Security Monitor is a self-contained Home Assistant app that turns public energy-system data into a cautious, inspectable assessment of national energy security.

It is designed around a simple operating model: install it, start it, and open the dashboard. By default the app uses the country configured for Home Assistant's HOME location. A country can be overridden with a two-letter ISO code when needed.

The app does not require a project account, an MQTT broker, a paid API, telemetry, or a server operated by this project. It does not fetch configuration or scoring logic from this repository while running. If this repository became unavailable after installation, the installed app would continue using the code, country profiles, source adapters, scoring rules and cache already inside its container.

> **Release status:** 0.1.0 is an experimental first public release. Hungary is the full reference profile. Broader European electricity and oil coverage is available, but country support is intentionally graded rather than pretending that every country publishes equivalent data.

## What it measures

The first release covers the parts of energy security that can be obtained reliably without making users wire together a collection of external services:

- electricity generation and load;
- current generation mix, including nuclear, solar, wind, hydro and thermal generation when the source exposes it;
- cross-border electricity trading where available in the current electricity feed;
- gas storage and gas-system measurements where a country adapter exists;
- emergency oil-stock data from Eurostat for EU profiles, interpreted conservatively;
- hydrological conditions where a country adapter exists; Hungary includes current Budapest and Paks Danube level, discharge and water temperature when published;
- seven-day heat, cold, wind and precipitation stress around HOME;
- generation diversity as a structural resilience signal;
- source freshness, provider failures, fallbacks and confidence.

The architecture deliberately leaves room for additional national adapters: reservoir storage, cooling-water temperature, plant outages, interconnector headroom, LNG send-out, pipeline flows, demand forecasts, pumped storage and batteries can be added without changing the scoring engine or dashboard contract.

## The score is not the data

The dashboard exposes three horizons:

- **Current** — operational conditions now / roughly the next 48 hours.
- **7-day outlook** — current conditions with weather stress added.
- **Strategic resilience** — slower-moving structural signals such as generation diversity, storage, oil data and hydrology.

The headline combines those horizons, but it is always paired with a separate **confidence** value.

Missing data does **not** become zero. Stale data does **not** keep contributing indefinitely. If a source fails, the app first tries its configured fallback; if no fresh fallback is available, it keeps the last known good observation only until that observation's freshness window expires. Missing domains reduce confidence instead of manufacturing a crisis.

The scoring model is deterministic. There is no LLM, news sentiment model or opaque remote scoring service in the decision path.

## Zero-configuration default

Normal installation requires no credentials.

```yaml
country: auto
refresh_minutes: 30
enable_ha_entities: true
enable_weather: true
```

Advanced users can optionally provide:

- a GIE AGSI key to improve gas-storage coverage where supported;
- an ENTSO-E Transparency Platform token for configured fallback use.

These credentials are optional enhancements, not prerequisites.

## Self-healing model

Each data domain has an ordered provider chain. A collector records provider health locally and applies bounded retry behaviour:

1. query the preferred supported provider;
2. on failure, try the next provider;
3. after repeated failures, temporarily open a circuit for the failing provider;
4. keep valid last-known-good observations in the local cache;
5. periodically probe the preferred provider again after its cooldown;
6. automatically return to it once it succeeds.

No remote project service coordinates this process.

Cache writes use a temporary file and atomic rename. The cache lives under Home Assistant app persistent storage (`/data`). Historical dashboard points are bounded to seven days.

## Dashboard

The built-in Ingress dashboard is intentionally progressive rather than a wall of gauges. The first view shows:

- national headline and confidence;
- current, seven-day and strategic scores;
- scored domains;
- current stress signals;
- electricity generation mix;
- seven-day local score history.

The detailed source machinery is under **Diagnostics**, including provider state, source age, measurement quality, fallback errors and stale state.

The UI has no external JavaScript, font, analytics or CDN dependency. It is compiled into the Go binary.

## Home Assistant states

With `enable_ha_entities: true`, the app publishes a compact set of Home Assistant state-machine sensors through Home Assistant's internal REST API, including:

- `sensor.energy_security_score`
- `sensor.energy_security_confidence`
- `sensor.energy_security_status`
- `sensor.energy_security_electricity`
- `sensor.energy_security_gas`
- `sensor.energy_security_oil`
- `sensor.energy_security_water`
- `sensor.energy_security_weather`
- selected measurements such as generation, load, nuclear output, renewable share and gas storage fill when available.

These states are republished after app startup. They are not a separate custom integration and do not require MQTT.

## Current source strategy

The release follows a strict order: official or well-established machine-readable feeds first, optional official APIs second, public national pages where no stable unauthenticated feed exists, then local last-known-good state.

Current adapters include:

| Domain | Primary / optional source | Credentials | Coverage in 0.1.0 |
|---|---|---:|---|
| Electricity | Energy-Charts / optional ENTSO-E fallback | None / optional | Broad Europe |
| Gas | FGSZ Hungary / optional GIE AGSI | None / optional | Hungary full; AGSI enhancement |
| Oil | Eurostat emergency-stock dataset | None | EU profiles, conservative parsing |
| Water | HYDROINFO | None | Hungary |
| Weather stress | Open-Meteo | None | HOME coordinates |

See [Data sources](docs/DATA_SOURCES.md) and [Support matrix](docs/SUPPORT_MATRIX.md) for details and limitations.

## Installation

In Home Assistant, add this repository to the App Store repositories, then install **Energy Security Monitor**.

The repository publishes a multi-architecture container image for `aarch64` and `amd64`. The app opens through Home Assistant Ingress and has no externally exposed port.

For local development, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Resource budget

The runtime is one statically linked Go process plus the Home Assistant base image. There is no Python interpreter, Node process, browser engine, database server or server-side chart renderer.

The release gate is **under 50 MB steady-state RSS**. The engineering target is **35 MB or less** under normal operation. CI enforces build/test correctness; a local process RSS check is documented in [Memory budget](docs/MEMORY_BUDGET.md) because meaningful container RSS should be measured on the target architecture after startup stabilises.

## Privacy and independence

The app sends requests only to the data providers needed for the configured country and, when enabled, to Home Assistant's internal API. It does not send usage data to ArrowSK or this repository.

There is no:

- telemetry endpoint;
- analytics SDK;
- remote configuration service;
- project API/backend;
- licence server;
- GitHub polling at runtime;
- automatic download of code or parsers.

See [Privacy](docs/PRIVACY.md) for the network model.

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
