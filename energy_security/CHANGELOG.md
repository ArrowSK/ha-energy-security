# Changelog

## 0.2.0 — 2026-08-14

### Added

- Standalone runtime mode for Docker and Railway. It uses the exact same `cmd/energy-security` binary, provider manager, scoring engine, cache, country profiles and embedded dashboard as the Home Assistant app.
- Environment-based standalone configuration for country, refresh cadence, optional weather location metadata and optional AGSI/ENTSO-E credentials.
- Normal HTTP serving in standalone mode, with the Home Assistant-specific ingress IP guard applied only to the Home Assistant runtime.
- Repository-root Dockerfile, Docker Compose definition, Railway config-as-code and standalone deployment documentation.
- CI coverage that builds the standalone image, verifies its compiled version against the Home Assistant manifest, validates Compose/Railway configuration and smoke-tests `/healthz` over normal HTTP.

### Shared-core/version contract

- `energy_security/config.yaml` remains the release-version source of truth.
- The standalone Dockerfile reads that Home Assistant manifest at build time and injects its version into the same Go binary. There is no independent standalone version constant to drift.
- Provider, scoring, freshness, fallback, dashboard and cache fixes continue to be made once under `energy_security/` and therefore apply to both deployment modes.

### Home Assistant compatibility

- Home Assistant defaults remain unchanged: `country: auto`, `/data/options.json`, Ingress, Supervisor-backed dashboard Setup and optional HA state publication.
- The existing Home Assistant `energy_security/Dockerfile`, App Store layout and Supervisor option schema remain in place.
- Standalone mode always disables Home Assistant entity publication and never requires a Supervisor token.

### Standalone behavior

- `ENERGY_SECURITY_COUNTRY` is mandatory and `auto` is rejected because standalone deployments have no Home Assistant HOME country to resolve.
- Optional latitude/longitude allow the same Open-Meteo weather logic to run outside Home Assistant. The coordinate pair is validated and must be supplied together.
- Dashboard configuration reads are supported without exposing credential values; writes are rejected because Docker/Railway environment variables are the standalone source of truth.
- `/data` remains the cache/history path and can be mounted as a Docker/Railway volume for persistence.

## 0.1.5 — 2026-08-13

### Dashboard

- Removed the duplicate visible **Generation mix** card from Signals. Electricity generation detail now lives in one place: the expandable supporting indicators inside the Electricity security-domain card.
- Supporting-indicator groups for every security domain start collapsed and expand on click. If the user opens one, that open state is preserved through normal periodic/manual dashboard refreshes.
- Electricity generation components now show their MW value plus percentage context. With live load available, the percentage is the component's share of current load. If live load is unavailable, the UI explicitly switches the denominator to current generation rather than presenting a derived load as live.
- Cross-border imports/exports show their percentage of current load when live load exists.
- Generation diversity remains a normalized 0–100 structural diversity score and is not labelled as a generation share.

### Interpretation

- Nuclear, solar, wind, gas-fired, coal/lignite, biomass, hydro and other source rows are observations, not independent security scores.
- The visual reorganisation does not change scoring weights, freshness handling, provider order, fallback selection, alerts or Home Assistant state publication.

## 0.1.4 — 2026-08-13

### Dashboard

- Reworked **Domains** into a proper hierarchy. Only Electricity, Gas, Oil reserves, Hydrology and Weather stress are rendered as top-level scored security domains.
- Nuclear and Renewables no longer appear as top-level cards with `—` scores. They are supporting electricity observations instead.
- Added compact supporting groups for electricity generation, gas storage, oil reserve evidence, hydrology and weather.
- Exposed generation diversity as the real normalized diversity sub-score used by Strategic Resilience.
- Diagnostics measurement groups now start collapsed and preserve the groups the user manually opens during refreshes.

### Interpretation

- Raw MW, TWh, percentage, river and weather observations are not converted into arbitrary 0–100 security scores merely to fill a progress bar.

## 0.1.3 — 2026-08-13

### Fixed

