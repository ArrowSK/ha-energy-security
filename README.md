<p align="center">
  <img src="energy_security/logo.png" width="120" alt="Energy Security Monitor logo">
</p>

<h1 align="center">Energy Security Monitor</h1>

<p align="center">
  <strong>A practical, transparent view of national energy security — without turning missing or delayed data into false certainty.</strong>
</p>

<p align="center">
  <a href="https://github.com/ArrowSK/ha-energy-security/actions/workflows/ci.yaml"><img src="https://github.com/ArrowSK/ha-energy-security/actions/workflows/ci.yaml/badge.svg?branch=main" alt="CI status"></a>
  <a href="https://github.com/ArrowSK/ha-energy-security/actions/workflows/lint.yaml"><img src="https://github.com/ArrowSK/ha-energy-security/actions/workflows/lint.yaml/badge.svg?branch=main" alt="Home Assistant app lint"></a>
  <img src="https://img.shields.io/badge/Home%20Assistant-App-41BDF5?logo=home-assistant&logoColor=white" alt="Home Assistant app">
  <img src="https://img.shields.io/badge/Docker-Standalone-2496ED?logo=docker&logoColor=white" alt="Docker standalone">
  <img src="https://img.shields.io/badge/License-PolyForm%20Noncommercial-6f42c1" alt="PolyForm Noncommercial license">
</p>

<p align="center">
  <a href="https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FArrowSK%2Fha-energy-security"><img src="https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg" alt="Add Energy Security Monitor repository to Home Assistant" height="40"></a>
  &nbsp;
  <a href="https://railway.com/new"><img src="https://railway.com/button.svg" alt="Deploy on Railway" height="40"></a>
  &nbsp;
  <a href="#docker--docker-compose"><img src="https://img.shields.io/badge/Run%20with-Docker%20Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Run with Docker Compose" height="40"></a>
</p>

<p align="center">
  <sub>Home Assistant · Docker / Docker Compose · Railway — all running the same core, scoring engine, providers and dashboard.</sub>
</p>

---

## What it gives you

Energy Security Monitor is built for the question people actually care about: **how exposed or resilient does the current energy position look, and how much confidence should you place in that answer?**

| | |
|---|---|
| **Five scored security domains** | Electricity, gas, oil reserves, hydrology and weather stress |
| **Three time horizons** | Current conditions, 7-day outlook and strategic resilience |
| **Confidence kept separate from score** | Missing or delayed data lower confidence instead of manufacturing a crisis |
| **Visible evidence** | Supporting indicators stay inspectable instead of disappearing behind one headline |
| **Source-aware fallbacks** | Provider failures, delayed publication and lower-fidelity fallbacks are shown in Diagnostics |
| **Mobile-friendly dashboard** | Compact navigation, collapsible supporting data and diagnostics designed for a phone as well as desktop |

No project account is required. There is no project telemetry service, licence server, analytics SDK, MQTT requirement, paid API requirement or project-operated cloud backend. Optional GIE AGSI and ENTSO-E credentials can improve coverage where supported, but normal use does not depend on them.

> **Current release: 0.2.0.** Hungary is the full reference profile. Broader European coverage is intentionally graded rather than pretending that every country publishes the same quality of electricity, gas, oil and hydrological data.

## Choose how to run it

### Home Assistant

The most integrated option if you already run Home Assistant.

