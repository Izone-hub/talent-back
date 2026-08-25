#!/usr/bin/env bash
# Vendors node_modules into every Node-based framework template so the
# sandbox can run `npm test` offline (containers run with --network none).
set -euo pipefail

cd "$(dirname "$0")"

for fw in react vue svelte express nextjs; do
  if [ -d "$fw/default" ]; then
    echo "==> Vendoring $fw/default ..."
    (cd "$fw/default" && npm install --no-audit --no-fund)
  fi
done

echo "Done. Templates are ready for offline sandbox execution."
