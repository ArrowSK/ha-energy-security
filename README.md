# Energy Security Monitor for Home Assistant

Energy Security Monitor is a self-contained Home Assistant app that turns public energy-system data into a cautious, inspectable assessment of national energy security.

The normal setup is deliberately simple: add the repository, install the app, start it, and open the Ingress dashboard. By default it uses the country configured for Home Assistant's HOME location. A two-letter ISO country code can be supplied as an override.

The app has no project account, telemetry service, licence server, remote configuration service, MQTT requirement, paid API requirement, or runtime dependency on this repository. Country profiles, provider logic, scoring rules, dashboard assets, a small versioned electricity reference table and fallback behaviour are shipped with the installed app. If this repository becomes unavailable after installation, the running app keeps working with its bundled logic and local cache.

> **Release status:** 0.1.5 is an experimental dashboard-hierarchy and interpretation release. Hungary is the full reference profile. Broader European coverage is graded rather than pretending that every country publishes equivalent data.

## What it measures

Depending on country support and source availability, the app can use:

- electricity generation and live load when available;
- a clearly labelled embedded annual-demand reference when fresh generation exists but live load is absent;
- generation mix, including nuclear, solar, wind, hydro and thermal generation;
- cross-border electricity trading where exposed by the selected feed;
- gas storage/system measurements or a clearly labelled lower-frequency national stock proxy;
- emergency oil-stock days-equivalent for EU profiles;
- hydrological conditions where a country adapter exists; Hungary includes Budapest/Paks Danube evidence and water temperature where published;
- seven-day heat, cold, wind and precipitation stress around HOME;
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

## Zero-configuration default

Normal installation requires no credentials:

```yaml
country: auto
refresh_minutes: 30
enable_ha_entities: true
enable_weather: true
```

Advanced users can optionally provide a GIE AGSI key to improve supported gas-storage coverage and an ENTSO-E Transparency Platform token for configured electricity fallback use. These are optional enhancements, not prerequisites.

All normal options can also be changed from **Dashboard → menu → Setup**. Secret values are never returned from the app backend to the browser; blank secret fields preserve existing values unless the explicit clear control is selected.

## Self-healing behaviour

Each domain has an ordered provider chain. Collection and recovery happen locally:

1. query the preferred supported provider;
2. on failure, try the next provider;
3. after repeated failures, temporarily open a local circuit for the failing provider;
4. keep valid last-known-good observations in `/data`;
5. probe the preferred provider again after cooldown;
6. automatically return to it when it recovers.

Cache writes use a temporary file and atomic rename. Historical dashboard points are bounded. No project server coordinates this process and the app never downloads parser code while running. The container also has a native Docker `HEALTHCHECK` against the local `/healthz` endpoint so process health is checked independently of provider health.

## Dashboard and Home Assistant states

The built-in dashboard is served through Home Assistant Ingress. It uses an Android-style sticky bottom navigator for **Overview, Domains, Signals, Trend and Diagnostics**, keeps manual Refresh as a compact title-bar icon, and places configuration under the title-bar menu.

**Domains** now has a strict hierarchy. Only **Electricity, Gas, Oil reserves, Hydrology and Weather stress** are shown as top-level scored security domains. Supporting evidence is nested underneath the relevant domain rather than being presented as another score. Nuclear and renewable generation therefore belong under Electricity; storage details belong under Gas; reserve measurements belong under Oil; and river/weather observations stay under their respective domains.

Supporting-indicator sections are collapsed by default and expand on click. A section that the user opens stays open during normal dashboard refreshes. For Electricity, generation components show MW plus a percentage share. When live load exists the denominator is current load, matching the Nuclear presentation; if live load is unavailable the UI explicitly switches to percentage of current generation. Cross-border balance shows percentage of current load when possible. Generation diversity remains a normalized structural score, not a generation share.

The older standalone **Generation mix** card is no longer shown because it duplicated the same electricity evidence already available under Electricity supporting indicators.

Diagnostics separates **Sources** from **Measurements**. Measurement rows are grouped into collapsible domain/provider sections, start collapsed, and use larger typography on mobile. A group manually opened by the user remains open through normal refreshes.

The frontend has no external JavaScript, font, analytics or CDN dependency; its assets are embedded in the Go binary.

With `enable_ha_entities: true`, the app also publishes state-machine sensors through Home Assistant's internal REST API, including:

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

The embedded electricity reference is scoring metadata and does not fabricate a live `electricity_load_mw` state.

No MQTT broker or companion cloud service is required.

## Current source strategy

The source order is: established machine-readable feeds first, optional official APIs second, public national pages where no stable unauthenticated structured feed exists, then explicitly labelled local fallbacks/cache where their semantics remain defensible.

| Domain | Primary / fallback sources | Credentials | Current coverage |
|---|---|---:|---|
| Electricity | Energy-Charts → optional ENTSO-E → embedded annual-demand reference when only live load is missing | None / optional | Broad Europe |
| Gas | FGSZ (HU) → optional GIE AGSI → Eurostat monthly gas stocks | None / optional | HU full chain; EU strategic fallback |
| Oil | Eurostat emergency-stock dataset | None | EU profiles |
| Water | HYDROINFO | None | Hungary |
| Weather stress | Open-Meteo | None | HOME coordinates |

The gas fallback deliberately distinguishes actual storage fill from the keyless Eurostat monthly stock proxy. The electricity fallback similarly distinguishes a derived annual-average load from live demand. Neither lower-fidelity fallback is allowed to masquerade as the primary measurement.

See [Data sources](docs/DATA_SOURCES.md), [Electricity reference](docs/ELECTRICITY_REFERENCE.md) and [Support matrix](docs/SUPPORT_MATRIX.md) for exact limitations.

## Installation

Add this repository to Home Assistant's App Store repositories, then install **Energy Security Monitor**.

The repository intentionally omits a runtime `image` setting. Supervisor builds the app locally during installation from the repository rather than requiring a separately managed project container registry. CI builds supported architectures independently for validation. Once installed, normal operation does not contact this repository.

The app opens through Home Assistant Ingress and exposes no user-facing external port.

## Resource budget

The runtime is one Go process plus the Home Assistant base image. There is no Python interpreter, Node process, browser engine, database server or server-side chart renderer.

The release gate is **under 50 MB steady-state RSS**. The engineering target is **35 MB or less** during normal operation. Build-time memory is separate. See [Memory budget](docs/MEMORY_BUDGET.md) for the measurement method.

## Privacy and independence

At runtime the app sends requests only to data providers needed for the configured country, Home Assistant's internal API when state publication is enabled, and Home Assistant Supervisor when the user explicitly saves dashboard Setup. It does not send usage data to ArrowSK, GitHub or any project-operated service.

There is no telemetry endpoint, analytics SDK, remote configuration backend, project API, licence server, GitHub polling, or automatic code/parser download.

## Documentation

- [Home Assistant app guide](energy_security/DOCS.md)
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
