# Energy Security Monitor

Energy Security Monitor can be installed in **Home Assistant**, run with **Docker / Docker Compose**, or deployed on **Railway**. All installation options use the same Go core, providers, scoring rules, fallback logic and dashboard.

This directory contains the **Home Assistant package**. It is the most integrated option if you already use Home Assistant, but it is not a separate edition of the project and it is not the only way to run Energy Security Monitor.

## Why use the Home Assistant package?

- opens directly from the Home Assistant sidebar through Ingress;
- can use the HOME country and coordinates automatically with `country: auto`;
- can publish selected energy-security states as Home Assistant sensors;
- lets you change normal options from the dashboard Setup screen;
- requires no project account, MQTT broker or mandatory API key for normal use.

The same core can also run completely standalone on a NAS, server, VPS or hosted Railway service. A provider, scoring, freshness or dashboard fix made in the shared core therefore applies to every deployment mode rather than being reimplemented separately.

## What the monitor does

Energy Security Monitor turns public electricity, fuel, hydrology and weather data into a cautious national energy-security assessment. It keeps the underlying measurements visible, shows a separate confidence value, and treats missing or delayed data as uncertainty rather than silently converting them into zero.

Version 0.2.0 includes:

- keyless European electricity collection through Energy-Charts;
- optional ENTSO-E fallback where a country profile supports it;
- an embedded country annual-demand/population reference used at reduced confidence when fresh generation exists but live load is missing;
- Hungary gas through FGSZ when its live values are available server-side, with keyless Eurostat monthly gas-stock fallback and optional GIE AGSI enhancement;
- EU emergency-oil stock evidence from Eurostat;
- Hungary Danube/hydrology through HYDROINFO;
- seven-day local weather stress through Open-Meteo;
- deterministic Current, 7-day Outlook and Strategic Resilience scores with a separate confidence value;
- local provider health, fallback, circuit-breaker and last-known-good caching;
- native Docker health checking;
- a mobile-friendly dashboard with five top-level scored security domains, collapsed supporting indicators, separated Sources/Measurements diagnostics and optional HA state sensors;
- standalone Docker/Railway runtime using environment configuration and normal HTTP serving;
- a repository-root Dockerfile whose compiled version is read directly from this Home Assistant app's `config.yaml`, with CI verifying that every deployment stays version-bound to the same core release.

The dashboard deliberately distinguishes scored domains from supporting observations. Electricity generation sources such as nuclear, solar, wind, hydro and thermal generation appear inside an expandable Electricity detail section rather than as independent security scores. Generation components show MW plus their percentage of current load when live load exists; when it does not, the UI explicitly falls back to percentage of current generation.

Home Assistant-specific behaviour remains isolated to the thin package layer: Ingress, Supervisor-backed Setup, `/data/options.json` configuration and optional HA entity publication. Standalone mode does not emulate those pieces; it simply runs the same binary and shared core with environment-based configuration.

For Home Assistant installation, configuration, interpretation, diagnostics and troubleshooting, continue with [DOCS.md](DOCS.md).

If you would rather run the project without Home Assistant, use the [Standalone Docker and Railway guide](../docs/STANDALONE.md).

For the project overview, scoring model, data sources and architecture, see the repository-level [README](../README.md).
