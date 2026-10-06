# Changelog

## [1.2.0] - 2026-10-06

### Hinzugefügt
- **Datei-Aktualisierung (In-Place)**: Bestehende IDs und Custom URLs bleiben beim Aktualisieren der Datei unverändert.
- **Versionshistorie & Rollback**: Automatische Versionierung aller Uploads/Updates und gezieltes Rollback auf vorherige Dateistände.
- **Sicherheitsprofile**: Drei abgestufte Sicherheitsstufen (`static`, `interactive-local`, `api-enabled`) mit angepassten CSP-Headern.
- **E2E & Browser-Tests**: Vollständige Playwright-Testsuite für CI/CD zur Validierung von Upload, Update, Rollback, Custom-Links und Downloads.
