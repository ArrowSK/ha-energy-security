# Data sources

The source policy is: use the simplest reliable public source first, keep credentials optional, preserve source identity on every observation, and fail visibly rather than guessing.

Most current observations are requested directly from the providers below. Country profiles, parser/scoring logic and one small electricity-reference table are bundled locally with the installed release. The embedded reference table is documented separately and is never presented as live data.

## Electricity — Energy-Charts

Provider ID: `energy_charts`

Energy-Charts is the default keyless electricity provider for supported European profiles. The collector requests the public-power endpoint with an explicit recent multi-day window and selects the newest usable sample. The explicit window avoids relying on the API's current-local-day default, which can legitimately have no published sample immediately after day rollover.

The adapter extracts, when present:

- total generation;
- load;
- cross-border trading;
- renewable share;
- nuclear, solar, wind, hydro, gas, coal/lignite, oil and biomass output;
- generation-mix attributes used for the structural diversity calculation.

Cross-border trading is retained as published; the scoring engine treats negative values as net imports and positive values as net exports. The parser rejects responses with no timestamp or no recognised measurement and never fabricates absent generation types.

## Embedded electricity reference when live load is missing

This is not a network provider. It is a release-time fallback table derived from Ember's yearly electricity-demand data (CC BY 4.0) and covers every country profile embedded in the app.

It is used only when:

1. a fresh live generation observation exists; and
2. no fresh live electricity load exists; and
3. the selected country has an embedded reference row.

The table stores source year, annual demand, per-capita demand and an approximate population derived from those two values. The scorer converts annual demand into an arithmetic annual-average load using `TWh × 1,000,000 / 8,760`.

That figure is **not current load, peak load, seasonal normal load or a forecast**. The electricity-domain description explicitly says that live consumption is unavailable and prints the reference year and basis. Confidence is reduced to 45% of the live generation quality, electricity's Current-composite weight is halved, and the estimated path cannot create the electricity-stress alert by itself. Fresh live load always overrides the reference.

Most rows in version 0.1.2 use 2024 Ember demand data. Ukraine uses 2022, the latest row available in the source snapshot used for this release; the year is therefore visible in the resulting description rather than silently treated as current.

See [Embedded electricity reference library](ELECTRICITY_REFERENCE.md) and `THIRD_PARTY_LICENSES.md`.

## Electricity fallback — ENTSO-E Transparency Platform

Provider ID: `entsoe`

ENTSO-E is optional because machine-to-machine access requires a user token. In the current embedded profiles, token-based EIC fallback is configured for Hungary. The adapter requests generation and load independently so one valid half of the response is not lost merely because the other call fails.

The token is sent only to ENTSO-E. It is not written into observation source URLs, cache metadata or dashboard diagnostics.

## Hungary gas — FGSZ

Provider ID: `fgsz_hu`

The Hungary reference profile first checks FGSZ's public system summary for explicitly labelled measurements:

- storage fill percentage;
- domestic production;
- storage withdrawal;
- domestic consumption;
- storage injection.

FGSZ has at times delivered the labels in server HTML while inserting live numerical values client-side. Energy Security Monitor intentionally does not embed Chromium or execute arbitrary page JavaScript. If the received HTML contains no trustworthy live values, the provider reports a safe failure and the gas provider chain continues.

This design is deliberate: a lightweight failed primary with a working fallback is preferable to an opaque browser scraper that raises memory use and maintenance risk.

## Optional gas source — GIE AGSI

Provider ID: `gie_agsi`

A user-supplied AGSI key can provide higher-quality storage observations such as:

- storage fill percentage;
- gas in storage (TWh);
- working gas capacity;
- injection and withdrawal;
- storage versus annual consumption;
- daily storage trend.

The key is sent in an HTTP header and is not copied into observation URLs or dashboard metadata. AGSI is optional; the app remains functional without it.

## Keyless EU gas fallback — Eurostat monthly gas stocks

Provider ID: `eurostat_gas`

For Eurostat-enabled profiles, version 0.1.1 added a credential-free strategic fallback based on dataset `nrg_stk_gasm` with these explicit filters:

- `siec=G3000` — natural gas;
- `stk_flow=STKCL_NAT` — closing stock on national territory;
- `unit=TJ_GCV` — terajoules, gross calorific value;
- latest 36 reporting periods requested.

The adapter publishes the latest positive monthly closing stock as `gas_national_stock_twh` and computes `gas_stock_index_pct` against the maximum value in the returned 36-month window.

The index is **not** physical storage capacity fill. It is a relative stock-position proxy. The observation carries `not_capacity_fill=true`, receives lower source quality than FGSZ/GIE, and the scoring engine explicitly states the proxy basis. Actual `gas_storage_fill_pct` remains preferred whenever fresh.

The longer TTL reflects Eurostat's monthly reporting frequency and publication lag. Refreshing the app every 30 minutes does not make this statistic real-time.

## Emergency oil stocks — Eurostat

Provider ID: `eurostat_oil`

For EU profiles the app queries dataset `nrg_stk_oem` for:

- `STK_EUE_DIR` — emergency stocks held by the Member State in days equivalent;
- `STK_MIN_CAL` — calculated minimum stock level for compliance, when available;
- `unit=NR`.

Version 0.1.1 requests the latest 12 dataset periods and then selects the newest period that actually contains the requested series for the selected country. This matters because countries can report with different lags; the newest global period can exist while one country's cell is still empty.

The emergency-stock value must pass a 30–365 day plausibility check. The adapter never substitutes another oil series simply because the expected days-equivalent series is absent.

Oil evidence is monthly and is used as a strategic-resilience input, not a live operational gauge.

## Hungary hydrology — HYDROINFO

Provider ID: `hydroinfo_hu`

The Hungary profile uses the national HYDROINFO service as an energy-relevant water source. It reads current Danube measurements for Budapest and Paks when published, including water level, 24-hour level change, discharge and water temperature. It also recognises explicit qualitative low-water/extreme-low-water/flood language and attempts to read Lake Balaton level from the English tables.

Paks water temperature is exposed because cooling-water conditions can matter to thermal generation, but the app does not infer a nuclear restriction from temperature alone. An actual restriction requires an authoritative plant/system notice or a dedicated future adapter.

## Weather stress — Open-Meteo

Provider ID: `open_meteo`

When enabled and HOME coordinates are available, a seven-day forecast supplies bounded temperature, wind and precipitation stress. Weather is only a stress modifier; it does not stand in for generation, storage or fuel supply.

## Source hierarchy

For new adapters, prefer:

1. official unauthenticated structured feed;
2. official free API;
3. established independent structured feed;
4. narrowly scoped official HTML parser;
5. an explicitly labelled, versioned local reference only where its semantics remain defensible;
6. local last-known-good observation.

A project-operated proxy, telemetry backend or remote parser/configuration service is intentionally out of scope because it would make installed systems depend on project infrastructure.

## Freshness and semantic compatibility

Fallback values are not interchangeable merely because they concern the same topic. If a fallback represents a different concept, it must use a different key or visibly reduced confidence rather than masquerading as the primary metric. The gas stock index and the derived electricity annual-average load are examples of this rule.

Each runtime observation retains source, source URL, observation time, retrieval time, quality, TTL and stale state. Stale observations remain visible for diagnosis but are excluded from scoring. Embedded electricity reference values are static release metadata and are identified as such in the domain description rather than emitted as live observations.