- Energy-Charts observations no longer become unusable immediately after the old 90-minute threshold. The preferred freshness window remains 90 minutes, but observations have a six-hour hard expiry and confidence decays progressively after 90 minutes.
- GIE AGSI storage observations no longer disappear from scoring merely because the source is two days behind. Storage fill, stored volume, working capacity and storage-versus-consumption use a 48-hour preferred window and seven-day hard expiry.
- AGSI flow/trend observations remain intentionally shorter-lived: 36-hour preferred freshness and four-day hard expiry.
- Scoring now applies age-based quality decay when an observation is delayed but still inside its hard validity window.
- Electricity and gas summaries disclose when confidence has been reduced because a still-usable observation is beyond its preferred freshness window.

### Safety/interpretation

- Hard expiry remains strict. Electricity older than six hours and AGSI storage-level evidence older than seven days are excluded even if the provider request itself succeeds.
- Provider request health and observation freshness remain separate concepts; a healthy provider can legally return a delayed reporting period.

### Tests

- Added regression coverage for two-hour Energy-Charts data, two-day AGSI storage data, AGSI per-metric TTL assignment and hard expiry.

## 0.1.2 — 2026-08-13

### Added

- Sticky Android-style bottom navigation for the dashboard sections.
- Compact title-bar Refresh and hamburger actions.
- Dashboard Setup editor backed by Home Assistant Supervisor's self-app options API. Secret values are never returned to the browser; unchanged blank secret fields preserve current values.
- Separate Sources and Measurements diagnostic cards.
- Collapsible measurement groups and larger diagnostic/mobile typography.
- Embedded electricity demand/population reference data for all 39 country profiles.

### Changed

- If fresh electricity generation exists but live load does not, the electricity scorer can use a clearly labelled annual-average demand reference instead of returning no electricity score.
- Reference-load confidence is 45% of generation-source quality and its Current-composite weight is reduced from 0.50 to 0.25.
- A score based on the annual reference cannot by itself emit the live electricity-stress alert.
- Version and runtime user agent are 0.1.2.

### Data attribution

- Embedded electricity references are derived from Ember yearly electricity-demand data (CC BY 4.0). See `docs/ELECTRICITY_REFERENCE.md` and `THIRD_PARTY_LICENSES.md`.

## 0.1.1 — 2026-08-13

### Fixed

- Energy-Charts no longer relies on the provider's implicit current-local-day window. The collector requests an explicit recent window, preventing false `HTTP 404` failures around day rollover while still selecting the newest usable sample.
- Eurostat oil collection now searches the latest 12 reporting periods and selects the newest available `STK_EUE_DIR` value for the selected country. Countries that report one or more months behind the newest dataset period no longer fail merely because their newest cell is empty.
- Diagnostics now renders an unset provider `last_success` timestamp as `never` instead of treating Go's zero time as an ancient real date.
- The FGSZ adapter now reports explicitly when the current public page contains labels but no server-rendered live values, so fallback behaviour is visible and deterministic.

### Added

- `eurostat_gas`: keyless monthly natural-gas closing-stock fallback for Eurostat-enabled profiles using `nrg_stk_gasm`, natural gas (`G3000`), national closing stock (`STKCL_NAT`) and `TJ_GCV`.
- `gas_national_stock_twh` and `gas_stock_index_pct` observations. The index compares the latest monthly national closing stock with the maximum in the returned 36-month window and is intentionally lower confidence than a true physical storage-fill measurement.
- Home Assistant state sensors for the Eurostat gas stock and stock index when those observations are active.
- Native Docker `HEALTHCHECK` monitoring through the local `/healthz` endpoint.
- App Store icon and logo assets.
- Expanded Home Assistant-facing documentation and provider troubleshooting.

### Safety/interpretation

- The Eurostat gas stock index is not labelled or scored as physical storage-capacity fill. It remains a data-limited strategic signal and cannot create a Current-horizon score by itself.
- Existing actual storage-fill observations from FGSZ/GIE remain preferred over the monthly Eurostat proxy.

## 0.1.0 — 2026-08-12

Initial experimental public release.

- HOME-country autodetection with ISO country override.
- Keyless Energy-Charts electricity collection with optional ENTSO-E fallback.
- Hungary FGSZ gas and HYDROINFO water adapters.
- Eurostat emergency-oil adapter for EU profiles.
- Open-Meteo seven-day weather-stress input.
- Local deterministic scoring, confidence, fallbacks, circuit breaker and atomic cache.
- Home Assistant Ingress dashboard and optional state-machine sensors.
- No project telemetry, backend, remote configuration or runtime GitHub dependency.
