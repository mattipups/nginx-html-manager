# Prüfprotokoll — 2026-10-05

## Hier erfolgreich ausgeführt

- JavaScript-Syntaxprüfung mit `node --check`.
- Sechs von sechs UI-Logiktests mit `node --test`: leere Liste, sichere Dateinamen-Darstellung, opener-Isolation, Filter, abgelehnte Dateiendung und abgebrochene Löschung.
- Bash-Syntaxprüfung aller sechs Shell-Skripte mit `bash -n`.
- YAML-Parsing von Compose und GitHub-Actions-Workflow. Das ist keine Docker-Compose-Schemavalidierung.
- JSON-Parsing der Renovate-Konfiguration. Das ist keine Renovate-Schemavalidierung.

- Makefile-Targets mit `make -n` geparst.
- Git-Ignore-Regeln in temporärem Repository geprüft: Secrets/Uploads ignoriert, Digest-Lock und Quellcode nicht ignoriert.
- Compose-Sicherheitsparameter statisch geprüft; keine Laufzeitverifikation.

## Nicht ausgeführt

- 14 bereitgestellte Go-Tests, `go vet`, Go-Kompilierung und Formatierung: Go ist in der Erstellungsumgebung nicht installiert.
- Docker-Build/Start, NGINX-`-t`, Stack-Smoke-Test und Browser-End-to-End-Test: Docker und NGINX sind nicht verfügbar.
- Digest-Auflösung, Registry-Pulls, Trivy-CVE-Scan und SBOM-Erstellung: kein entsprechender Runtime-/Registry-Test durchgeführt.

Die UI-Tests verwenden einen DOM-Mock, keinen realen Browser. Das Paket ist ein Implementierungsstand mit bereitgestellten Verifikationsschritten, kein Nachweis eines bereits geprüften Produktivdeployments. Digest-Werte werden ausschließlich über `scripts/lock-images.sh` auf dem Zielsystem ermittelt. Eine allgemeine oder dauerhafte CVE-Freiheit wird nicht zugesichert.
