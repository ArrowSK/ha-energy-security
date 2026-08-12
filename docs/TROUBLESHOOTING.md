# Troubleshooting

## Dashboard says the country is unknown

With `country: auto`, confirm that Home Assistant has a country configured under HOME/general settings. If you deliberately want another country, set a two-letter ISO code in the app options and restart the app.

A valid but unsupported country may still show a dashboard with low confidence. The app does not invent data to fill unsupported domains.

## A domain says Unknown or Data limited

Open **Diagnostics** and check:

- provider state;
- last success and last failure;
- exact last error;
- observation source and age;
- stale flag;
- current notes.

Unknown is an intentional safe state. It means there is not enough fresh evidence to score the domain defensibly.

## Provider states

`healthy` means the last collection attempt succeeded. `failed` means a recent attempt failed and the manager can still retry normally. `degraded` means repeated failures triggered the local circuit-breaker cooldown.

`Last success never` means the current app process has not yet received a valid observation from that provider. Older releases could display Go's zero timestamp as roughly 739,000 days ago; version 0.1.1 fixes that display bug.

A failed provider does not mean the country is in an energy emergency. Check whether another provider in the same domain is healthy.

## Energy-Charts: `http 404`

Version 0.1.1 requests an explicit recent multi-day window. This prevents the false 404 that can occur when relying on the provider's implicit current-local-day window before that day has published data.

After updating:

1. press **Refresh** in the dashboard;
2. wait for the collection cycle to finish;
3. check whether `Energy-Charts (Fraunhofer ISE)` becomes healthy;
4. if 404 persists, record country, app version and exact error.

Do not work around a persistent upstream/country issue by hard-coding invented data.

## Eurostat oil: `returned no oil-stock values`

Version 0.1.1 asks Eurostat for the latest 12 reporting periods and chooses the newest available `STK_EUE_DIR` value for the selected country. This handles reporting lags between Member States.

If Oil remains unknown after update, the expected days-equivalent series may genuinely be absent from the returned recent periods. The app intentionally does not substitute a different oil series.

## FGSZ: `no server-rendered live gas values`

FGSZ can publish the labels on its public page while loading numerical values client-side. The app intentionally does not run a browser engine. When it cannot see trustworthy values in the received HTML, FGSZ fails safely and the gas chain continues.

For Hungary, look for one of these after FGSZ:

- `GIE AGSI` if an optional key is configured;
- `Eurostat gas stocks` without a key.

The Eurostat fallback is monthly and lower confidence. Its `gas_stock_index_pct` is not physical storage-capacity fill.

## A source is degraded

No immediate action is normally required. The provider manager uses the next supported fallback and retries the preferred source after its local circuit-breaker cooldown.

If all providers for a group fail, still-fresh cached observations can bridge a short outage. Once their TTL expires they stop contributing to the score and confidence falls.

## Optional API key does not work

The normal app remains usable without optional credentials. Remove the optional source temporarily and restart to confirm the keyless path.

For ENTSO-E, token-based fallback only works for embedded profiles that currently include an EIC mapping. For AGSI, actual coverage depends on GIE's API and the selected country.

## The app starts but Home Assistant sensors do not update

First verify that the Ingress dashboard works. Then check `enable_ha_entities: true` and inspect the app log for `home assistant entity publish` errors.

The dashboard does not depend on HA state-machine publication; an entity publishing failure does not destroy locally collected data.

## Weather is missing

Weather requires HOME coordinates from Home Assistant and `enable_weather: true`. A manual national country override does not supply replacement coordinates.

## After an internet outage or restart

The app restores `/data/state.json`, recalculates stale state from each original observation timestamp/TTL, and continues trying providers. Restarting never makes old data fresh.

## Provider failed but the domain still has a score

This is usually correct. The score may be using:

- a later provider in the fallback chain; or
- a still-fresh last-known-good observation.

Check the observation's `source`, `observed_at`, quality and stale fields in Diagnostics.

## Reporting a parser/provider problem

Include:

- app version;
- selected country;
- provider ID/name;
- exact error text;
- whether a fallback succeeded;
- approximate time of the failed refresh.

A safely redacted upstream response fixture is useful when available.

For health-state semantics and fallback details, see `docs/PROVIDER_HEALTH.md`.
