#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
source images.lock.env 2>/dev/null || true
TRIVY_IMAGE=${TRIVY_IMAGE:-aquasec/trivy:latest}
mkdir -p artifacts
severity=${TRIVY_SEVERITY:-UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL}
result=0
for name in backend nginx; do
  if [[ $name == backend ]]; then image=local/html-manager:1.4.0; else image=local/html-manager-nginx:1.4.0; fi
  archive="artifacts/${name}.tar"
  docker save "$image" -o "$archive"
  chmod 644 "$archive"
  runner=(docker run --rm --user "$(id -u):$(id -g)" --read-only --cap-drop ALL --security-opt no-new-privileges --tmpfs /tmp:size=1g,mode=1777 -e TRIVY_CACHE_DIR=/tmp/trivy -v "$PWD/artifacts:/artifacts" "$TRIVY_IMAGE")
  "${runner[@]}" image --input "/artifacts/${name}.tar" --scanners vuln --severity "$severity" --exit-code 1 --format json --output "/artifacts/${name}-trivy.json" || result=1
  "${runner[@]}" image --input "/artifacts/${name}.tar" --format cyclonedx --output "/artifacts/${name}-sbom.cdx.json" || result=1
  rm -f "$archive"
done
exit "$result"
