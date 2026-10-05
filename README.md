# NGINX HTML Manager

Version 1.0.0 · deutschsprachige, Git-fähige HTML-Veröffentlichung.

Ein Go-Backend nimmt HTML-Dateien an. NGINX liefert die gespeicherten Dateien direkt aus. Die Verwaltungsoberfläche ist in das Go-Binary eingebettet; sie benötigt weder Node.js zur Laufzeit noch ein JavaScript-Framework, eine Datenbank oder externe Go-Module.

## Funktionsumfang

- Mehrere `.html`/`.htm`-Dateien über Auswahl oder Drag & Drop importieren, seriell mit Fortschrittsanzeige.
- Veröffentlichte Seiten auflisten, nach Dateiname filtern, öffnen, Link kopieren und nach Bestätigung löschen.
- Pro Upload eine kryptografisch zufällige ID; gleicher Dateiname überschreibt keine existierende Seite.
- Metadaten mit Originalname, Dateigröße, UTC-Zeitstempel und SHA-256.
- Basic Authentication, Origin-Prüfung bei Änderungen, Upload- und Seitenlimits.
- Persistente Dateien unter `data/`; Neustarts erfordern kein erneutes Hochladen.
- NGINX liest nur `data/public`; Metadaten und Secrets sind nicht öffentlich gemountet.
- Default: 5 MiB pro Datei, 500 veröffentlichte Seiten, nur ein paralleler Upload.

Nicht enthalten: OIDC, Benutzer-/Rollenverwaltung, Virenscanner, ZIP-Import, Asset-Upload, Bearbeitung vorhandener Seiten und Helm-Chart. Dieses Paket enthält Docker Compose, keine fertige Kubernetes-Installation.

## Architektur

```text
Browser -> http://localhost:8080 -> NGINX admin listener -> Go-Backend :8080
                                                             |
                                                             v
                                                   data/meta + data/public
Browser -> http://localhost:8081/pages/<id>.html -> NGINX public listener
                                                             |
                                                             v
                                                   data/public (read-only)
```

Zwei Ports sind zwei Browser-Origins, auch bei gleichem Hostnamen. Basic-Auth-Requests zur Verwaltung bleiben auf der Verwaltungs-Origin. Die öffentliche Origin bietet keine API, kein Listing und keine Weiterleitung zum Backend. Der Backend-Port wird nicht auf dem Host veröffentlicht.

HTML bleibt potenziell gefährlicher aktiver Inhalt. Der Standard-CSP der öffentlichen Seiten verwendet `sandbox`, verhindert JavaScript, Formulare, Einbettung und externe Ressourcen. Inline-CSS und `data:`-Bilder sind erlaubt. Es gibt keine HTML-Sanitization; Formatprüfung und CSP sind keine Schadcodeprüfung.

## Voraussetzungen

- Linux-Server mit Docker Engine, Docker Compose v2 und Docker Buildx; Compose mit `--wait` und mehreren `--env-file`-Argumenten.
- Bash, OpenSSL, Python 3, curl, Git und für Erstinitialisierung sudo/root.
- Internetzugriff auf die verwendeten Registries und die Trivy-Datenbank. Ggf. `docker login cgr.dev` für die eigene Registry-/Katalogberechtigung.
- Optional Node.js für die sechs UI-Logiktests; kein Node.js im Produktivbetrieb.
- Default-Anleitung für rootful Docker auf Linux. Bei rootless Docker, User-Namespace-Remapping, SELinux, Docker Desktop oder NFS müssen UID-Mapping bzw. Volume-Labels/Berechtigungen passend zum Host angepasst werden.

## Schnellstart

ZIP entpacken und in den Projektordner wechseln:

```bash
cd nginx-html-manager
chmod +x scripts/*.sh
./scripts/init.sh
./scripts/lock-images.sh
./scripts/compose.sh up -d --build --wait
./scripts/compose.sh ps
cat secrets/admin_password.txt
```

`init.sh` erstellt `.env` und ein zufälliges 64-Zeichen-Hex-Passwort. Es initialisiert die Datenverzeichnisse für UID/GID 65532. Compose-Dateisecrets erhalten Host-Dateirechte; die Passwortdatei ist deshalb lesbar gemountet, ihr Host-Elternverzeichnis bleibt mit Modus 0700 geschützt. `.env`, `secrets/` und Upload-Daten gehören nicht in Git.

