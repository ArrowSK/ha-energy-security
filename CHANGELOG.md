# Changelog

## 0.2.0 — 2026-08-14

Shared-core standalone deployment release.

- Added a repository-root standalone Docker image and `compose.yaml` while preserving the existing Home Assistant app container and Ingress runtime.
- Added Railway config-as-code with root Dockerfile build, `/healthz` deployment checking and on-failure restart policy.
- Added standalone environment configuration for explicit country, refresh cadence, optional weather coordinates/location metadata and optional AGSI/ENTSO-E credentials. Home Assistant entity publication is always disabled outside Home Assistant.
- Kept one implementation of providers, scoring, cache, history and dashboard: both deployment modes execute the same `energy_security/cmd/energy-security` binary and `energy_security/internal/` core.
- Bound standalone image versioning to the Home Assistant app manifest. The root Dockerfile reads `energy_security/config.yaml` at build time, and CI requires the standalone binary version to match it exactly.
- Preserved Home Assistant behaviour: `country: auto`, `/data/options.json`, Supervisor-backed dashboard Setup, Ingress-only request guarding and optional HA state publication remain the Home Assistant path.
- Added standalone normal-HTTP serving without the Home Assistant Ingress IP guard. Dashboard configuration mutation is read-only in standalone mode because Docker/Railway environment variables are the source of truth.
- Added standalone Docker build, manifest-version, Compose, Railway-config and HTTP smoke tests to CI.
- Added a dedicated Docker/Railway deployment guide and secret-safe `.env.example`/`.gitignore` handling.

## 0.1.5 — 2026-08-13

Dashboard hierarchy refinement.

- Removed the duplicate visible Generation mix card from Signals; electricity generation detail now has one canonical presentation under Electricity supporting indicators.
- Made every domain's supporting-indicator section collapsed by default and expandable on click. A section the user opens remains open through normal dashboard refreshes.
- Added percentage context to electricity generation components. When live load is available, each generation source is shown as a percentage of current load; if live load is unavailable, the fallback basis is explicitly current generation instead.
- Added percentage-of-load context for cross-border electricity balance when live load is available.
- Kept generation diversity as a normalized structural score rather than mislabelling it as a generation share.
- Kept the scoring engine, provider order, freshness rules, fallbacks and Home Assistant entity behaviour unchanged.

## 0.1.4 — 2026-08-13

Dashboard hierarchy correction.

- Reduced the top-level domain view to the five actually scored security domains: Electricity, Gas, Oil reserves, Hydrology and Weather stress.
- Moved nuclear, renewables and other generation evidence underneath Electricity as supporting indicators instead of rendering them as fake scored domains with empty bars.
- Added compact supporting-indicator groups for gas storage, oil reserves, hydrology and weather without inventing 0–100 scores for raw measurements.
- Exposed generation diversity as the genuine normalized sub-score already used by the resilience model.
- Made Diagnostics measurement groups collapsed on first load while preserving groups the user manually opens during dashboard refreshes.

## 0.1.3 — 2026-08-13

Freshness-policy correction.

- Replaced the abrupt electricity freshness cliff with a preferred 90-minute window plus a six-hour hard expiry for Energy-Charts observations. Delayed-but-usable data now keep the electricity score and reduce confidence progressively instead of disappearing.
- Corrected GIE AGSI freshness handling to match its daily reporting semantics: storage-level/capacity evidence remains usable for up to seven days, while daily flow/trend observations have a four-day hard limit.
- Added age-based confidence decay between the preferred freshness window and hard expiry.
- Kept true hard expiry behaviour: observations beyond their allowed window still stop contributing rather than being treated as current.
- Added regression tests covering two-hour electricity data, two-day AGSI storage data, and final hard expiry.

## 0.1.2 — 2026-08-13

Dashboard usability and electricity-reference release.

- Added an Android-style sticky bottom section navigator with Overview, Domains, Signals, Trend and Diagnostics destinations.
- Moved manual refresh into the title bar as an icon action and added a title-bar menu.
- Added dashboard Setup for country, refresh interval, Home Assistant entity publication, weather and optional AGSI/ENTSO-E credentials. Existing secrets are never returned to the browser; blank credential fields preserve them and explicit clear controls remove them.
- Split Diagnostics into Sources and Measurements. Measurements are grouped into collapsible domain/provider sections and mobile diagnostic typography is larger.
- Added an embedded electricity reference library for every supported country profile so fresh generation can still produce a cautious electricity score when live load is unavailable.
- The electricity fallback derives annual-average load from Ember yearly demand data, discloses the derivation and source year, sharply reduces confidence/weight, and cannot trigger an electricity-stress alert by itself.
- Added Ember CC BY 4.0 attribution and reference-data documentation.

## 0.1.1 — 2026-08-13

Reliability and documentation release.

- Fixed Energy-Charts collection around midnight by requesting an explicit multi-day window instead of relying on the API's current-day default.
- Fixed Eurostat oil-stock selection by looking back across recent reporting periods and choosing the latest available country value rather than assuming every Member State has data in the newest global period.
- Added a keyless Eurostat monthly natural-gas stock fallback for EU profiles. It is deliberately exposed as a lower-confidence stock index, not as storage-capacity fill.
- Improved the FGSZ failure path when its public page does not server-render live values, allowing the provider chain to fall through cleanly.
- Fixed Diagnostics showing year-0001 provider timestamps as hundreds of thousands of days ago; a provider with no successful sample now shows `never`.
- Added Home Assistant App Store `icon.png` and `logo.png` assets.
- Expanded app-facing documentation, data-source notes, support matrix and provider troubleshooting.
- Bumped the app version and runtime user agent to 0.1.1.

## 0.1.0 — 2026-08-12

Initial public release. See `energy_security/CHANGELOG.md` for the app-level change list.
