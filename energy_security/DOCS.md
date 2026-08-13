# Energy Security Monitor — Home Assistant app guide

## Quick start

1. Add `ArrowSK/ha-energy-security` to Home Assistant App Store repositories.
2. Install **Energy Security Monitor**.
3. Start the app with the default options.
4. Open **Energy Security** from the Home Assistant sidebar.

`country: auto` uses the country and coordinates configured for Home Assistant HOME. A normal installation needs no external account, project service, MQTT broker or mandatory API key.

Energy Security Monitor is an analytical monitor, not an official security-of-supply declaration. Read confidence and Diagnostics together with the headline score.

## Dashboard navigation

The dashboard uses a sticky bottom navigation bar designed for both phone and desktop use. The five destinations are **Overview**, **Domains**, **Signals**, **Trend** and **Diagnostics**. The currently visible section is highlighted as the page scrolls.

Manual refresh is the circular-arrow action in the title bar. The menu action beside it opens **Setup**.

## Domain hierarchy and supporting indicators

The dashboard distinguishes a security score from the observations that explain it.

Only five top-level cards are scored security domains:

- **Electricity**
- **Gas**
- **Oil reserves**
- **Hydrology**
- **Weather stress**

Nuclear output, renewable share, individual generation sources, gas-storage details, reserve measurements, river measurements and weather readings are supporting evidence. They are not promoted into separate 0–100 security scores merely to fill a progress bar.

Each domain has an expandable supporting-indicator section. These sections start **collapsed by default** and open on click. Once a section is manually opened, the dashboard preserves that state through normal periodic or manual refreshes.

Electricity is the most detailed example. Its supporting indicators replace the older standalone **Generation mix** card so the same generation evidence is not shown twice. Generation rows show their MW value and a percentage share:

- when live electricity load is available, the percentage is **share of current load**;
- when live load is unavailable but generation is available, the UI explicitly uses **share of current generation** instead;
- cross-border imports/exports show their percentage of current load when live load exists;
- **Generation diversity** remains a normalized structural 0–100 score derived from the mix and is deliberately not labelled as a generation share.

The aggregate Renewable share observation remains a percentage of current load as supplied/derived by the electricity feed. Individual source rows such as Nuclear, Solar, Wind, Gas-fired, Coal/lignite, Biomass and Hydroelectric provide their own MW and percentage context.

## Configuration and dashboard Setup

All normal app options can be changed from **menu → Setup** without leaving the dashboard. Saving asks Home Assistant Supervisor to update this app's own options and restart the app so the new configuration is loaded cleanly.

| Option | Default | Purpose |
|---|---|---|
| `country` | `auto` | HOME country, or an ISO 3166-1 alpha-2 override such as `HU`, `DE`, `FR` or `GB`. |
| `refresh_minutes` | `30` | Collection interval, 10–180 minutes. Slow monthly datasets remain monthly regardless of this setting. |
| `enable_ha_entities` | `true` | Publish the core assessment and selected measurements into the HA state machine. |
| `enable_weather` | `true` | Include seven-day local weather stress from HOME coordinates. |
| `agsi_key` | empty | Optional higher-quality gas-storage source where supported. |
| `entsoe_token` | empty | Optional electricity fallback where the bundled country profile supports it. |

Credential handling in Setup is deliberately asymmetric: the backend tells the browser only whether a key/token is configured. It never sends the stored value back to the browser. Leaving a credential field blank preserves the current secret; an explicit **Clear saved key/token** control removes it.

A country override changes national data-source selection. If HOME coordinates are available, weather continues to use those coordinates; the app does not silently relocate weather to an overridden country's capital.

## What the app can assess

Depending on country support and public-source availability, the app can use electricity generation/load and generation mix, cross-border trading, nuclear/renewable output, gas storage or stock evidence, emergency oil stocks, hydrology and seven-day weather stress.

Not every country publishes equivalent data. Missing evidence remains unknown unless a specifically documented lower-confidence fallback exists. It is never converted to zero or guessed from a neighbouring country.

## Electricity when live load is missing

A recurring source limitation is that generation may be available while current load/consumption is absent. The app has an embedded electricity reference row for every current country profile. The reference contains a source year, annual electricity demand, annual demand per capita and approximate population derived from the same source row.

If fresh usable generation is present but live load is not, the app can derive an annual-average reference load:

`annual demand TWh × 1,000,000 / 8,760 = average MW`

This is not current or peak demand. The electricity card explicitly says that **live consumption is unavailable**, identifies the Ember reference year and displays the annual demand/population/per-capita basis. Most references use 2024; Ukraine uses 2022 and that older year is shown rather than hidden.

Safeguards are intentional: confidence is reduced to 45% of live generation quality, electricity's Current-composite weight is halved from 0.50 to 0.25, and the derived path cannot emit the live electricity-stress alert by itself. Fresh live load always wins immediately.

The fallback reference is used by scoring only. It does not fabricate a live `electricity_load_mw` observation or Home Assistant sensor. In the supporting-indicator UI, percentage labels therefore use current generation as the explicit denominator when no live load observation exists.

See `docs/ELECTRICITY_REFERENCE.md` for the complete methodology and attribution.

## Score horizons

- **Current** — fresh or still-usable operational evidence, weighted mainly toward electricity and actual gas-storage evidence. A derived annual electricity reference may contribute only at reduced weight/confidence.
- **7-day Outlook** — current evidence combined with near-term weather stress.
- **Strategic Resilience** — slower-moving evidence such as generation diversity, gas stocks/storage, emergency oil and hydrology.

