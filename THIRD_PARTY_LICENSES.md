# Third-party software and data

## Runtime software dependencies

The Go module intentionally has no third-party Go module dependencies. The container uses Home Assistant's base image and Alpine packages according to their respective licences. Build tooling uses the official Go toolchain.

## External data providers

No external provider dataset is copied into or redistributed with this repository. The running app requests current observations directly from configured public providers. Their data, websites, APIs and trademarks remain governed by their own terms.

Energy-Charts states that its API data are CC BY 4.0 unless otherwise noted and requires attribution to Energy-Charts.info. The app preserves the source name and source URL with observations.

Eurostat and other government/public operators are referenced as data sources, not software dependencies. Users and redistributors remain responsible for complying with provider terms applicable to their use.

Open-Meteo, GIE, ENTSO-E, FGSZ and HYDROINFO are not affiliated with this project.
