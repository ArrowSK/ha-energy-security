# Changelog

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
- Fixed Eurostat emergency-oil selection by looking back across recent reporting periods and choosing the latest available country value rather than assuming every Member State has data in the newest global period.
- Added a keyless Eurostat monthly natural-gas stock fallback for EU profiles. It is deliberately exposed as a lower-confidence stock index, not as storage-capacity fill.
- Improved the FGSZ failure path when its public page does not server-render live values, allowing the provider chain to fall through cleanly.
- Fixed Diagnostics showing year-0001 provider timestamps as hundreds of thousands of days ago; a provider with no successful sample now shows `never`.
- Added Home Assistant App Store `icon.png` and `logo.png` assets.
- Expanded app-facing documentation, data-source notes, support matrix and provider troubleshooting.
- Bumped the app version and runtime user agent to 0.1.1.

## 0.1.0 — 2026-08-12

Initial public release. See `energy_security/CHANGELOG.md` for the app-level change list.