The **Headline** combines those horizons. **Confidence** reflects evidence coverage and source quality.

A weaker fallback may still produce a score with reduced confidence. Observations stop contributing after their hard validity window.

## Freshness and delayed reporting

Provider request health and observation freshness are separate concepts. A provider can answer successfully today while the newest published observation is older.

Energy-Charts electricity observations have a preferred freshness window of 90 minutes and a six-hour hard expiry. Between those points the data remain usable while confidence decays progressively.

GIE AGSI storage-level evidence is daily data: storage fill, stored volume, working capacity and storage-versus-consumption have a 48-hour preferred window and a seven-day hard expiry. Daily flow/trend observations use a shorter 36-hour preferred window and four-day hard expiry.

This avoids both extremes: a modest publication delay does not erase an otherwise useful domain score, while genuinely old cache data cannot remain current indefinitely.

## Provider order and self-healing

Providers are tried locally in a fixed order. The principal chains are:

- Electricity: Energy-Charts → optional ENTSO-E → local last-known-good cache; if usable generation survives but live load does not, the embedded annual demand reference may be used for scoring.
- Hungary gas: FGSZ → optional GIE AGSI → keyless Eurostat monthly gas stocks → cache.
- EU gas where Eurostat is enabled: keyless Eurostat monthly gas stocks, with AGSI available when configured.
- Oil: Eurostat emergency-oil dataset → cache.
- Hungary water: HYDROINFO → cache.
- Weather: Open-Meteo → cache.

After repeated provider failures, a bounded local circuit breaker temporarily skips that source. Once the cooldown expires the preferred source is probed again automatically. Successful recovery clears its failure state.

Successful observations are stored in `/data/state.json`. Restarting the app never makes an old observation fresh. Country profiles, parsers, scoring rules, dashboard assets and electricity reference data are bundled with the installed release; the runtime does not fetch parser/configuration updates from this repository.

The container includes a native Docker `HEALTHCHECK` against the app's local `/healthz` endpoint. That covers process/service health independently of provider failures.

## Diagnostics

Diagnostics is split into two distinct areas.

**Sources** shows provider health, last success/failure information, latency and collection notes. A failed provider is a data-source event, not evidence of an energy emergency; another source in the same domain may still be active.

**Measurements** shows the observations separately. Measurements are grouped into collapsible electricity/gas/oil/hydrology/weather/etc. sections so a mobile screen is not forced through one long flat list. Every measurement group starts collapsed. If the user opens one, that state is preserved during normal dashboard refreshes. Each row shows value, source, age/stale state and quality.

Provider states mean:

- `healthy` — latest attempt returned valid observations;
- `failed` — latest attempt failed, but normal retry is still allowed;
- `degraded` — repeated failures opened the local cooldown;
- `Last success never` — the current app process has not yet received a valid observation from that provider.

## Gas fallback semantics

The app deliberately separates unlike measurements.

`gas_storage_fill_pct` means an actual physical storage-fill percentage from a source that publishes it.

`gas_stock_index_pct` is the keyless Eurostat monthly fallback: latest reported national closing stock divided by the maximum monthly closing stock in the returned 36-month window. It is a lower-confidence strategic stock-position proxy, **not physical storage capacity fill**. It cannot create a Current-horizon score by itself. Actual FGSZ/GIE fill remains preferred whenever available.

The related raw monthly amount is exposed as `gas_national_stock_twh`.

## Home Assistant entities

With `enable_ha_entities: true`, the app publishes:

- `sensor.energy_security_score`
- `sensor.energy_security_confidence`
- `sensor.energy_security_status`
- `sensor.energy_security_electricity`
- `sensor.energy_security_gas`
- `sensor.energy_security_oil`
- `sensor.energy_security_water`
- `sensor.energy_security_weather`

Selected measurement sensors appear when the underlying runtime observation exists, including electricity generation/load, nuclear output, renewable share, gas storage fill, national gas stock and gas stock index. The derived electricity reference is scoring metadata, not a fabricated `electricity_load_mw` sensor.

The Ingress dashboard does not depend on HA state publication, so it can remain functional if state publication fails.

## Troubleshooting checklist

If a domain is Unknown/Data limited:

1. open Diagnostics → Sources;
2. identify which provider failed and its exact error;
3. check whether a later provider succeeded;
4. open Diagnostics → Measurements and expand the relevant group to check source, age and stale flag;
5. press the title-bar Refresh icon once after an app update;
6. if all sources continue failing, report app version, country, provider ID, exact error and approximate refresh time.

If Setup cannot save, check the app log for `dashboard setup save failed`. The implementation uses Home Assistant Supervisor's documented `/addons/self/*` API, which apps may call for their own lifecycle/options without requesting broad Supervisor API access.

After an internet outage, the app restores `/data/state.json`, recalculates stale state from the original timestamps/TTLs, and continues provider retries.

## Further documentation

- repository `README.md` — project overview and installation model;
- `docs/DATA_SOURCES.md` — exact provider behaviour and semantic limitations;
- `docs/ELECTRICITY_REFERENCE.md` — embedded annual-demand fallback methodology;
- `docs/SUPPORT_MATRIX.md` — country coverage;
- `docs/PROVIDER_HEALTH.md` — provider states, circuit breaker and fallback semantics;
- `docs/SCORING.md` — deterministic scoring model;
- `docs/ARCHITECTURE.md` — runtime architecture;
- `docs/PRIVACY.md` — network/privacy model;
- `docs/TROUBLESHOOTING.md` — detailed fault isolation;
- `docs/MEMORY_BUDGET.md` — resource measurement procedure.
