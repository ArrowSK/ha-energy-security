# Third-party software and data

## Runtime software dependencies

The Go module intentionally has no third-party Go module dependencies. The container uses Home Assistant's base image and Alpine packages according to their respective licences. Build tooling uses the official Go toolchain.

## Embedded reference data

Energy Security Monitor embeds a small country electricity-reference table derived from Ember's yearly electricity-demand data. Ember publishes its data under **CC BY 4.0**. The table is used only when a live electricity source provides generation but not live load; the UI and scoring description disclose that the load is derived rather than live.

Attribution: **Yearly electricity demand data, Ember**.

The embedded fields are source year, annual demand, per-capita annual demand and a population value derived from those two fields. See `docs/ELECTRICITY_REFERENCE.md` for the derivation and scoring safeguards.

The CC BY 4.0 terms apply to the Ember-derived data independently of this repository's PolyForm Noncommercial software licence. The project does not impose additional restrictions on the Ember data itself.

## Runtime external data providers

Other current observations are requested directly from configured public providers at runtime. Their data, websites, APIs and trademarks remain governed by their own terms.

Energy-Charts states that its API data are CC BY 4.0 unless otherwise noted and requires attribution to Energy-Charts.info. The app preserves the source name and source URL with observations.

Eurostat and other government/public operators are referenced as data sources, not software dependencies. Users and redistributors remain responsible for complying with provider terms applicable to their use.

Open-Meteo, GIE, ENTSO-E, FGSZ, HYDROINFO, Energy-Charts and Ember are not affiliated with this project.
