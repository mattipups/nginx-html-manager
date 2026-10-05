# Prüfprotokoll 1.0.2 — 2026-10-06

Basis: c9a0399ff3499c42abdc5c7006d6514a234e0bd9.

## Tatsächlich ausgeführt

- Compose YAML parsing
- CI workflow YAML parsing
- Named volume and public-only read-only NGINX mount
- Static container security and tmpfs settings
- Build and scan image version consistency
- VERSION file consistency
- Bash syntax for init.sh and scan.sh
- All Makefile targets parsed with make -n
- CI embedded Python syntax
- Existing UI IDs preserved and logo routes match
- CSS stylesheet parsing
- Original logo SHA-256 and repository Git blob identity
- Static storage initializer and original HTML validation pattern
- Static Docker build inclusion of tests, logos and owned data tree

## Nicht ausgeführt

- Go compilation, gofmt, Go tests and go vet
- Helm lint/rendering
- Docker/Compose schema and runtime
- NGINX syntax and stack smoke test
- Kubernetes PVC/CSI runtime
- Trivy scan, digest refresh and SBOM
- Real browser E2E

Go, Helm, Docker und NGINX sind hier nicht installiert. Statische Prüfungen ersetzen keine Kompilierung, echtes Helm-Rendering oder Laufzeitabnahme. Bereitgestellte Tests/CI müssen vor Produktion erfolgreich laufen. Die neuen Go-Tests prüfen Original-Logo-Prüfsummen, Authentifizierung, HEAD und idempotente Speicherinitialisierung. Keine Zusicherung dauerhafter CVE-Freiheit.
