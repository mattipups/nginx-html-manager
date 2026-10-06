#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
command -v openssl >/dev/null || { echo "openssl ist erforderlich" >&2; exit 1; }
[[ -f .env ]] || cp .env.example .env
mkdir -p secrets
if [[ ! -s secrets/admin_password.txt ]]; then openssl rand -hex 32 > secrets/admin_password.txt; fi
chmod 700 secrets
chmod 444 secrets/admin_password.txt
# The image initializes a NEW named volume. Existing ./data is never moved.
printf 'Initialisiert. Passwort: cat secrets/admin_password.txt\n'
printf 'Vorhandene Uploads zuerst gemäß docs/v1.0.2.md migrieren.\n'
