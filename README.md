# Energy Security Monitor

A practical dashboard for understanding a country's energy position — without pretending that missing or delayed data are certainty.

**Run it the way that fits your setup:**

| Installation | Best fit | What you get |
|---|---|---|
| **Home Assistant app** | You already run Home Assistant | Ingress dashboard, dashboard Setup, optional HA sensors, automatic HOME location support |
| **Docker / Docker Compose** | NAS, home server, VPS, Raspberry Pi, homelab | The same dashboard and scoring engine as a normal standalone web service |
| **Railway** | You want a hosted deployment without managing a server | The same standalone service, built directly from this repository |

All three use the **same Go core, providers, scoring rules, fallback logic and dashboard**. There is no separate "Docker edition" to drift away from Home Assistant. A fix to the shared core is a fix for every deployment mode.

## Highlights

- **One view of several energy-security dimensions** — electricity, gas, oil reserves, hydrology and weather stress.
- **Transparent rather than dramatic** — every score has a separate confidence value, and missing data reduce confidence instead of being turned into a fake emergency.
- **Built around public data and fallbacks** — provider failures, delayed publication and lower-quality fallback data are visible in Diagnostics.
- **No project account or telemetry service** — no licence server, analytics SDK, MQTT requirement, paid API requirement or project-operated cloud backend.
- **Useful on a phone** — the dashboard is designed for both mobile and desktop, with collapsible supporting evidence instead of endless raw tables.
- **One codebase for every installation option** — Home Assistant, Docker and Railway stay version-bound to the same release.

Energy Security Monitor is meant to answer a fairly simple question: **how exposed or resilient does the current energy position look, and how much confidence should you place in that answer?** It gathers public energy-system data, applies deterministic scoring, keeps the underlying measurements inspectable, and clearly marks when a conclusion is based on delayed, incomplete or lower-fidelity evidence.

No account is required for normal use. Most deployments also need no API keys at all. Optional GIE AGSI and ENTSO-E credentials can improve coverage where supported, but they are enhancements rather than prerequisites.

> **Current release: 0.2.0.** Hungary is the full reference profile. Broader European coverage is intentionally graded rather than pretending that every country publishes the same quality of electricity, gas, oil and hydrological data.

## Choose an installation

### Home Assistant

If you already use Home Assistant, this is the most integrated option. Add this repository to the Home Assistant App Store, install **Energy Security Monitor**, start it, and open **Energy Security** from the sidebar.

The default `country: auto` uses the country and coordinates configured for Home Assistant HOME. The app opens through Ingress, can publish selected HA sensors, and lets you change normal options from **Dashboard → menu → Setup**.

See the [Home Assistant app guide](energy_security/DOCS.md).

### Docker / Docker Compose

If you do not use Home Assistant, or simply want the monitor on a NAS, server, VPS or homelab, run the standalone container instead.

```sh
cp .env.example .env
# edit .env, then:
docker compose up -d --build
```

Standalone mode uses environment variables, serves the same embedded dashboard over normal HTTP, and stores cache/history under `/data` when a persistent volume is used.

See the [Standalone Docker and Railway guide](docs/STANDALONE.md).

### Railway

Railway uses the same root Dockerfile as standalone Docker. Create a service from this repository's `main` branch, set at least `ENERGY_SECURITY_COUNTRY` to a two-letter country code, and deploy. The included `railway.toml` configures the Docker build, `/healthz` check and restart policy; Railway's injected `PORT` is used automatically.

The Railway deployment is not a separate application. It is the same binary and shared core as the Home Assistant and Docker builds.

## What it measures

Depending on country support and source availability, the monitor can use:

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

The provider model is intentionally extensible. Reservoir storage, cooling-water constraints, plant outages, interconnector headroom, LNG send-out, pipeline flows, demand forecasts, pumped storage and batteries can be added later without replacing the scoring/dashboard contract.

## Scores and confidence

The dashboard exposes three horizons:

- **Current** — operational conditions now and roughly the next 48 hours.
- **7-day outlook** — current conditions with near-term weather stress.
- **Strategic resilience** — slower-moving structural signals such as generation diversity, storage/stocks, oil evidence and hydrology.

The headline combines those horizons and is always accompanied by a separate **confidence** value.

Missing data never becomes zero. Stale data does not contribute indefinitely. If a source fails, the app tries the next configured provider. If no fresh fallback exists, a last-known-good observation is retained only inside its defined freshness window. Missing domains reduce confidence instead of manufacturing a crisis.

When live electricity load is absent but fresh generation is present, the app can use an embedded recent annual-demand reference for the selected country. It converts annual demand into an annual-average MW value, explicitly labels that derivation in the electricity description, sharply reduces confidence and composite weight, and never treats the reference as current or peak demand. Fresh live load always wins.

The scoring model is deterministic. No LLM, news sentiment model or remote scoring service is in the decision path.

## One core, several deployment wrappers

The installation options do not maintain separate provider or scoring implementations:

- shared binary entry point: `energy_security/cmd/energy-security`;
- shared core: `energy_security/internal/`;
- Home Assistant container wrapper: `energy_security/Dockerfile` plus Supervisor options/Ingress behaviour;
- standalone Docker/Railway wrapper: repository-root `Dockerfile` plus environment configuration.

