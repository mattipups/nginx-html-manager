# Interaktive HTML-Dateien

## Änderung

Die öffentliche CSP in Docker und Helm erlaubt eingebettetes JavaScript,
Inline-CSS, LocalStorage, Downloads, alert/confirm/prompt und Dokumentationslinks.
HTTPS-API-Aufrufe sind für den Git-Commit des ArgoCD-Builders erlaubt.
Externe Skripte, eval, Formulareinsendungen, Frames und Plugins bleiben gesperrt.
Die Verwaltungs-CSP, Basic Authentication, Origin-Prüfung und Upload-Validierung
werden nicht gelockert. Es werden keine Java-Dateien oder separaten Assets zugelassen.
Diese Änderung ersetzt die bisherigen Hinweise auf vollständig gesperrte Skripte.

## Vertrauen und Grenzen

Die Richtlinie gilt für ALLE veröffentlichten HTML-Dateien, auch vorhandene.
Nur vertrauenswürdige Inhalte hochladen. Alle Seiten auf derselben Public-Origin
teilen LocalStorage und können dort gespeicherte Daten gegenseitig lesen oder ändern.
HTTPS-Verbindungen zu beliebigen Hosts erlauben auch das Senden sensibler Eingaben.
Bei engeren Anforderungen connect-src auf eigene Git-API-Hosts begrenzen.
Keine vertraulichen Dokumente veröffentlichen; zufällige IDs sind kein Zugriffsschutz.
Admin und Public weiterhin auf getrennten Origins betreiben; für starke Isolation
von Cookies und sonstigen Daten getrennte Domains ohne gemeinsame Domain-Cookies nutzen.

## Betrieb und Abnahme

Nach Übernahme in Docker beide Images neu bauen, da auch der UI-Hinweis geändert wurde:

```bash
bash scripts/compose.sh up -d --build --wait
bash scripts/test.sh
SMOKE_HTML_FILE=argocd-builder-v2.3.7.html bash scripts/smoke.sh
```

Die Testdatei vorher lokal im Projektordner ablegen, NICHT ins Repository committen.
Der Smoke-Test veröffentlicht sie vorübergehend anonym und löscht den Testupload danach.
Er prüft bytegleiche Auslieferung und CSP-Header, nicht die Ausführung im Browser.
Für Helm das aktualisierte Backend-Image bauen und bereitstellen und das geänderte
Chart per helm upgrade übernehmen; der ConfigMap-Checksum löst den Pod-Neustart aus.

Danach die Datei regulär hochladen und über Öffnen testen: YAML-Erzeugung/-Import,
Presets speichern/laden, Theme, Base64, YAML-/JSON-Downloads, Zwischenablage,
Bestätigungsdialoge und Dokumentationslinks. Für Zwischenablage HTTPS oder localhost
verwenden; Browserberechtigungen können den Zugriff begrenzen.
Git-Commit nur mit einem freigegebenen Testrepository und passendem Token prüfen.
CORS, TLS, Tokenrechte und Branch-Schutz bleiben Voraussetzungen der Ziel-API;
die neue CSP umgeht diese Prüfungen nicht. Kein pauschaler CORS-Proxy wird eingerichtet.
Vorgeschaltete Ingress-/Reverse-Proxys dürfen keine zusätzliche restriktive CSP setzen.

## Prüfstand

Statische CSP-Tests und Bash-Syntaxprüfung sind keine Laufzeitabnahme.
NGINX-Syntaxprüfung, Container-/Helm-Laufzeit und vollständiger Browser-End-to-End-Test
müssen vor produktiver Freigabe erfolgreich durchgeführt werden.
