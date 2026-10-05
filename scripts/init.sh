#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
command -v openssl >/dev/null || { echo "openssl ist erforderlich" >&2; exit 1; }
[[ -f .env ]] || cp .env.example .env
mkdir -p secrets
if [[ ! -s secrets/admin_password.txt ]]; then openssl rand -hex 32 > secrets/admin_password.txt; fi
# Compose file secrets retain host permissions; protect the parent, mount a readable file.
chmod 700 secrets
chmod 444 secrets/admin_password.txt
if [[ $(id -u) == 0 ]]; then
  install -d -m 0755 -o 65532 -g 65532 data data/public data/meta
else
  sudo install -d -m 0755 -o 65532 -g 65532 data data/public data/meta
fi
printf 'Initialisiert. Passwort anzeigen: cat secrets/admin_password.txt\n'
printf 'Danach: ./scripts/lock-images.sh && ./scripts/compose.sh up -d --build --wait\n'