`energy_security/config.yaml` is the release-version source of truth. The root standalone Dockerfile reads that Home Assistant manifest during its build and compiles the same version into the standalone binary. CI verifies that the standalone image's `--version` exactly matches the manifest.

That means a provider, scoring, freshness or dashboard fix is made once in the shared core and inherited by every deployment mode.

## Home Assistant behaviour

Normal Home Assistant installation requires no credentials:

```yaml
country: auto
refresh_minutes: 30
enable_ha_entities: true
enable_weather: true
```

Advanced users can optionally provide a GIE AGSI key to improve supported gas-storage coverage and an ENTSO-E Transparency Platform token for configured electricity fallback use.

All normal Home Assistant options can also be changed from **Dashboard → menu → Setup**. Secret values are never returned from the app backend to the browser; blank secret fields preserve existing values unless the explicit clear control is selected.

The Home Assistant app intentionally omits a runtime `image` setting. Supervisor builds the app locally during installation from the repository rather than requiring a separately managed project container registry. Once installed, normal operation does not contact this repository.

The Home Assistant app opens through Ingress and exposes no user-facing external port.

## Standalone behaviour

Standalone mode uses environment variables because there is no Home Assistant Supervisor. It deliberately disables HA entity publication and never tries to contact Supervisor.

A `/data` volume is recommended so cached observations and dashboard history survive container replacement. Setup values are read-only in the dashboard in standalone mode; change the Docker/Railway environment variables and restart or redeploy instead.

Standalone deployments do not add an authentication layer by themselves. If the dashboard should not be public, keep it on a private network or place it behind an authenticated reverse proxy.

See [Standalone Docker and Railway](docs/STANDALONE.md) for the complete variable list and deployment examples.

## Self-healing behaviour

Each domain has an ordered provider chain. Collection and recovery happen locally:

1. query the preferred supported provider;
2. on failure, try the next provider;
3. after repeated failures, temporarily open a local circuit for the failing provider;
4. keep valid last-known-good observations in `/data`;
5. probe the preferred provider again after cooldown;
6. automatically return to it when it recovers.

Cache writes use a temporary file and atomic rename. Historical dashboard points are bounded. No project server coordinates this process and the app never downloads parser code while running. Both container variants expose `/healthz`; the Home Assistant image also keeps its native Docker `HEALTHCHECK` independently of provider health.

## Dashboard

The built-in dashboard uses an Android-style sticky bottom navigator for **Overview, Domains, Signals, Trend and Diagnostics**. Manual Refresh is a compact title-bar icon; configuration lives under the title-bar menu where that deployment mode supports editing.

**Domains** has a strict hierarchy. Only **Electricity, Gas, Oil reserves, Hydrology and Weather stress** are shown as top-level scored security domains. Supporting evidence is nested underneath the relevant domain rather than being presented as another score. Nuclear and renewable generation belong under Electricity; storage details belong under Gas; reserve measurements belong under Oil; and river/weather observations stay under their respective domains.

Supporting-indicator sections are collapsed by default and expand on click. A section that the user opens stays open during normal dashboard refreshes. For Electricity, generation components show MW plus a percentage share. When live load exists the denominator is current load; if live load is unavailable the UI explicitly switches to percentage of current generation. Cross-border balance shows percentage of current load when possible. Generation diversity remains a normalized structural score, not a generation share.

Diagnostics separates **Sources** from **Measurements**. Measurement rows are grouped into collapsible domain/provider sections, start collapsed, and use larger typography on mobile. A group manually opened by the user remains open through normal refreshes.

The frontend has no external JavaScript, font, analytics or CDN dependency; its assets are embedded in the Go binary.

With `enable_ha_entities: true`, the Home Assistant app can also publish state-machine sensors including:

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

## Resource budget

The runtime is one Go process plus a small container base. There is no Python interpreter, Node process, browser engine, database server or server-side chart renderer.

For the Home Assistant build, the release gate is **under 50 MB steady-state RSS**. The engineering target is **35 MB or less** during normal operation. Build-time memory is separate. See [Memory budget](docs/MEMORY_BUDGET.md) for the measurement method.

## Privacy and independence

At runtime the application sends requests only to data providers needed for the configured country. Home Assistant mode additionally uses Home Assistant's internal API when state publication is enabled and Home Assistant Supervisor when the user explicitly saves dashboard Setup. Standalone mode does neither.

It does not send usage data to ArrowSK, GitHub or any project-operated service. There is no telemetry endpoint, analytics SDK, remote configuration backend, project API, licence server, GitHub polling, or automatic code/parser download.

The project has no account, telemetry service, licence server, remote configuration service, MQTT requirement, paid API requirement, or runtime dependency on this repository. Country profiles, provider logic, scoring rules, dashboard assets, the versioned electricity reference table and fallback behaviour are shipped with the application binary. If this repository becomes unavailable after installation, a running deployment keeps working with its bundled logic and local cache.

## Documentation

Start with the guide for the way you want to run it:

- [Home Assistant app guide](energy_security/DOCS.md)
- [Standalone Docker and Railway](docs/STANDALONE.md)

Then, for the underlying model and technical detail:

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
