# Changelog

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
