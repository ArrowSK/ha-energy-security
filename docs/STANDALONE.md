# Standalone Docker and Railway deployment

Energy Security Monitor can run either as a Home Assistant app or as a standalone web service. Both modes execute the same Go binary and the same `internal/` provider, cache, scoring and dashboard code. There is no second implementation to keep in sync.

## Version coupling

The Home Assistant app manifest remains the release source of truth:

`energy_security/config.yaml`

The root standalone `Dockerfile` reads that manifest during the build and compiles the exact same version into the standalone binary. CI then starts the image with `--version` and requires it to match the Home Assistant manifest. A future core fix therefore lands once in `energy_security/`; the Home Assistant image and standalone image both build from that code, and a normal Home Assistant version bump also becomes the standalone image version automatically.

The Home Assistant-specific container remains `energy_security/Dockerfile`. The standalone container is the repository-root `Dockerfile`. Keeping those thin deployment wrappers separate avoids introducing Railway/Docker assumptions into the working Home Assistant runtime.

## Configuration

Standalone mode uses environment variables because there is no Home Assistant Supervisor to persist app options. Home Assistant mode continues to use `/data/options.json` and the existing dashboard Setup flow.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `ENERGY_SECURITY_COUNTRY` | yes | none | Two-letter ISO 3166-1 country code, for example `HU`, `DE` or `FR`. `auto` is intentionally rejected outside Home Assistant. |
| `ENERGY_SECURITY_REFRESH_MINUTES` | no | `30` | Assessment interval, bounded to 10–180 minutes. |
| `ENERGY_SECURITY_ENABLE_WEATHER` | no | `true` | Enables weather collection. |
| `ENERGY_SECURITY_LATITUDE` | no | none | Latitude for local weather. Must be supplied together with longitude. |
| `ENERGY_SECURITY_LONGITUDE` | no | none | Longitude for local weather. Must be supplied together with latitude. |
| `ENERGY_SECURITY_TIMEZONE` | no | none | Display/location timezone, for example `Europe/Budapest`. |
| `ENERGY_SECURITY_LOCATION_NAME` | no | none | Friendly location name shown in the dashboard. |
| `ENERGY_SECURITY_AGSI_KEY` | no | none | Optional GIE AGSI credential. |
| `ENERGY_SECURITY_ENTSOE_TOKEN` | no | none | Optional ENTSO-E credential. |
| `ENERGY_SECURITY_DATA_DIR` | no | `/data` | Persistent cache/history directory. |
| `PORT` | platform | `8080` in standalone mode | HTTP listening port. Railway injects this automatically. |

Standalone mode always disables Home Assistant entity publication. It does not attempt to contact the Home Assistant Supervisor. If coordinates are omitted, country-level electricity/fuel data can still work, but local weather does not have a defensible location and may remain unavailable.

## Docker Compose

1. Copy the example environment file:

   ```sh
   cp .env.example .env
   ```

2. Edit `.env`, especially `ENERGY_SECURITY_COUNTRY` and the optional location values.

3. Build and start:

   ```sh
   docker compose up -d --build
   ```

4. Open `http://localhost:8099` unless `ENERGY_SECURITY_PORT` was changed.

The Compose definition uses a named volume at `/data`, so cached observations and seven-day score history survive container replacement.

To stop the service without deleting its data:

```sh
docker compose down
```

## Plain Docker

Build from the repository root:

```sh
docker build -t energy-security .
```

Run it:

```sh
docker run -d \
  --name energy-security \
  -p 8099:8080 \
  -e ENERGY_SECURITY_COUNTRY=HU \
  -e ENERGY_SECURITY_ENABLE_WEATHER=false \
  -v energy-security-data:/data \
  energy-security
```

The image's built-in health check uses `/healthz`.

## Railway

The repository includes `railway.toml`. Railway is configured to use the root `Dockerfile`, check `/healthz`, and restart the service on failure.

For a GitHub deployment:

1. Create a Railway service from this repository and deploy the `main` branch.
2. Add `ENERGY_SECURITY_COUNTRY` in the service Variables. Add latitude/longitude/timezone/location name if weather should be available. Optional provider credentials belong in Railway Variables, not in the repository.
3. Enable Public Networking to expose the dashboard.
4. Optional but recommended: attach a Railway volume mounted at `/data` so cached state and history survive deployments.

No custom start command or fixed port is required. The standalone binary binds to `0.0.0.0:$PORT`, using `8080` only when `PORT` is not supplied.

### Railway persistence

The service works without a volume, but Railway's filesystem is not a substitute for durable application data. Without a `/data` volume, a replacement deployment starts with an empty cache/history and recollects public data. With a volume, the same local state model used by the Home Assistant app is retained.

## Dashboard Setup in standalone mode

`GET /api/v1/config` still reports the active configuration without returning credential values. Configuration mutation is deliberately rejected in standalone mode because Docker/Railway environment variables are the source of truth and cannot be safely rewritten by the process. Change the environment variables and redeploy/restart instead.

The Home Assistant version is unchanged in this respect: dashboard Setup continues to save through the Supervisor and restart the Home Assistant app as before.

## Network and security boundary

Home Assistant mode keeps its existing ingress-only request guard. Standalone mode removes only that Home Assistant-specific network restriction so Docker/Railway can serve normal HTTP traffic. Existing security headers remain enabled.

The standalone dashboard does not implement user authentication. If the service should not be publicly visible, keep it on a private network or place it behind an authenticated reverse proxy. Provider credentials are accepted only through environment variables and are never returned by the configuration API.

## Health and diagnostics

- `GET /healthz` reports process health, deployment mode, refresh state and whether a snapshot exists.
- `GET /api/v1/status` returns the dashboard snapshot.
- `POST /api/v1/refresh` starts the same guarded manual refresh used by the Home Assistant dashboard; only one refresh can run at a time.
- Diagnostics and scoring use the same core implementation in both deployment modes.
