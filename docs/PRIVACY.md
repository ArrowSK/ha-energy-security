# Privacy and network model

Energy Security Monitor is designed to operate without an account or project-operated backend.

## Data that stays local

The following are stored only in Home Assistant app persistent storage (`/data`):

- last-known-good provider observations;
- seven-day bounded score history;
- provider/source metadata needed for diagnostics.

Country profiles, parsers, scoring rules and the web UI are part of the installed image. They are not downloaded from GitHub at runtime.

## Outbound requests

Depending on country and options, the app may contact:

- Home Assistant's internal Core API;
- Energy-Charts;
- ENTSO-E, only when an optional token is configured and the country profile supports it;
- FGSZ for the Hungary gas profile;
- GIE AGSI when an optional key is configured;
- Eurostat for EU oil-stock information;
- HYDROINFO for the Hungary water profile;
- Open-Meteo when weather stress is enabled and HOME coordinates are available.

The app does not contact ArrowSK, GitHub, an analytics provider, an advertising network or a licence server during normal runtime.

## Location

With `country: auto`, the app reads HOME country, latitude, longitude, timezone and location name from Home Assistant. The national providers receive a country code where needed. Open-Meteo receives HOME latitude/longitude for local weather if weather stress is enabled.

The project does not receive those coordinates.

## Optional credentials

`agsi_key` and `entsoe_token` are Home Assistant app options. They are consumed locally by the process and sent only to the relevant provider.

The ENTSO-E request token is deliberately stripped from the source URL written to observations. AGSI sends its key in a header. Credentials are not returned by the dashboard API or copied into the score cache.

Home Assistant itself stores app options; protect Home Assistant backups and administrator access accordingly.

## Telemetry

There is no telemetry. Provider-health diagnostics remain inside the app. The repository does not receive automatic failure reports or usage statistics.
