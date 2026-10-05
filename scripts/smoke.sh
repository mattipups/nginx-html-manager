#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Local defaults only; set these for a remote or HTTPS deployment.
admin=${ADMIN_TEST_URL:-http://localhost:8080}
public=${PUBLIC_TEST_URL:-http://localhost:8081}
user=${ADMIN_TEST_USER:-admin}
password=$(cat secrets/admin_password.txt)
response=$(curl --fail --silent --show-error -u "$user:$password" -H "Origin: $admin" -H 'X-File-Name: demo.html' -H 'Content-Type: text/html' --data-binary @examples/demo.html "$admin/api/upload")
id=$(printf '%s' "$response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
cleanup() { curl --fail --silent --show-error -u "$user:$password" -H "Origin: $admin" -X DELETE "$admin/api/pages/$id" >/dev/null; }
trap cleanup EXIT
curl --fail --silent --show-error "$public/pages/$id.html" | grep -q 'Die Veröffentlichung funktioniert'
code=$(curl --silent -o /dev/null -w '%{http_code}' "$public/")
[[ $code == 404 ]]
headers=$(curl --fail --silent --show-error -I "$public/pages/$id.html")
grep -qi 'Content-Security-Policy: sandbox;' <<< "$headers"
printf 'Upload, öffentliche Auslieferung, CSP und gesperrter Root-Pfad erfolgreich.\n'
