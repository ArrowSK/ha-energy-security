#!/usr/bin/with-contenv bashio
set -euo pipefail
exec /usr/local/bin/energy-security \
  --config /data/options.json \
  --data-dir /data \
  --listen :8099
