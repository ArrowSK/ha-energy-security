# Privacy and network model

Energy Security Monitor is designed to operate without an account or project-operated backend. Home Assistant and standalone Docker/Railway deployments use the same data-collection and scoring core but have different local configuration boundaries.

## Data that stays local

The following are stored only in the deployment's persistent `/data` directory or its platform configuration store:

- last-known-good provider observations;
- seven-day bounded score history;
- provider/source metadata needed for diagnostics;
- Home Assistant app options when running under Supervisor;
- standalone environment variables when configured by Docker/Railway. These are supplied by the deployment platform and are not copied into the score cache.

Country profiles, the embedded electricity reference table, parsers, scoring rules and the web UI are part of the installed image. They are not downloaded from GitHub at runtime.

## Home Assistant requests

Home Assistant mode may contact Home Assistant's internal Core API to read HOME metadata and publish optional state-machine sensors.

When an administrator explicitly saves **Dashboard → Setup**, the Home Assistant app also calls Home Assistant Supervisor's documented `/addons/self/*` API to replace this app's own options and restart itself. The Supervisor token supplied by Home Assistant is never returned by the dashboard API.

Standalone mode does not use the Home Assistant Core API or Supervisor and always disables Home Assistant entity publication.

## Standalone HTTP boundary

Home Assistant mode keeps the existing ingress-only request guard. Standalone mode removes only that Home Assistant-specific IP restriction so a Docker host or Railway proxy can serve the embedded dashboard normally. The same security headers remain enabled.

The standalone dashboard does not add user authentication. If a standalone deployment should be private, keep it on a private network or place it behind an authenticated reverse proxy.

`GET /api/v1/config` reports configuration state but never returns provider credential values. Standalone configuration writes are rejected because Docker/Railway environment variables are the source of truth.

## External outbound requests

Depending on country and options, the application may contact:

- Energy-Charts;
- ENTSO-E, only when an optional token is configured and the country profile supports it;
- FGSZ for the Hungary gas profile;
- GIE AGSI when an optional key is configured;
- Eurostat for EU oil/gas-stock information;
- HYDROINFO for the Hungary water profile;
- Open-Meteo when weather stress is enabled and valid coordinates are available.

The embedded Ember electricity reference is release data and causes no runtime request to Ember.

The application does not contact ArrowSK, GitHub, an analytics provider, an advertising network or a licence server during normal runtime.

## Location

In Home Assistant mode, `country: auto` reads HOME country, latitude, longitude, timezone and location name from Home Assistant. Open-Meteo receives HOME latitude/longitude when weather stress is enabled.

In standalone mode, `ENERGY_SECURITY_COUNTRY` is explicit and `auto` is rejected. Optional `ENERGY_SECURITY_LATITUDE` and `ENERGY_SECURITY_LONGITUDE` are sent to Open-Meteo only when weather is enabled. The project does not receive those coordinates.

## Optional credentials

Home Assistant mode stores `agsi_key` and `entsoe_token` as app options. Standalone mode reads the equivalent `ENERGY_SECURITY_AGSI_KEY` and `ENERGY_SECURITY_ENTSOE_TOKEN` environment variables.

Credentials are consumed locally by the process and sent only to the relevant provider. The dashboard configuration API never returns their stored values; it exposes only whether a credential is configured. The ENTSO-E request token is stripped from the source URL written to observations. AGSI sends its key in a header. Credentials are not copied into the score cache.

Protect Home Assistant backups, Docker environment files and Railway project access according to the deployment mode in use. `.env` files are ignored by this repository; `.env.example` contains no real secret.

## Telemetry

There is no telemetry. Provider-health diagnostics remain inside the deployment. The repository does not receive automatic failure reports or usage statistics.
