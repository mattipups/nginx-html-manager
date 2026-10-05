# Changelog

## 1.0.2 — 2026-10-06

- Helm-Chart mit PVC für HTML/Metadaten, Ein-Pod-Zwei-Container-Deployment, Recreate, Service und optionalem HTTPS-Ingress.
- StorageClass, PVC-Größe, bestehender Claim und PVC-Erhalt konfigurierbar.
- Docker Named Volume; NGINX sieht ausschließlich public/ read-only.
- Originale Light-/Dark-Logos eingebettet, automatische Farbschema-Auswahl.
- Shell-freie Storage-Initialisierung, zusätzliche Go-Tests, Migration/Backup-Dokumentation und Helm-Prüfungen.

## 1.0.1 — 2026-10-06

- Erneute Bereitstellung des beschriebenen v1.0-Funktionsumfangs als nginx-html-manager-v1.0.1.zip.
- Projektdateien anhand des Gesprächs neu zusammengestellt, da das vorherige Archiv nicht mehr verfügbar war.
- Versionsmetadaten und Dokumentation auf 1.0.1 aktualisiert; Prüfungen erneut ausgeführt.
- Keine zusätzliche Authentifizierung oder neue Funktion: weiterhin Basic Auth, kein OIDC.
- Keine byteidentische Reproduktion des nicht mehr verfügbaren v1.0.0-Archivs zugesichert.

## 1.0.0 — 2026-10-05

- Deutsche HTML-Upload- und Verwaltungsoberfläche.
- Go-Backend ohne externe Module, Basic Authentication und Origin-Prüfung.
- NGINX-Auslieferung auf separatem Port mit restriktiver CSP.
- Chainguard-Runtime-Images, Non-root, read-only Filesystem und Digest-Lock.
- Docker Compose, Tests, Scan/SBOM-Skripte und GitHub Actions.
