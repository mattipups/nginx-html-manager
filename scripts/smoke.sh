#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
admin=${ADMIN_TEST_URL:-http://localhost:8080}
public=${PUBLIC_TEST_URL:-http://localhost:8081}
user=${ADMIN_TEST_USER:-admin}
fixture=${SMOKE_HTML_FILE:-examples/demo.html}
[[ -f "$fixture" ]] || { echo 'HTML-Testdatei fehlt' >&2; exit 1; }
filename=$(python3 -c 'import pathlib,sys,urllib.parse; print(urllib.parse.quote(pathlib.Path(sys.argv[1]).name, safe=""))' "$fixture")
password=$(cat secrets/admin_password.txt)
response=$(curl --fail --silent --show-error -u "$user:$password" -H "Origin: $admin" -H "X-File-Name: $filename" -H 'Content-Type: text/html' --data-binary "@$fixture" "$admin/api/upload")
id=$(printf '%s' "$response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
download=$(mktemp)
cleanup() {
  rm -f "$download"
  curl --fail --silent --show-error -u "$user:$password" -H "Origin: $admin" -X DELETE "$admin/api/pages/$id" >/dev/null
}
trap cleanup EXIT
curl --fail --silent --show-error "$public/pages/$id.html" -o "$download"
cmp "$fixture" "$download"
code=$(curl --silent -o /dev/null -w '%{http_code}' "$public/")
[[ $code == 200 ]]
curl --fail --silent --show-error "$public/" | grep -q "$id.html"
headers=$(curl --fail --silent --show-error -I "$public/pages/$id.html")
grep -qi 'Content-Security-Policy: sandbox allow-scripts allow-same-origin allow-downloads allow-modals' <<< "$headers"
grep -qi "script-src 'unsafe-inline'" <<< "$headers"
grep -qi 'connect-src https:' <<< "$headers"
printf 'Upload, bytegleiche Auslieferung, interaktive CSP und Link-Übersicht auf Port 8081 erfolgreich.\n'
