#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ -f .env && -f images.lock.env ]] || { echo "Zuerst scripts/init.sh und scripts/lock-images.sh ausführen" >&2; exit 1; }
exec docker compose --env-file .env --env-file images.lock.env "$@"
