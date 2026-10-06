# Changelog

## 1.3.0 — 2026-10-06

- Liste der verfügbaren Seiten-Links auf Port 8081 bereitgestellt (Autoindex unter `/` und `/pages/`).
- Öffentliche Root-URL liefert kein 404 mehr, sondern eine navigierbare Link-Übersicht aller publizierten HTML-Dateien und eigenen URLs.
- Smoke-Test, NGINX-Konfiguration und Helm-Templates auf Version 1.3.0 synchronisiert.

## 1.2.0 — 2026-10-06

- Datei-Aktualisierung (In-Place) ohne Bruch bestehender ID-Links oder eigener Namen.
- Versionshistorie und Rollback-Mechanismus.
- Sicherheitsprofile für statisches HTML, interaktive Apps und HTTPS-API-Ziele.
- Integrationstests und automatisierte Browser-Tests.

## 1.1.0 — 2026-10-06

- Eigene URL-Namen pro Upload mit Konfliktprüfung; bestehende ID-Links bleiben gültig.
- Originaldatei-Download über die angemeldete Verwaltung, inklusive HEAD.
- Link ändern und Herunterladen in der Oberfläche; neue Backend- und UI-Tests.
- Interaktive HTML-Unterstützung in Docker und Helm bleibt erhalten.
- Anwendung, Compose-Images, Scanner und Helm auf 1.1.0 angehoben.
- Unbeabsichtigte FastCGI-Temporärpfadänderung im Backend-Commit korrigiert.

## 1.0.2 — 2026-10-06

- Helm-Chart mit PVC für HTML/Metadaten, Ein-Pod-Zwei-Container-Deployment, Recreate, Service und optionalem HTTPS-Ingress.
