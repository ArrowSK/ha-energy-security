# Scoring model

Energy Security Monitor uses deterministic scoring. The score is a summary of available evidence, not a prediction and not a government/TSO adequacy declaration.

## Four outputs, not one

The app reports:

- **Current** — operational evidence available now;
- **7-day outlook** — current conditions adjusted for forecast weather stress;
- **Strategic resilience** — slower-moving diversity, fuel reserve and water signals;
- **Headline** — a weighted combination of the three horizons.

Every score is paired with a separate **confidence** percentage.

The headline weights are:

- current: 55%;
- seven-day: 25%;
- strategic: 20%.

These values are versioned implementation choices and may be refined as more sector-specific sources are added.

## Missing data

Missing and stale data are excluded from score arithmetic. They are not converted to zero.

The weighted-score helper tracks the amount of expected evidence that was actually present. Confidence is multiplied by that coverage. Therefore a country can still have a provisional score when a domain is unavailable, while confidence falls because the evidence base is incomplete.

Version 0.1.2 adds one explicitly bounded exception to “no derivation”: if fresh electricity generation exists but live load is absent, an embedded annual-demand reference for the selected country can provide a low-confidence annual-average load. It is not inserted as a live observation and its derivation is disclosed in the electricity description.

## Freshness and quality

A usable runtime observation must be inside its TTL. Each provider also supplies a quality estimate between 0 and 1. Domain confidence is derived from quality and evidence coverage.

Cached observations are allowed to bridge short provider outages only while still fresh. Once their TTL expires they remain visible in Diagnostics but stop contributing.

## Current score

The normal expected operational weights are:

| Domain | Expected weight |
|---|---:|
| Electricity | 50% |
| Gas | 30% |
| Water | 10% |
| Weather | 10% |

A missing gas/water/weather domain is not forced into the score; the missing expected weight reduces confidence.

### Electricity with live load

When both generation and load are available, generation relative to load is the main short-term adequacy proxy. Net electricity imports can close part of an observed generation shortfall; net exports do not count as extra domestic reserve.

This remains an operational proxy rather than certified available capacity or operating reserve.

### Electricity with derived reference load

If live load is unavailable but fresh generation exists, the scorer can look up the selected country in the embedded Ember-derived electricity reference library. Annual demand is converted to annual-average MW:

`annual demand TWh × 1,000,000 / 8,760`

This reference is not current demand, peak demand, seasonal demand or a forecast. The resulting electricity-domain description explicitly states that live consumption is unavailable and gives the source year, annual demand, approximate population and per-capita annual demand.

To prevent false precision:

- electricity-domain confidence becomes `generation quality × 45%`;
- electricity weight in the Current composite is reduced from 50% to 25%;
- the estimated path cannot emit the live electricity-stress alert by itself;
- fresh live load immediately bypasses the reference.

See `ELECTRICITY_REFERENCE.md` for source and update methodology.

### Gas

Fresh physical gas storage fill is compared with a month-dependent seasonal target. If only the Eurostat monthly national-stock proxy is available, that signal remains strategic/data-limited and is excluded from the Current horizon when its confidence is below 40%.

### Water

The Hungary HYDROINFO qualitative hydrological proxy affects the score modestly because it is not a direct national generation-capacity measurement.

### Weather

Forecast heat, cold, wind and precipitation extremes can lower the seven-day stress component. Ordinary weather is neutral-to-positive; it cannot compensate for missing physical supply data.

## Strategic score

Strategic resilience currently combines evidence from:

- electricity generation diversity;
- gas storage/stock evidence;
- plausible emergency-oil days where available;
- hydrological state.

Generation diversity is based on the latest source mix using a Herfindahl-Hirschman-style concentration measure. It is a structural resilience signal only: a diverse system can still be short of supply, and a concentrated system can be operationally secure.

## Oil safeguards

The Eurostat oil dataset is deliberately gated. A parsed days-equivalent value below 30 or above 365 is treated as data-limited rather than scored. This prevents an ambiguous dimension/series from generating a false emergency signal.

## Status labels

Numeric scores are mapped to plain-language bands for display. Confidence stays separate; a green-looking provisional score with low confidence is visibly marked as such.

## What is intentionally not scored

The app excludes:

- news sentiment;
- social-media signals;
- LLM interpretation;
- unsourced geopolitical judgements;
- consumer energy prices as a direct security score;
- carbon intensity as a security score.

Those signals either answer a different question or are too difficult to make reproducible without substantial false-positive risk. Formal official emergency notices may be added later as explicit inputs.
