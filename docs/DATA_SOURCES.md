# Data sources

The source policy is: use the simplest reliable public source first, keep credentials optional, preserve source identity on every observation, and fail visibly rather than guessing.

No third-party dataset is bundled in the application. Runtime observations are requested directly from the providers below.

## Electricity — Energy-Charts

Provider ID: `energy_charts`

Energy-Charts is the default electricity provider for supported European country profiles. The collector uses the public-power endpoint without an API key and reads the newest usable sample for:

- total generation;
- load;
- cross-border trading when present;
- renewable share when present;
- nuclear, solar, wind, hydro, gas, coal/lignite, oil and biomass output when present.

Cross-border trading is retained as the provider publishes it; the scoring engine treats a negative value as net imports and positive as net exports.

The parser rejects responses with no timestamp or no recognised measurement. It does not fabricate absent generation types.

## Electricity fallback — ENTSO-E Transparency Platform

Provider ID: `entsoe`

ENTSO-E is optional because machine-to-machine access requires a user token. In 0.1.0 the embedded EIC fallback profile is enabled for Hungary only.

The token is sent to ENTSO-E for the request but is never placed in the observation source URL, cache or dashboard. The public observation URL points to the Transparency Platform without credentials.

Generation and load are requested separately. If generation succeeds and load fails, valid generation observations are retained and the load failure is attached as diagnostic metadata rather than discarding the entire sample.

## Hungary gas — FGSZ

Provider ID: `fgsz_hu`

The Hungary reference profile reads FGSZ's public system summary. It currently extracts only explicitly labelled measurements:

- storage fill percentage;
- domestic production;
- domestic withdrawal;
- domestic consumption;
- domestic injection.

The HTML parser is deliberately strict. If the expected labels disappear after a site redesign, this provider fails and the fallback/cached-data path is used rather than interpreting unrelated numbers.

## Optional gas fallback — GIE AGSI

Provider ID: `gie_agsi`

A user-supplied AGSI key can enable additional gas-storage observations such as stored TWh, working capacity, injection, withdrawal and trend. The key is optional and is sent as an HTTP header, not stored in observation metadata.

The current release does not require AGSI to start or produce electricity/security information.

## Emergency oil stocks — Eurostat

Provider ID: `eurostat_oil`

For EU profiles the app queries Eurostat's emergency oil-stock statistics. This is slow-moving monthly information, not a live reserve gauge.

The dataset contains multiple stock/flow series. The adapter explicitly requests Eurostat stock-flow code `STK_EUE_DIR` (emergency stocks held by the Member State in days equivalent) and, when available, `STK_MIN_CAL` (calculated minimum stock level for compliance). The returned emergency-stock value must also pass a 30–365 day plausibility check. If the expected series is missing or implausible, the oil domain becomes unknown and confidence falls; a different Eurostat series is never silently substituted.

Oil data are monthly and should be read as a strategic-resilience signal, not a live operational reserve gauge.

## Hungary hydrology — HYDROINFO

Provider ID: `hydroinfo_hu`

The Hungary reference profile uses the national HYDROINFO service as an energy-relevant water source. It reads current Danube measurements for Budapest and Paks when the national table provides them, including water level, 24-hour level change, discharge and water temperature. It also recognises explicit qualitative low-water / extreme-low-water / flood language and attempts to read the Lake Balaton level from the English tables.

Paks water temperature is exposed as an observation because cooling-water conditions can matter to thermal generation, but 0.1.0 does not infer a nuclear restriction from temperature alone. Any actual restriction must come from an authoritative plant or system notice in a future adapter. This is not presented as a universal river-security metric; other countries need their own reliable hydrology adapters.

## Weather stress — Open-Meteo

Provider ID: `open_meteo`

When enabled and HOME coordinates are available, a seven-day forecast supplies a bounded weather-stress signal based on temperature, wind and precipitation extremes. Weather does not stand in for generation or fuel supply; it only modifies the seven-day outlook.

## Source hierarchy

For future adapters, prefer:

1. official unauthenticated structured feed;
2. official free API;
3. established independent structured feed;
4. narrowly scoped official HTML parser;
5. local last-known-good observation.

A project-operated proxy or data broker is intentionally out of scope because it would make installed systems depend on project infrastructure.
