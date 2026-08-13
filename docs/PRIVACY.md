# Privacy and network model

Energy Security Monitor is designed to operate without an account or project-operated backend.

## Data that stays local

The following are stored only in Home Assistant app persistent storage (`/data`) or Home Assistant's own app configuration storage:

- last-known-good provider observations;
- seven-day bounded score history;
- provider/source metadata needed for diagnostics;
- app options, including optional provider credentials, managed by Home Assistant Supervisor.

Country profiles, the embedded electricity reference table, parsers, scoring rules and the web UI are part of the installed image. They are not downloaded from GitHub at runtime.

## Local Home Assistant requests

The app may contact Home Assistant's internal Core API to read HOME metadata and publish optional state-machine sensors.

When an administrator explicitly saves **Dashboard → Setup**, the app also calls Home Assistant Supervisor's documented `/addons/self/*` API to replace this app's own options and restart itself. Those self-app endpoints do not require broad Supervisor API access. The Supervisor token supplied by Home Assistant is never returned by the dashboard API.

## External outbound requests

Depending on country and options, the app may contact:

- Energy-Charts;
- ENTSO-E, only when an optional token is configured and the country profile supports it;
- FGSZ for the Hungary gas profile;
- GIE AGSI when an optional key is configured;
- Eurostat for EU oil/gas-stock information;
- HYDROINFO for the Hungary water profile;
- Open-Meteo when weather stress is enabled and HOME coordinates are available.

The embedded Ember electricity reference is release data and causes no runtime request to Ember.

The app does not contact ArrowSK, GitHub, an analytics provider, an advertising network or a licence server during normal runtime.

## Location

With `country: auto`, the app reads HOME country, latitude, longitude, timezone and location name from Home Assistant. The national providers receive a country code where needed. Open-Meteo receives HOME latitude/longitude for local weather if weather stress is enabled.

The project does not receive those coordinates.

## Optional credentials

`agsi_key` and `entsoe_token` are Home Assistant app options. They are consumed locally by the process and sent only to the relevant provider.

The dashboard Setup API never returns their stored values. It exposes only `configured: true/false`; a blank submitted credential preserves the existing secret and a separate clear option removes it. The ENTSO-E request token is stripped from the source URL written to observations. AGSI sends its key in a header. Credentials are not copied into the score cache.

Home Assistant itself stores app options; protect Home Assistant backups and administrator access accordingly.

## Telemetry

There is no telemetry. Provider-health diagnostics remain inside the app. The repository does not receive automatic failure reports or usage statistics.
