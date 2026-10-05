#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ -f images.lock.env ]] || { echo "Zuerst scripts/lock-images.sh ausführen" >&2; exit 1; }
GO_IMAGE=$(sed -n 's/^GO_IMAGE=//p' images.lock.env)
[[ $GO_IMAGE =~ @sha256:[a-f0-9]{64}$ ]] || exit 1
# Test stage includes go test and go vet; the ordinary application build also runs them.
docker build --target test --build-arg "GO_IMAGE=$GO_IMAGE" -t local/html-manager-test:dev .
# NGINX syntax test; no shell or curl needed in the runtime image.
./scripts/compose.sh run --rm --no-deps --entrypoint /usr/sbin/nginx nginx -t
if command -v node >/dev/null; then node --test tests/ui.test.cjs; fi