- Verwaltung: [http://localhost:8080](http://localhost:8080), Benutzer `admin`, Passwort aus der Datei.
- Beispielseite: `examples/demo.html` im Browser importieren und anschließend auf „Öffnen“ klicken.
- Öffentliche URL: `http://localhost:8081/pages/<32-stellige-id>.html`.
- Öffentliche Root-URL `/` liefert absichtlich 404: Es gibt kein öffentliches Inhaltsverzeichnis.

Die Ports binden standardmäßig nur an `127.0.0.1`. Beim Zugriff über SSH-Tunnel beide Ports weiterleiten und dieselben localhost-URLs verwenden. `localhost` und `127.0.0.1` sind unterschiedliche Origins: Die Browser-URL muss exakt zu `ADMIN_ORIGIN` passen.

## Images und CVE-Strategie

| Aufgabe | Image-Referenz zur Auflösung | Bedeutung |
|---|---|---|
| Go-Builder/Testphase | `cgr.dev/chainguard/go:latest-dev` | Build-Werkzeuge, nicht im Runtime-Image |
| Backend-Runtime | `cgr.dev/chainguard/static:latest` | Minimaler Unterbau für das statische Go-Binary |
| NGINX-Runtime | `cgr.dev/chainguard/nginx:latest` | Minimaler NGINX-Unterbau |
| CI-Scanner | `aquasec/trivy:latest` | Scan-Werkzeug, nicht Teil der Anwendung |

Chainguard dokumentiert die distroless/minimalen Images unter [NGINX](https://images.chainguard.dev/directory/image/nginx/overview), [static](https://images.chainguard.dev/directory/image/static/overview) und [Go](https://images.chainguard.dev/directory/image/go/overview). Verfügbarkeit und Zugang hängen vom Registry-Katalog ab. Bei einem Pull-/Berechtigungsfehler keine unüberprüften Alternativen automatisch verwenden: Registry-Anmeldung bzw. eigene freigegebene Image-Referenzen einrichten.

`lock-images.sh` fragt die echten Registry-Digests mit Docker Buildx ab und schreibt `images.lock.env`. Es gibt absichtlich keine erfundenen oder historisch geratenen SHA-256-Werte im Paket. `compose.sh` lädt zuerst `.env`, danach die Lock-Datei: Image-Digests haben Vorrang. Bei privater Registry die Referenzen im Lock-Skript ändern und erneut auflösen.

Nach dem Build:

```bash
./scripts/test.sh
./scripts/smoke.sh
./scripts/scan.sh
```

Der Scan exportiert beide endgültigen Runtime-Images als temporäre Archive. Trivy scannt diese ohne Zugriff auf den Docker-Socket. JSON-Berichte und CycloneDX-SBOMs landen in `artifacts/`. Die temporären Image-Archive werden bei normalem Abschluss entfernt. Scanner und Images sind im Lock per Digest fixiert; die CVE-Datenbank wird aktuell geladen.

Default-Gate: jeder gemeldete CVE-Befund in `UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL` blockiert, inklusive Befunden ohne verfügbaren Fix. Alternativ für einen ausdrücklich weniger strengen Gate:

```bash
TRIVY_SEVERITY=HIGH,CRITICAL ./scripts/scan.sh
```

Das ist dann kein Null-CVE-Gate. Ein leerer Report ist nur ein Scanbefund für genau diesen Digest, diese Plattform, diesen Scanner und den verwendeten Datenbankstand, keine Garantie dauerhafter CVE-Freiheit. Die Go-Standardbibliothek im Binary muss ebenfalls aktuell bleiben. Build-Images/CI-Runner zusätzlich prüfen; `scan.sh` konzentriert sich auf die beiden ausgelieferten Runtime-Images. SBOM-Erstellung ersetzt keine Signatur, Provenance oder Admission-Policy.

Beide Anwendungscontainer laufen mit UID/GID 65532, read-only Rootfilesystem, entfernten Linux-Capabilities und `no-new-privileges`. Nur das Backend-Datenvolume und NGINX `/tmp` sind beschreibbar. Ressourcen-, PID- und Loglimits sind gesetzt.

## HTTPS und Serverbetrieb

Basic Authentication außerhalb von localhost ausschließlich über HTTPS verwenden. Das Paket selbst enthält keine Zertifikatsverwaltung und keine automatisierte TLS-Terminierung.

Empfohlen sind getrennte DNS-Namen hinter einem bestehenden HTTPS-Reverse-Proxy:

```dotenv
ADMIN_ORIGIN=https://html-admin.example.test
PUBLIC_URL=https://pages.example.test
BIND_ADDRESS=127.0.0.1
```

Den Reverse-Proxy für `html-admin.example.test` auf `127.0.0.1:8080` und für `pages.example.test` auf `127.0.0.1:8081` konfigurieren. `Origin` und `Authorization` unverändert weiterreichen. TLS außen terminieren; Backend-Verbindung nur lokal/vertrauenswürdig betreiben. HSTS am TLS-Proxy setzen. Die Verwaltungs-Origin idealerweise zusätzlich auf VPN/interne Netze beschränken. Bei vorgeschaltetem Proxy sieht das eingebaute NGINX dessen IP; per-Nutzer-Rate-Limits dann am äußeren Proxy konfigurieren.

Anschließend:

```bash
./scripts/compose.sh up -d --force-recreate
```

`BIND_ADDRESS=0.0.0.0` macht die HTTP-Ports im Netz erreichbar. Nur mit bewusst konfiguriertem TLS-Vorschaltproxy und Firewall einsetzen; nicht die Basic-Auth-Verwaltung unverschlüsselt im Internet öffnen.

Alle veröffentlichten Seiten sind anonym lesbar. Zufällige IDs sind keine Zugriffskontrolle. Keine vertraulichen Dokumente veröffentlichen. Wer private Dokumente benötigt, muss auch den öffentlichen Listener durch eigene Authentifizierung schützen.

## HTML-Kompatibilität

Unterstützt werden einzelne UTF-8-Dokumente mit `<!doctype html>` oder `<html>`. Externe Dateien, relative Bilder, CSS-Dateien, JavaScript-Bundles und ZIP-Pakete werden nicht mit hochgeladen. Für den Default Inline-CSS und optional eingebettete `data:`-Bilder nutzen. Seiten mit JavaScript/Formularen/externer CDN-Abhängigkeit funktionieren im Default bewusst nicht vollständig.

Falls vertrauenswürdige Inhalte zwingend JavaScript benötigen, die Public-CSP in `nginx/nginx.conf` bewusst anpassen, z. B. nur `sandbox allow-scripts` und eine passend begrenzte `script-src`. Nicht pauschal `allow-same-origin`, Formularzugriffe, externe Zielhosts oder CORS freischalten. Solche Änderungen verändern das Sicherheitsmodell und benötigen eigene Tests. Keine CSP-Lockerung auf der Verwaltungs-Origin vornehmen. Upload-Sicherheit: [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html).

## Konfiguration

| Variable | Default | Hinweis |
|---|---|---|
| `ADMIN_USER` | `admin` | Gemeinsamer Administrator, keine Rollen |
| `ADMIN_ORIGIN` | `http://localhost:8080` | Exakt eine Origin, ohne abschließenden Slash |
| `PUBLIC_URL` | `http://localhost:8081` | Separater Ursprung, ohne Pfad/Slash |
| `MAX_UPLOAD_BYTES` | `5242880` | 5 MiB; bei Erhöhung NGINX-`client_max_body_size` ebenfalls anpassen |
| `MAX_PAGES` | `500` | Maximale Anzahl aktuell publizierter Seiten |
| `BIND_ADDRESS` | `127.0.0.1` | Host-Bind-Adresse der beiden NGINX-Ports |

Passwort rotieren: eine neue Datei mit mindestens 16 Byte erzeugen, ihre beschriebenen Leserechte wieder setzen und `uploader` mit `--force-recreate` neu starten. Das Backend liest das Passwort nur beim Start. Bestehende Browser-Anmeldedaten ggf. mit neuer Sitzung/Privatfenster ersetzen. Keine Passwörter in Prozessargumenten für dauerhafte Dienste verwenden; die Smoke-Test-curl-Aufrufe sind nur ein lokaler Test.

## API

Authentifizierung: HTTP Basic. Bei POST/DELETE zusätzlich `Origin: <ADMIN_ORIGIN>`; auch CLI-Clients müssen diesen Header setzen. Eine Origin-Prüfung ist CSRF-Schutz im Browser, keine Ersatz-Authentifizierung: CLI-Clients können Header frei setzen.

| Methode | Pfad | Funktion |
|---|---|---|
| GET | `/api/pages` | JSON-Liste mit Metadaten und öffentlicher URL |
| POST | `/api/upload` | Roher HTML-Body; Header `X-File-Name` mit URL-kodiertem Dateinamen |
| DELETE | `/api/pages/<id>` | Veröffentlichte Datei und Metadaten löschen |
| GET | `/healthz` | Backend-Healthcheck ohne Auth; am NGINX-Admin-Listener gesperrt |

`POST /api/upload` verwendet bewusst keinen Multipart-Parser. Die Browseroberfläche setzt den Header automatisch. Fehler liefern JSON mit einem `error`-Feld. Typische Statuscodes: 201 erstellt, 204 gelöscht, 400 ungültige Eingabe, 401 Auth fehlt/falsch, 403 falsche Origin, 409 Seitenlimit, 413 Upload zu groß, 429 anderer Import läuft/Rate-Limit, 500 Speicherfehler. NGINX-Fehler können statt JSON eine generische HTML-Fehlerseite liefern.

## Git-Struktur und Pflege

```text
nginx-html-manager/
├── cmd/server/
│   ├── main.go                  # API, Auth, Storage, eingebettete Oberfläche
│   ├── main_test.go             # 14 Go-Tests
│   └── web/                    # HTML, CSS, JavaScript
├── nginx/                      # NGINX-Dockerfile und Konfiguration
├── scripts/                    # Setup, Digest-Lock, Compose, Test, Scan, Smoke
├── tests/ui.test.cjs            # 6 UI-Logiktests
├── examples/demo.html
├── .github/workflows/ci.yml
├── compose.yaml
├── Dockerfile
├── go.mod
├── .env.example
├── images.lock.env             # Nach make lock erzeugen und committen
├── renovate.json
├── Makefile
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
├── VALIDATION.md
├── LICENSE
└── VERSION
```

Ein lokales Repository anlegen; noch kein Remote-Repository und kein Commit sind im Paket vorweggenommen:

```bash
git init -b main
git add .
git status --short             # Keine Secrets/Uploads/.env im Index!
git commit -m "Initial HTML manager 1.0.0"
git tag -a v1.0.0 -m "HTML manager 1.0.0"
```

Vor dem ersten Commit `make lock` ausführen. `images.lock.env` ist bewusst nicht ignoriert. Auf GitHub ist die CI absichtlich rot, wenn keine echte Digest-Lock-Datei committet ist. Im Repo Branch Protection und verpflichtende CI-Prüfung aktivieren.

Die GitHub-Actions-Pipeline baut die Runtime-Images, führt Go-Tests und `go vet`, UI-Logiktests, NGINX-Syntaxprüfung sowie einen laufenden Stack-Smoke-Test aus und prüft beide finalen Images mit Trivy. Berichte werden als CI-Artefakte gespeichert. Zusätzlich läuft sie montags. Sie veröffentlicht keine Images und deployt nicht automatisch. `actions/checkout@v4` und `actions/upload-artifact@v4` sind bequem lesbare Starter-Refs: Für eine produktive Supply Chain auf geprüfte Commit-SHAs pinnen.

Renovate unterstützt die Image-Referenzen in `images.lock.env`. Nach einem Digest-Update erst bauen, testen und scannen, dann freigeben. Bei gewünschter Harbor-/Air-Gap-Nutzung die Basen und Scanner spiegeln und Lock-Referenzen auf die Mirror-Registry umstellen. Die Trivy-Datenbank muss dann ebenfalls kontrolliert offline bereitgestellt werden; das ist kein Bestandteil dieses Pakets.

## Betrieb und Grenzen

```bash
make logs
make down                    # Daten bleiben unter data/ erhalten
make lock build up test smoke scan
```

Nur eine Backend-Instanz pro Datenverzeichnis betreiben. Der Mutex koordiniert genau einen Prozess; es gibt keine verteilte Sperre und keine Multi-Replica-Storage-Semantik. Dateiinhalt wird erst nach vollständigem Schreiben per Rename veröffentlicht, Metadaten getrennt davon. Das ist keine vollständig transaktionale Datenbank; Stromausfall oder Prozessabbruch können verwaiste Metadaten hinterlassen. Die Liste überspringt Metadaten ohne öffentliche Datei. Verwaiste Daten nur nach Backup und manueller Prüfung entfernen.

Backups: Stack stoppen und `data/` konsistent sichern; Passwortdatei getrennt und verschlüsselt sichern. Wiederherstellung mit passender UID/GID 65532. Für produktiven Betrieb zusätzlich Host-Disk-Quotas, Monitoring und externe Backup-Retention vorsehen. Das Seiten-/Dateilimit begrenzt Nutzdaten, ist keine Host-Quota.

Der Gesamtgrößenrahmen bei Defaults ist etwa 2,5 GiB Nutzdaten plus Metadaten. Das Backend hat HTTP-Zeitlimits; sehr langsame Uploads können abgebrochen werden. Bei höherer Parallelität gibt es 429 statt unbeschränkter Upload-Speicherbelegung.

## Nachweisstand

Die in der Erstellung tatsächlich ausgeführten Prüfungen stehen in `VALIDATION.md`. Go, Docker, NGINX und Registry-/Trivy-Zugriff waren in der Erstellungsumgebung nicht verfügbar. Deshalb sind weder erfolgreiche Go-Kompilierung, Containerstarts, NGINX-Syntax noch ein aktueller CVE-freier Digest nachgewiesen. Diese Prüfungen sind in den Build/Test/Scan-Skripten und der CI vorgesehen und müssen auf dem Zielsystem vor Produktivbetrieb erfolgreich laufen.
