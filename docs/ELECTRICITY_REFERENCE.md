# Embedded electricity reference library

## Purpose

Some electricity feeds publish current generation but omit current load/consumption. Returning no electricity score in that case throws away useful live supply evidence, while pretending that an annual statistic is the current load would be misleading.

Energy Security Monitor therefore ships a small, versioned reference table for every country profile currently embedded in the app. It is a fallback of last resort for electricity scoring when a fresh live generation value exists but live load does not.

Live load always takes precedence.

## Data and attribution

The reference values are derived from Ember's country overview yearly electricity-demand data, published under CC BY 4.0. The embedded rows contain:

- country code;
- source year;
- population in millions;
- annual electricity demand in TWh;
- annual electricity demand per capita in MWh/person.

Most current profiles use 2024. Ukraine uses the latest row available in the reference dataset used for this release, 2022. The app always includes the reference year in the electricity-domain description so an older reference cannot silently look current.

Population is derived from the same Ember row as `annual demand TWh / annual demand MWh per capita`, which yields population in millions. The stored value is rounded to three decimals. It is reference metadata for transparency; it is not a demographic forecast.

Ember attribution: **Yearly electricity demand data, Ember**.

## Derived load

When live load is absent, the scoring engine computes:

`reference average load MW = annual demand TWh × 1,000,000 / 8,760`

This is the arithmetic annual-average demand. It is **not** the current load, peak load, seasonal norm, or a demand forecast. It does not attempt to infer time-of-day, weather, holidays, industrial cycles, or demand response.

The dashboard/domain description explicitly says that live consumption is unavailable and lists the reference year, annual demand, approximate population and per-capita demand.

## Scoring safeguards

An estimated reference is deliberately weaker than a live load observation:

- electricity-domain confidence is reduced to 45% of the live generation source quality;
- electricity weight in the Current composite is halved from 0.50 to 0.25;
- the derived reference cannot by itself trigger the electricity-stress alert;
- the moment a fresh live load observation is available, the reference path is bypassed.

The objective is continuity with visible uncertainty, not manufactured precision.

## Updating the library

Reference rows are code-reviewed release data, not downloaded at runtime. Updating them requires a normal application release and tests. This preserves the project's no-runtime-project-backend design and makes every fallback value inspectable in source control.
