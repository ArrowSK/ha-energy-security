# Support matrix

Support levels describe source coverage, not the quality of a country's energy system.

- **Full** — electricity plus national gas/hydrology adapters and EU oil where applicable.
- **Good** — keyless electricity plus EU oil data and HOME weather; additional fuel/water domains may be unknown.
- **Electricity** — keyless electricity plus HOME weather when location is available.
- **Limited** — profile recognised but no default national electricity source in 0.1.0.

The score engine reduces confidence when expected evidence is absent. A `good` coverage label must not be read as a high security score.

## 0.1.0 profiles

| Coverage | Countries |
|---|---|
| Full | Hungary (HU) |
| Good | Austria, Belgium, Bulgaria, Cyprus, Czechia, Germany, Denmark, Estonia, Spain, Finland, France, Greece, Croatia, Ireland, Italy, Lithuania, Luxembourg, Latvia, Malta, Netherlands, Poland, Portugal, Romania, Sweden, Slovenia, Slovakia |
| Electricity | Albania, Bosnia and Herzegovina, Switzerland, Montenegro, North Macedonia, Norway, Serbia, Türkiye, Ukraine, United Kingdom, Kosovo |
| Limited | Iceland |

All currently embedded non-limited electricity profiles use Energy-Charts as the default keyless source. Hungary additionally has an optional ENTSO-E token-based electricity fallback, FGSZ gas collection and HYDROINFO hydrology.

EU profiles marked `good`/`full` enable the Eurostat emergency-oil collector, but the oil domain remains conservative/experimental in 0.1.0 and may show unknown if a trustworthy days-equivalent series cannot be selected.

## Unknown countries

A manually entered valid ISO code that has no embedded profile resolves safely to the code itself with partial/unknown coverage. The app does not guess a neighbouring country's sources.

## Roadmap criteria

A country moves to a stronger support class only after its adapter has:

1. a source with acceptable legal/operational access;
2. bounded timeouts and parsing;
3. explicit timestamp and unit handling;
4. safe failure behaviour;
5. deterministic tests/fixtures where practical;
6. no mandatory paid credential.
