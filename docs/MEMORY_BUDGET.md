# Memory budget

The release requirement is **less than 50 MB steady-state RSS**. The engineering target is **35 MB or less** on supported architectures under ordinary operation.

## Runtime design choices

The production image runs one Go executable. The following are intentionally absent:

- Python runtime;
- Node.js runtime;
- browser/Chromium process;
- database server;
- server-side chart renderer;
- unbounded in-memory history;
- heavyweight analytics libraries.

HTTP response bodies are size-limited, provider requests have timeouts, score history is capped, and the persistent cache is a small JSON file.

## Local measurement

A useful RSS check measures the running process after the HTTP server is ready and the initial allocation burst has settled.

Example on Linux:

```sh
./energy-security --config ./test-options.json --data ./tmp-data --listen 127.0.0.1:8099 &
pid=$!
sleep 10
awk '/VmRSS/ {print}' /proc/$pid/status
kill $pid
```

For the release gate, repeat on the Home Assistant target architecture/container because allocator behaviour and base-image libraries can differ from a developer workstation.

## What counts as failure

A reproducible steady-state RSS of 50 MB or more on `aarch64` or `amd64` blocks a stable release. A transient build process or browser memory does not count toward the app's runtime RSS; a persistent collector leak does.

## Regression approach

Memory should be measured after changes that add:

- new parsers with large response payloads;
- long-lived caches;
- new protocol libraries;
- additional background goroutines;
- richer server-side processing.

When possible, improve bounded streaming/parsing before increasing the budget.
