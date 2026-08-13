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

## Dashboard hierarchy

Only Electricity, Gas, Oil reserves, Hydrology and Weather stress are top-level scored security domains. Supporting evidence is nested under its parent domain and starts collapsed.

Electricity supporting indicators replace the older standalone Generation mix card. Generation rows show MW plus percentage context: share of current load when live load exists, otherwise explicitly share of current generation. Cross-border balance shows its percentage of current load when possible. Generation diversity remains a normalized structural score rather than a generation share.

Diagnostics separates Sources from Measurements. Measurement groups also start collapsed, and groups or supporting sections manually opened by the user remain open through normal dashboard refreshes.

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
