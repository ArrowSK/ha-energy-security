# Support matrix

Support levels describe source coverage, not the quality of a country's energy system.

- **Full** — keyless electricity, EU strategic fuel evidence where applicable, plus dedicated national gas/hydrology adapters and optional higher-quality fallbacks.
- **Good** — keyless electricity, HOME weather and Eurostat strategic oil/gas evidence for eligible EU profiles; dedicated national fuel/water adapters may still be absent.
- **Electricity** — keyless electricity plus HOME weather when coordinates are available.
- **Limited** — country profile is recognised but no default national electricity source is configured.

The scoring engine reduces confidence when expected evidence is absent. A `good` support label must never be read as a high energy-security score.

## 0.1.1 profiles

| Coverage | Countries |
|---|---|
| Full | Hungary (HU) |
| Good | Austria, Belgium, Bulgaria, Cyprus, Czechia, Germany, Denmark, Estonia, Spain, Finland, France, Greece, Croatia, Ireland, Italy, Lithuania, Luxembourg, Latvia, Malta, Netherlands, Poland, Portugal, Romania, Sweden, Slovenia, Slovakia |
| Electricity | Albania, Bosnia and Herzegovina, Switzerland, Montenegro, North Macedonia, Norway, Serbia, Türkiye, Ukraine, United Kingdom, Kosovo |
| Limited | Iceland |

All currently embedded non-limited electricity profiles use Energy-Charts as their default keyless electricity source.

Eurostat-enabled EU profiles additionally enable:

- emergency oil-stock evidence through `nrg_stk_oem`; and
- keyless monthly natural-gas closing-stock evidence through `nrg_stk_gasm`.

The Eurostat gas fallback is intentionally a lower-confidence strategic stock proxy, not a live storage-capacity fill measure.

## Hungary reference profile

Hungary additionally includes:

- optional ENTSO-E token-based electricity fallback;
- FGSZ as the preferred public gas-system page adapter;
- optional GIE AGSI higher-quality gas-storage data;
- Eurostat monthly gas stocks as a keyless fallback when FGSZ does not expose server-rendered live values;
- HYDROINFO Danube/hydrology data for Budapest/Paks and related published measurements.

A failed FGSZ provider does not by itself reduce Hungary to an unsupported profile. The gas domain can continue from AGSI or Eurostat, with confidence reflecting the source actually used.

## Unknown countries

A manually entered valid ISO code with no embedded profile resolves safely to that code with partial/unknown coverage. The app does not guess a neighbouring country's providers.

## Roadmap criteria

A country moves to a stronger support class only after its adapter has:

1. a source with acceptable legal and operational access;
2. bounded timeouts and response sizes;
3. explicit timestamp and unit handling;
4. safe failure/fallback behaviour;
5. deterministic tests or fixtures where practical;
6. no mandatory paid credential;
7. semantic separation when a fallback measures a proxy rather than the same physical quantity.
