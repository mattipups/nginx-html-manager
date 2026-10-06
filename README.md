# NGINX HTML Manager 1.1.0

HTML-Dateien veröffentlichen, eigene URL-Namen vergeben und Originaldateien herunterladen.

## Bedienung

- Link ändern: z. B. argocd-builder ergibt /pages/argocd-builder.html.
- Erlaubt: 1–64 Kleinbuchstaben, Ziffern und Bindestriche; kein Bindestrich am Rand.
- Keine Pfade, Domains, Dateiendungen oder 32-stelligen Hex-IDs eingeben.
- Leere Eingabe entfernt den eigenen Namen. Der ID-Link bleibt immer gültig.
- Ein neuer Name ersetzt den bisherigen eigenen Link; belegte Namen werden abgewiesen.
- Herunterladen liefert nach Anmeldung die unveränderte Datei mit ihrem Originalnamen.
- Löschen entfernt ID-Datei, eigenen Link und Metadaten.

## Betrieb

```bash
make init lock build up test smoke scan chart
```

Für bestehende Installationen beide Images neu bauen und testen; Volumes NICHT löschen.
Compose und Scanner nutzen local/html-manager:1.1.0 und local/html-manager-nginx:1.1.0.
Für Helm beide Images testen und in die eigene Registry pushen, dann Chart 1.1.0 ausrollen.
Bestehende Secret-/PVC-/Origin-Einstellungen beibehalten. Keine automatische Bereitstellung,
kein Release-Tag und kein GitHub-Release durch den Code-Commit.
Speichermigration und PVC-Betrieb: docs/v1.0.2.md; dortige Image-Tags sind historisch.

Eigene Links verwenden Hardlinks innerhalb public/; das Volume muss sie unterstützen.
Nur eine Backend-Instanz pro Datenbestand. Link/Metadaten-Änderungen sind nicht vollständig
absturztransaktional; nach einem Abbruch Backups und Dateien vor manueller Bereinigung prüfen.

## Sicherheit und Prüfstand

Admin und Public getrennt halten. Veröffentlichte Seiten bleiben anonym lesbar.
Interaktives HTML ist freigegeben: nur vertrauenswürdige Dateien hochladen. Seiten derselben
Public-Origin teilen LocalStorage und dürfen beliebige HTTPS-APIs aufrufen.
Weitere Grenzen: docs/interactive-html.md. Archivierte Dokumentation: README_org.md.

Bei Vorbereitung: sieben neue UI-Tests, JS-/Bash-Syntax und Versions-/YAML-Prüfungen bestanden.
Die früheren sechs Backend-Vorbereitungsprüfungen waren statische/Dateisystem-Prüfungen.
Go-Tests/Kompilierung, NGINX-/Docker-/Helm-Laufzeit, Browser-End-to-End und CVE-Scan wurden
hier nicht ausgeführt. Vor Produktion CI und Tests auf dem Zielsystem erfolgreich abschließen.
