# NGINX HTML Manager 1.0.2

Deutsche HTML-Veröffentlichung mit Go-Backend, NGINX, Docker Compose und Helm-Chart. HTML-Dateien importieren, Seiten auflisten/filtern, öffnen, Links kopieren und löschen. Metadaten enthalten Name, Größe, UTC-Zeit und SHA-256; zufällige IDs verhindern Überschreiben gleicher Dateinamen.

Neu: PVC für HTML UND Metadaten, Named Volume, eingebettete originale Light-/Dark-Logos. Vollständige Installations-, Migrations- und Kubernetes-Anleitung: [docs/v1.0.2.md](docs/v1.0.2.md). Vorhandenes ./data wird nicht automatisch übernommen!

```bash
make init lock build up test smoke scan chart
cat secrets/admin_password.txt
```

Admin: http://localhost:8080, Benutzer admin; Public: http://localhost:8081/pages/<id>.html. Die öffentliche Root-URL liefert 404. examples/demo.html zum Test hochladen. Browser-URL und ADMIN_ORIGIN müssen exakt übereinstimmen. Moderne Docker-/Compose-Version mit Named-Volume-Subpath-Unterstützung erforderlich.

HTTP Basic nur lokal oder über HTTPS. Getrennte Admin-/Public-Origins; Public-CSP blockiert Skripte, Formulare und externe Ressourcen. Keine HTML-Sanitization, kein Virenscanner, keine privaten Dokumente: Seiten sind anonym lesbar. Nur ein Backend-Prozess pro Datenbestand. Keine verteilten Locks, OIDC, Rollen, ZIP-/Asset-Uploads oder HTML-Bearbeitung. Runtime-Container laufen als UID/GID 65532 mit read-only Rootfilesystem und entfernten Capabilities.

Echte Image-Digests über scripts/lock-images.sh aktualisieren und nach Build/Tests/Scan committen. Trivy scannt beide Runtime-Images, erstellt SBOMs und blockiert standardmäßig jeden CVE-Befund. Ein leerer Scan ist keine dauerhafte CVE-Garantie. CI veröffentlicht weder Images noch Releases. Prüfstand und offene Laufzeittests: [VALIDATION.md](VALIDATION.md). CONTRIBUTING.md und SECURITY.md gelten weiterhin.
