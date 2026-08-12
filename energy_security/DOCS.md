# Energy Security Monitor — app documentation

## Start here

The default configuration is intentional. Install the app, start it, and open **Energy Security** from the Home Assistant sidebar.

`country: auto` reads the country configured for Home Assistant's HOME location. No account, project server, telemetry service, MQTT broker or API key is required.

## Configuration

### `country`

Default: `auto`

`auto` uses Home Assistant's configured country. To monitor another country, enter its ISO 3166-1 alpha-2 code, for example `HU`, `DE`, `FR` or `GB`.

A manual country override changes national data sources only. If Home Assistant location information is available, local weather stress still uses the HOME latitude and longitude.

### `refresh_minutes`

Default: `30`

Allowed range: 10–180 minutes. Data sources have their own freshness limits, so reducing this setting does not make slow-moving oil or storage statistics real-time.

### `enable_ha_entities`

Default: `true`

Publishes a compact set of states such as `sensor.energy_security_score`, `sensor.energy_security_confidence`, domain scores and selected live measurements. These are state-machine entities created by the app; they are republished after the app starts and are not a separate Home Assistant integration.

### `enable_weather`

Default: `true`

Adds a seven-day heat, cold, wind and precipitation stress signal using the HOME coordinates.

### `agsi_key`

Default: empty.

Optional GIE AGSI key. It is not required for the app to operate. If supplied, it can provide a higher-quality gas-storage fallback in countries covered by GIE.

### `entsoe_token`

Default: empty.

Optional ENTSO-E Transparency Platform token. It is used only where the bundled country profile explicitly supports the adapter. The default electricity source does not require a key.


## How to read the dashboard

The headline is not a claim that every risk is known. It is displayed together with **confidence**. Missing or stale data reduce confidence and are never silently converted to zero.

Three horizons are shown:

- **Now** — present operational conditions, primarily electricity, gas, hydrology and current stress signals.
- **Outlook** — current conditions combined with the seven-day weather stress outlook.
- **Resilience** — slower-moving structural signals such as generation diversity, gas storage, oil data and hydrology where available.

Open **Diagnostics** to see provider health, source freshness, fallback state and collection errors.

## Self-healing behaviour

A provider failure does not erase a last-known-good observation. The app retries sources on later cycles, switches to the next configured provider where available, opens a temporary circuit after repeated failures, and automatically retries the preferred provider after the cooldown. Cache writes are atomic and stored under `/data`.

If the internet is unavailable, cached observations remain visible until their individual freshness limits expire. Once stale, they stop contributing to the score.

## Privacy

There is no telemetry and no runtime call to the project repository. See `docs/PRIVACY.md` in the repository for the exact network model.
