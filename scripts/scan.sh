#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ -f images.lock.env ]] || { echo "images.lock.env fehlt" >&2; exit 1; }
TRIVY_IMAGE=$(sed -n 's/^TRIVY_IMAGE=//p' images.lock.env)
[[ $TRIVY_IMAGE =~ @sha256:[a-f0-9]{64}$ ]] || { echo "Scanner muss per Digest referenziert sein" >&2; exit 1; }
mkdir -p artifacts
severity=${TRIVY_SEVERITY:-UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL}
result=0
for name in backend nginx; do
  if [[ $name == backend ]]; then image=local/html-manager:dev; else image=local/html-manager-nginx:dev; fi
  archive="artifacts/${name}.tar"
  docker save "$image" -o "$archive"
  chmod 644 "$archive"
  # Scan saved archives without exposing the Docker socket to the scanner.
  runner=(docker run --rm --user "$(id -u):$(id -g)" --read-only --cap-drop ALL --security-opt no-new-privileges --tmpfs /tmp:size=1g,mode=1777 -e TRIVY_CACHE_DIR=/tmp/trivy -v "$PWD/artifacts:/artifacts" "$TRIVY_IMAGE")
  "${runner[@]}" image --input "/artifacts/${name}.tar" --scanners vuln --severity "$severity" --exit-code 1 --format json --output "/artifacts/${name}-trivy.json" || result=1
  "${runner[@]}" image --input "/artifacts/${name}.tar" --format cyclonedx --output "/artifacts/${name}-sbom.cdx.json" || result=1
  rm -f "$archive"
done
exit "$result"
