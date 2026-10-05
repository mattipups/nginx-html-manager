#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v helm >/dev/null || { echo 'Helm ist erforderlich' >&2; exit 1; }
chart=charts/nginx-html-manager
helm lint "$chart" --strict
helm template verify "$chart" >/dev/null
helm template verify "$chart" --set persistence.existingClaim=existing-data >/dev/null
helm template verify "$chart" --set-string persistence.storageClass= >/dev/null
helm template verify "$chart" --set ingress.enabled=true --set ingress.adminHost=admin.example.test --set ingress.publicHost=pages.example.test --set config.adminOrigin=https://admin.example.test --set config.publicUrl=https://pages.example.test --set 'ingress.tls[0].hosts[0]=admin.example.test' --set 'ingress.tls[0].hosts[1]=pages.example.test' --set ingress.tls[0].secretName=html-tls >/dev/null
if helm template verify "$chart" --set config.maxPages=0 >/dev/null 2>&1; then
  echo 'Ungültiges Seitenlimit akzeptiert' >&2; exit 1
fi
if helm template verify "$chart" --set config.publicUrl=http://localhost:8080 >/dev/null 2>&1; then
  echo 'Identische Origins akzeptiert' >&2; exit 1
fi
printf 'Helm lint und Rendering-Varianten erfolgreich.\n'
