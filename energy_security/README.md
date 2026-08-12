# Energy Security Monitor

Energy Security Monitor is a self-contained Home Assistant app that assesses national energy-security conditions from public electricity, fuel, hydrology and weather data.

Normal use is intentionally simple: install the app, start it, and open **Energy Security** from the Home Assistant sidebar. With the default `country: auto`, the app uses the country configured for Home Assistant HOME. No project account, MQTT broker or mandatory API key is required.

The app keeps provider selection, fallback logic, country profiles, scoring and dashboard assets locally. It does not download runtime configuration or parser code from this repository and does not send telemetry to the project.

Version 0.1.1 includes:

- keyless European electricity collection through Energy-Charts;
- optional ENTSO-E fallback where a country profile supports it;
- Hungary gas through FGSZ when its live values are available server-side, with keyless Eurostat monthly gas-stock fallback and optional GIE AGSI enhancement;
- EU emergency-oil stock evidence from Eurostat;
- Hungary Danube/hydrology through HYDROINFO;
- seven-day local weather stress through Open-Meteo;
- deterministic Current, 7-day Outlook and Strategic Resilience scores with a separate confidence value;
- local provider health, fallback, circuit-breaker and last-known-good caching;
- native Docker health checking;
- Home Assistant Ingress dashboard and optional HA state sensors.

See [DOCS.md](DOCS.md) for installation, configuration, interpretation, diagnostics and troubleshooting. The repository-level [README](../README.md) contains architecture, source and development documentation.
