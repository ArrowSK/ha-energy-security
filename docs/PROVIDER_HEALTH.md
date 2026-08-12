# Provider health and fallback semantics

Energy Security Monitor separates **energy-system condition** from **data-provider condition**. A failed provider is not evidence that a country's supply is failing.

## Health states

| State | Meaning | Runtime action |
|---|---|---|
| `healthy` | Latest attempt returned at least one valid observation. | Use observations and reset failure count. |
| `failed` | Latest attempt failed; failure threshold for cooldown has not yet been reached. | Continue to the next supported provider in the group. |
| `degraded` | Repeated failures opened the local circuit breaker. | Skip the provider until its bounded cooldown ends, then probe it again automatically. |
| no success timestamp | The current process has never received a valid observation from that provider. | Diagnostics renders `Last success never`. |

Provider health is diagnostic state. Observation freshness is separate. A previously collected value can remain valid while its original provider is temporarily failing.

## Fallback groups

A provider group returns the first successful supported provider in its configured order. This preserves a clear source hierarchy and prevents multiple feeds with different definitions from being silently blended into one measurement.

Where fallbacks have different semantics, they use different observation keys. The important example is gas:

- `gas_storage_fill_pct` means an actual storage-fill percentage from a source that publishes it;
- `gas_stock_index_pct` is a lower-confidence Eurostat monthly stock proxy and explicitly does not mean capacity fill.

The scoring engine prefers the actual fill value and uses the proxy only when a fresh actual fill is unavailable.

## Circuit breaker

After repeated failures the provider receives an exponentially increasing cooldown, capped at one hour. The circuit breaker is local and requires no project service. Successful recovery clears the failure count and restores the provider to normal use on subsequent collection cycles.

## Cache

Successful observations are stored in `/data/state.json`. Each observation keeps its own source timestamp, retrieval timestamp and TTL. Restarting the app does not make an old value fresh. Once the TTL expires, the observation is marked stale and excluded from the score.

## Supervisor watchdog

The provider manager handles data-source failures. Separately, Home Assistant Supervisor can monitor the app process through the local `/healthz` endpoint. A remote provider outage does not make the process-health endpoint fail; otherwise a harmless upstream outage could create a restart loop.

## What users should do

Normally, nothing. If one provider is failed but another provider in the same domain is healthy and the domain confidence is acceptable, fallback is doing its job.

Investigation is warranted when:

- all providers in a critical domain fail repeatedly;
- the domain becomes unknown or confidence collapses;
- a parser error persists after an app update; or
- a value is clearly implausible despite source/age being current.

When reporting a problem, include app version, country, provider ID, exact error and whether another provider succeeded.
