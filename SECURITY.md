# Security policy

## Supported version

Security fixes are applied to the latest published version of Energy Security Monitor.

## Reporting a vulnerability

Please do not publish exploit details in a public issue before a fix is available. Use GitHub's private vulnerability reporting feature for this repository when available. If that feature cannot be used, open a minimal issue stating that you need a private security contact without including the exploit or sensitive data.

## Security model

The app is designed to run behind Home Assistant Ingress. It exposes no host port through `config.yaml`, requests no privileged capabilities, does not mount Home Assistant configuration, does not use the Docker API, and does not execute downloaded code.

The dashboard server accepts loopback requests for local health/testing and the Home Assistant Ingress proxy address. All other direct client addresses are rejected.

Optional provider tokens are read from `/data/options.json`, used only in requests to their respective provider, and are never returned by the dashboard API or written into the state cache.

## Out of scope

Availability or correctness defects in third-party public data providers are not software vulnerabilities by themselves. Report cases where malformed provider data can bypass validation, cause code execution, disclose credentials, or materially corrupt state as security issues.
