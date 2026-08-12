# Architecture

Energy Security Monitor is deliberately self-contained. The installed Home Assistant app does not depend on a service run by this project and does not download country profiles, parsers or scoring rules at runtime.

## Runtime flow

```text
Home Assistant HOME country / manual override
                 |
                 v
          embedded country profile
                 |
       +---------+----------+
       |                    |
       v                    v
 ordered provider chains   HOME location
       |                    |
       +---------+----------+
                 v
        normalised observations
                 |
        freshness + validation
                 |
                 v
       deterministic scoring
                 |
       +---------+----------+
       |                    |
       v                    v
 Ingress dashboard      HA states
       |
 local /data cache
```

The executable is a single Go process. The web dashboard is embedded into that binary and rendered by the user's browser. There is no database server, Python runtime, Node process, browser engine or external UI asset service.

## Country resolution

`country: auto` asks Home Assistant for its configured country and HOME coordinates through the internal Home Assistant API. A two-letter manual override changes the national profile. HOME coordinates remain the weather location when they are available.

Country profiles are compiled into the application from `internal/country/data/countries.json`. Unknown countries resolve to a safe partial profile: the app remains available but does not invent national sources or a complete score.

## Provider groups and fallbacks

Providers are grouped by domain. Each group has an ordered provider list and an observation TTL. The manager tries the first supported healthy provider, then falls back through the chain.

A provider that repeatedly fails is put behind a local circuit breaker. After its cooldown expires it is tried again. A successful request closes the circuit automatically, so recovery does not require an app restart.

Provider health is local process state. The last-known-good observations are persisted under `/data/state.json` and restored after restart. Cache writes use a temporary file followed by rename so an interrupted write does not replace a valid cache with a partial file.

## Freshness model

Every numeric observation carries:

- measurement key and domain;
- value and unit;
- observation time;
- retrieval time;
- source name and public source URL;
- quality estimate;
- TTL;
- stale flag;
- provider group / provider ID metadata.

A stale value stays visible for diagnosis but is excluded from scoring. Missing data never becomes zero.

Current default freshness windows are deliberately conservative:

| Group | TTL |
|---|---:|
| Electricity | 90 minutes |
| Gas | 36 hours |
| Oil | 45 days |
| Hydrology | 36 hours |
| Weather stress | 6 hours |

## Scoring boundary

Providers only collect and normalise data. They do not decide whether the country is secure. `internal/engine` consumes the normalised observations and produces current, seven-day, strategic and headline scores plus a separate confidence value.

That separation is intentional. A source parser can be replaced without changing the score contract, and a scoring change can be tested without making live network requests.

## Home Assistant boundary

The app uses Home Assistant's internal API for two purposes:

1. reading HOME country / location;
2. publishing the compact `sensor.energy_security_*` state set when enabled.

The dashboard itself is served through Ingress. Port 8099 is not exposed on the host. The HTTP layer additionally rejects dashboard/API requests that do not originate from loopback or the Home Assistant Ingress proxy address.

## No project runtime infrastructure

The runtime has no endpoint belonging to ArrowSK or this repository. GitHub is relevant to source distribution, installation and image updates only. An already installed image continues operating if the repository becomes unavailable; only future installation/update discovery would be affected.
