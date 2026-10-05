#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v docker >/dev/null
command -v python3 >/dev/null
out=$(mktemp)
trap 'rm -f "$out"' EXIT
for item in \
  "GO_IMAGE|cgr.dev/chainguard/go:latest-dev" \
  "STATIC_IMAGE|cgr.dev/chainguard/static:latest" \
  "NGINX_IMAGE|cgr.dev/chainguard/nginx:latest" \
  "TRIVY_IMAGE|aquasec/trivy:latest"; do
  key=${item%%|*}; image=${item#*|}
  digest=$(docker buildx imagetools inspect "$image" --format '{{json .Manifest}}' | python3 -c 'import json,sys; print(json.load(sys.stdin)["digest"])')
  [[ $digest =~ ^sha256:[a-f0-9]{64}$ ]] || { echo "Ungültiger Digest für $image" >&2; exit 1; }
  printf '%s=%s@%s\n' "$key" "$image" "$digest" >> "$out"
done
mv "$out" images.lock.env
chmod 644 images.lock.env
printf 'images.lock.env aktualisiert. Nach Tests und Scan in Git committen.\n'