[![Add repository to Home Assistant](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FArrowSK%2Fha-energy-security)

The button opens your Home Assistant instance with this repository URL pre-filled. Then install **Energy Security Monitor** from the App Store, start it, and open **Energy Security** from the sidebar.

Home Assistant mode adds Ingress, `country: auto`, dashboard Setup and optional HA sensors while keeping the same underlying core used everywhere else.

[Home Assistant app guide →](energy_security/DOCS.md)

### Docker / Docker Compose

Best for a NAS, home server, VPS, Raspberry Pi or homelab when you want the monitor as a normal standalone web service.

```sh
cp .env.example .env
# set ENERGY_SECURITY_COUNTRY and any optional location values
docker compose up -d --build
```

Open `http://localhost:8099` unless you changed the exposed port. A named `/data` volume keeps cache and dashboard history across container replacement.

[Standalone Docker guide →](docs/STANDALONE.md#docker-compose)

### Railway

Best if you want a hosted deployment without maintaining your own server.

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/new)

Railway builds the repository-root Dockerfile and uses the included `railway.toml`. In the new-project flow, select `ArrowSK/ha-energy-security`, set at least `ENERGY_SECURITY_COUNTRY` to a two-letter country code, and deploy. Railway supplies `PORT` automatically.

The Railway button currently opens Railway's standard new-project flow rather than a published marketplace template, so it does not hide the required country choice from you.

[Railway deployment guide →](docs/STANDALONE.md#railway)

## How to read the dashboard

| Current | 7-day outlook | Strategic resilience |
|---|---|---|
| Operational conditions now and roughly the next 48 hours | Current conditions plus near-term weather stress | Slower-moving structural signals such as generation diversity, gas stocks/storage, oil evidence and hydrology |

The headline combines those horizons, but **confidence is always shown separately**. A lower-confidence score means the app has less or weaker evidence — not that the energy situation itself is automatically worse.

Only the main domains receive security scores. Supporting observations stay under their parent domain: nuclear, solar, wind and thermal generation under Electricity; storage details under Gas; reserve evidence under Oil; river and weather readings under their own domains. These sections start collapsed and expand when you want the detail.

Diagnostics deliberately separates **Sources** from **Measurements**, so a provider problem is not confused with an energy-security event.

## What it can measure

Depending on country support and source availability, the monitor can use:

- electricity generation and live load;
- a clearly labelled annual-demand reference when fresh generation exists but live load is absent;
- nuclear, solar, wind, hydro and thermal generation mix;
- cross-border electricity trading;
- gas storage/system measurements or a lower-frequency national stock proxy;
- emergency oil-stock days-equivalent for EU profiles;
- hydrological conditions where a country adapter exists;
- seven-day heat, cold, wind and precipitation stress around the configured location;
- generation diversity as a structural resilience signal;
- source freshness, fallbacks and provider health.

The scoring model is deterministic. No LLM, news-sentiment model or remote scoring service sits in the decision path.

## Data-source strategy

| Domain | Primary / fallback sources | Credentials |
|---|---|---:|
| Electricity | Energy-Charts → optional ENTSO-E → embedded annual-demand reference when only live load is missing | None / optional |
| Gas | FGSZ (HU) → optional GIE AGSI → Eurostat monthly gas stocks | None / optional |
| Oil | Eurostat emergency-stock dataset | None |
| Hydrology | HYDROINFO where supported | None |
| Weather stress | Open-Meteo | None |

Lower-fidelity fallbacks are labelled as such. A monthly gas-stock proxy is not presented as physical real-time storage fill, and an annual-average electricity reference is not presented as live demand.

[Data sources →](docs/DATA_SOURCES.md) · [Support matrix →](docs/SUPPORT_MATRIX.md) · [Electricity reference →](docs/ELECTRICITY_REFERENCE.md)

## One core, every installation

Home Assistant, standalone Docker and Railway are deployment wrappers around the same application:

```text
energy_security/cmd/energy-security   shared binary entry point
energy_security/internal/             providers, scoring, cache, history, dashboard
energy_security/Dockerfile            Home Assistant wrapper
Dockerfile                             standalone Docker / Railway wrapper
```

`energy_security/config.yaml` is the release-version source of truth. The standalone Docker build reads that manifest and compiles the same version into the same binary; CI verifies the versions stay aligned. A provider, scoring, freshness or dashboard fix is therefore made once and inherited by every deployment mode.

## Privacy and independence

The application contacts only the data providers needed for the configured country. Home Assistant mode additionally talks to Home Assistant's internal API when HA sensor publication is enabled and to Supervisor when you explicitly save dashboard Setup. Standalone mode does neither.

It does **not** send usage data to ArrowSK, GitHub or any project-operated backend. Runtime provider logic, country profiles, scoring rules, dashboard assets and fallback behaviour ship with the installed release. A running deployment does not need this GitHub repository to remain online.

Standalone deployments do not add authentication by themselves. If you expose one publicly, put it behind an authenticated reverse proxy or keep it on a private network.

[Privacy model →](docs/PRIVACY.md) · [Architecture →](docs/ARCHITECTURE.md) · [Security policy →](SECURITY.md)

## Documentation

| Start here | Deeper reference |
|---|---|
| [Home Assistant app guide](energy_security/DOCS.md) | [Scoring model](docs/SCORING.md) |
| [Docker & Railway guide](docs/STANDALONE.md) | [Provider health & fallbacks](docs/PROVIDER_HEALTH.md) |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | [Support matrix](docs/SUPPORT_MATRIX.md) |
| [Data sources](docs/DATA_SOURCES.md) | [Memory budget](docs/MEMORY_BUDGET.md) |
| [Privacy](docs/PRIVACY.md) | [Contributing](CONTRIBUTING.md) |

## Licence

Copyright 2026 ArrowSK.

Energy Security Monitor is licensed under the **PolyForm Noncommercial License 1.0.0**. It is source-available for noncommercial use and is not an OSI-approved open-source licence. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

Runtime provider data remains subject to the respective provider terms. The embedded electricity reference is derived from Ember yearly demand data under CC BY 4.0 and is attributed separately in [Third-party software and data](THIRD_PARTY_LICENSES.md).
