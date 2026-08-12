# Scoring model

Energy Security Monitor uses deterministic scoring. The score is a summary of available evidence, not a prediction and not a government/TSO adequacy declaration.

## Four outputs, not one

The app reports:

- **Current** — operational evidence available now;
- **7-day outlook** — current conditions adjusted for forecast weather stress;
- **Strategic resilience** — slower-moving diversity, fuel reserve and water signals;
- **Headline** — a weighted combination of the three horizons.

Every score is paired with a separate **confidence** percentage.

The headline weights in 0.1.0 are:

- current: 55%;
- seven-day: 25%;
- strategic: 20%.

These values are versioned implementation choices and may be refined as more sector-specific sources are added.

## Missing data

Missing and stale data are excluded from score arithmetic. They are not converted to zero.

The weighted-score helper tracks the amount of expected evidence that was actually present. Confidence is multiplied by that coverage. Therefore:

- a country can still have a reasonable provisional score when a domain is unavailable;
- its confidence falls because the evidence base is incomplete;
- a country with sparse sources does not look artificially dangerous merely because it publishes less data.

## Freshness and quality

A usable observation must be inside its TTL. Each provider also supplies a quality estimate between 0 and 1. Domain confidence is derived from quality and evidence coverage.

Cached observations are allowed to bridge short provider outages only while still fresh. Once their TTL expires they remain visible in Diagnostics but stop contributing.

## Current score

The current score is built from available operational domains. The default expected weights are:

| Domain | Expected weight |
|---|---:|
| Electricity | 50% |
| Gas | 30% |
| Water | 10% |
| Weather | 10% |

The app does not force a missing gas/water/weather domain into the score; the missing expected weight reduces confidence.

### Electricity

When both generation and load are available, generation relative to load is the main short-term adequacy proxy. Net electricity imports can close part of an observed generation shortfall; net exports do not count as extra domestic reserve.

This is intentionally cautious: public generation/load data are not the same as certified available capacity or operating reserve. The current electricity score is therefore capped and described as an operational proxy, not an adequacy guarantee.

### Gas

Gas storage fill is compared with a month-dependent seasonal target. Storage is a resilience signal, not a complete supply model; pipeline/LNG flow and demand coverage should become additional evidence as adapters mature.

### Water

In Hungary 0.1.0 uses the HYDROINFO qualitative hydrological proxy. It affects the score modestly because it is not yet a direct national generation-capacity measurement.

### Weather

Forecast heat, cold, wind and precipitation extremes can lower the seven-day stress component. Ordinary weather is neutral-to-positive; it cannot compensate for missing physical supply data.

## Strategic score

Strategic resilience currently combines evidence from:

- electricity generation diversity;
- gas storage;
- plausible emergency-oil days where available;
- hydrological state.

Generation diversity is based on the latest source mix using a Herfindahl-Hirschman-style concentration measure. It is a structural resilience signal only: a diverse system can still be short of supply, and a concentrated system can be operationally secure.

## Oil safeguards

The Eurostat oil dataset is deliberately gated. A parsed days-equivalent value below 30 or above 365 is treated as data-limited rather than scored. This prevents an ambiguous dimension/series from generating a false emergency signal.

## Status labels

Numeric scores are mapped to plain-language bands for display. Confidence stays separate; a green-looking provisional score with low confidence is visibly marked as such.

## What is intentionally not scored

0.1.0 excludes:

- news sentiment;
- social-media signals;
- LLM interpretation;
- unsourced geopolitical judgements;
- consumer energy prices as a direct security score;
- carbon intensity as a security score.

Those signals either answer a different question or are too difficult to make reproducible without substantial false-positive risk. Formal official emergency notices may be added later as explicit inputs.
