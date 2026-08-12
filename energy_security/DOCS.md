# Energy Security Monitor — Home Assistant app guide

## Quick start

1. Add `ArrowSK/ha-energy-security` to Home Assistant App Store repositories.
2. Install **Energy Security Monitor**.
3. Start the app with the default options.
4. Open **Energy Security** from the Home Assistant sidebar.

`country: auto` uses the country and coordinates configured for Home Assistant HOME. A normal installation needs no external account, project service, MQTT broker or mandatory API key.

Energy Security Monitor is an analytical monitor, not an official security-of-supply declaration. Read the separate confidence value and Diagnostics page together with the headline score.

## Configuration

| Option | Default | Purpose |
|---|---|---|
| `country` | `auto` | HOME country, or an ISO 3166-1 alpha-2 override such as `HU`, `DE`, `FR` or `GB`. |
| `refresh_minutes` | `30` | Collection interval, 10–180 minutes. Slow monthly datasets remain monthly regardless of this setting. |
| `enable_ha_entities` | `true` | Publish the core assessment and selected measurements into the HA state machine. |
| `enable_weather` | `true` | Include seven-day local weather stress from HOME coordinates. |
| `agsi_key` | empty | Optional higher-quality gas-storage source where supported. |
| `entsoe_token` | empty | Optional electricity fallback where the bundled country profile supports it. |

A country override changes national data-source selection. If HOME coordinates are available, weather continues to use those coordinates; the app does not silently relocate weather to an overridden country's capital.

## What the app can assess

Depending on country support and public-source availability, the app can use electricity generation/load and generation mix, cross-border trading, nuclear/renewable output, gas storage or stock evidence, emergency oil stocks, hydrology and seven-day weather stress.

Not every country publishes equivalent data. Missing evidence remains unknown. It is never converted to zero or guessed from a neighbouring country.

## Score horizons

- **Current** — fresh operational evidence, weighted mainly toward electricity and fresh actual gas-storage evidence.
- **7-day Outlook** — current evidence combined with near-term weather stress.
- **Strategic Resilience** — slower-moving evidence such as generation diversity, gas stocks/storage, emergency oil and hydrology.

The **Headline** combines those horizons. **Confidence** reflects evidence coverage and source quality.

A weaker fallback may still produce a strategic score with reduced confidence. Stale values stop contributing after their TTL.

## Provider order and self-healing

Providers are tried locally in a fixed order. In 0.1.1 the principal chains are:

- Electricity: Energy-Charts → optional ENTSO-E → local last-known-good cache.
- Hungary gas: FGSZ → optional GIE AGSI → keyless Eurostat monthly gas stocks → cache.
- EU gas where Eurostat is enabled: keyless Eurostat monthly gas stocks, with AGSI available when configured.
- Oil: Eurostat emergency-oil dataset → cache.
- Hungary water: HYDROINFO → cache.
- Weather: Open-Meteo → cache.

After repeated provider failures, a bounded local circuit breaker temporarily skips that source. Once the cooldown expires the preferred source is probed again automatically. Successful recovery clears its failure state.

Successful observations are stored in `/data/state.json`. Restarting the app never makes an old observation fresh. Country profiles, parsers, scoring rules and dashboard assets are bundled with the installed release; the runtime does not fetch parser/configuration updates from this repository.

The Home Assistant Supervisor watchdog also checks the app's local `/healthz` endpoint. Provider outages are handled separately by the fallback manager and do not by themselves mean the app process is unhealthy.

## Diagnostics

Provider states mean:

- `healthy` — latest attempt returned valid observations;
- `failed` — latest attempt failed, but normal retry is still allowed;
- `degraded` — repeated failures opened the local cooldown;
- `Last success never` — the current app process has not yet received a valid observation from that provider.

A failed provider is a data-source event, not evidence of an energy emergency. Check whether another source in the same domain supplied the active observation.

Observation rows show source, measurement age, stale state and quality. A provider can be failed while a still-fresh fallback/cache observation remains usable.

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

Selected measurement sensors appear when the underlying observation exists, including electricity generation/load, nuclear output, renewable share, gas storage fill, national gas stock and gas stock index.

The Ingress dashboard does not depend on HA state publication, so it can remain functional if state publication fails.

## 0.1.1 provider fixes

### Energy-Charts `http 404`

The collector now requests an explicit recent multi-day window instead of relying on the provider's implicit current-day window. This avoids false failures around day rollover before the new day has published data.

### Eurostat oil `returned no oil-stock values`

The collector now requests the latest 12 reporting periods and selects the newest period that actually contains the expected emergency-stock series for the selected country. It does not substitute another oil series when that series is absent.

### FGSZ no recognised measurements

FGSZ can expose labels while inserting live numerical values client-side. The app intentionally does not run a browser engine. If the received HTML has no trustworthy live values, FGSZ fails safely and the chain proceeds to AGSI when configured or the keyless Eurostat monthly gas-stock fallback.

### `Last success 739839 d ago`

That was a presentation bug caused by interpreting Go's zero timestamp as a real date. Version 0.1.1 displays `Last success never` until the provider has actually succeeded.

## Troubleshooting checklist

If a domain is Unknown/Data limited:

1. open Diagnostics;
2. identify which provider failed and its exact error;
3. check whether a later provider succeeded;
4. check the observation source, age and stale flag;
5. press **Refresh** once after an app update;
6. if all sources continue failing, report app version, country, provider ID, exact error and approximate refresh time.

After an internet outage, the app restores `/data/state.json`, recalculates stale state from the original timestamps/TTLs, and continues provider retries.

## Further documentation

- repository `README.md` — project overview and installation model;
- `docs/DATA_SOURCES.md` — exact provider behaviour and semantic limitations;
- `docs/SUPPORT_MATRIX.md` — country coverage;
- `docs/PROVIDER_HEALTH.md` — provider states, circuit breaker and fallback semantics;
- `docs/SCORING.md` — deterministic scoring model;
- `docs/ARCHITECTURE.md` — runtime architecture;
- `docs/PRIVACY.md` — network/privacy model;
- `docs/TROUBLESHOOTING.md` — detailed fault isolation;
- `docs/MEMORY_BUDGET.md` — resource measurement procedure.
