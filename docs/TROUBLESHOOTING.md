# Troubleshooting

## Dashboard says the country is unknown

With `country: auto`, confirm that Home Assistant has a country configured under its HOME/general settings. If you deliberately want another country, set the two-letter ISO code in the app options and restart the app.

A valid but unsupported country may still show a dashboard with low confidence. That is expected; the app will not invent data to fill unsupported domains.

## A domain says unknown or data limited

Open **Diagnostics** and check:

- provider state;
- last success/failure;
- observation age;
- stale flag;
- current notes.

Unknown is an intentional safe state. It means there is not enough fresh evidence to score that domain.

## A source is degraded

No immediate action is normally required. The provider manager uses the next supported fallback and periodically retries the preferred source after its local circuit-breaker cooldown.

If all providers for a group fail, still-fresh cached observations can bridge a short outage. Once their TTL expires they stop contributing to the score and confidence falls.

## Optional API key does not work

The normal app remains usable without optional credentials. Remove the key/token temporarily and restart to confirm the default keyless path.

For ENTSO-E, the current embedded token-based fallback profile is Hungary only. For AGSI, coverage depends on provider/API availability for the selected country.

Do not paste credentials into a public issue. The dashboard intentionally never displays them.

## The app starts but Home Assistant sensors do not update

First verify the Ingress dashboard itself works. Then check that `enable_ha_entities` is true and inspect the app log for `home assistant entity publish` errors.

The dashboard does not depend on the state-machine sensors; a publishing failure does not destroy locally collected data.

## Weather is missing

Weather requires HOME coordinates from Home Assistant and `enable_weather: true`. A manual national country override does not supply substitute coordinates.

## After an internet outage

The app restores `/data/state.json`, marks observations according to their original timestamps/TTLs, and continues trying providers. Stale values remain visible for diagnosis but are excluded from scoring.

## Reporting a parser problem

Include:

- app version;
- selected country;
- provider ID from Diagnostics;
- exact error text;
- whether a fallback succeeded.

Do not include optional keys/tokens or Home Assistant backups. If the upstream public response can be saved safely, a redacted fixture is useful for reproducing parser changes.
