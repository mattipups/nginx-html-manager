# NGINX HTML Manager 1.0.1

Deutschsprachige Upload-Oberfläche mit Go-Backend und direkter NGINX-Auslieferung. Git-fähiges Projekt mit Docker Compose, Tests, CVE-Gate und SBOM-Erstellung.

## Versionshinweis

Version 1.0.1 ist die erneute Bereitstellung des im Gespräch beschriebenen v1.0-Funktionsumfangs. Da das alte Archiv nicht mehr verfügbar war, wurden die Dateien anhand des Gesprächs neu zusammengestellt. Es wird keine byteidentische Reproduktion von v1.0.0 zugesichert. OIDC oder neue Zusatzfunktionen sind nicht enthalten.

## Funktionsumfang

- Mehrere `.html`/`.htm`-Dateien per Auswahl oder Drag & Drop importieren, seriell mit Fortschrittsanzeige.
- Seiten auflisten, nach Dateiname filtern, öffnen, Link kopieren und nach Bestätigung löschen.
- Jede Datei bekommt eine zufällige ID; gleichnamige Uploads überschreiben bestehende Seiten nicht.
- Metadaten: Originalname, Größe, UTC-Zeitstempel, SHA-256 und öffentliche URL.
- Basic Authentication und Origin-Prüfung für Änderungen.
- Default: 5 MiB pro Datei, 500 publizierte Seiten, ein paralleler Import.
- Persistente Dateien unter `data/`; Metadaten liegen außerhalb des öffentlichen Mounts.

Nicht enthalten: OIDC, Benutzer-/Rollenverwaltung, Virenscanner, HTML-Sanitization, ZIP-/Asset-Upload, Bearbeitung veröffentlichter Seiten und Helm-Chart. Das Paket ist eine Docker-Compose-Installation, kein Kubernetes-Deployment.

## Architektur

```text
Browser -> localhost:8080 -> NGINX Admin -> Go-Backend :8080
                                              |
                                              v
                                    data/meta + data/public
Browser -> localhost:8081/pages/<id>.html -> NGINX Public
                                              |
                                              v
                                    data/public (read-only)
```

Administration und öffentliche Seiten verwenden getrennte Browser-Origins. Die öffentliche Origin enthält weder API noch Inhaltsverzeichnis. Der Go-Port wird nicht auf dem Host veröffentlicht. NGINX liest nur `data/public`, nicht Metadaten oder Secrets.

HTML ist aktiver Inhalt. Im Standard verhindert die öffentliche CSP JavaScript, Formulare, Frames und externe Ressourcen. Inline-CSS und `data:`-Bilder sind erlaubt. Die Formatprüfung ist kein Nachweis harmloser Inhalte. Nur vertrauenswürdige Administratoren dürfen veröffentlichen.

## Voraussetzungen

- Linux mit Docker Engine, Compose v2 und Buildx. Compose muss `--wait` und mehrere `--env-file`-Argumente unterstützen.
- Bash, OpenSSL, Python 3, curl, Git sowie sudo/root für die Verzeichnisinitialisierung.
- Internetzugriff auf Registries und Trivy-Datenbank; ggf. passende Anmeldung/Berechtigung für `cgr.dev`.
- Optional Node.js für UI-Logiktests; kein Node.js im Runtime-Image.
- Die Standardanleitung gilt für rootful Docker auf Linux. Rootless Docker, UID-Remapping, SELinux, Docker Desktop und NFS benötigen ggf. angepasste Mount-Rechte/Labels.

## Schnellstart

Nach dem Entpacken:

```bash
cd nginx-html-manager
chmod +x scripts/*.sh
./scripts/init.sh
./scripts/lock-images.sh
./scripts/compose.sh up -d --build --wait
./scripts/compose.sh ps
cat secrets/admin_password.txt
```

Verwaltung: [http://localhost:8080](http://localhost:8080), Benutzer `admin`, Passwort aus der Datei. Als ersten Upload `examples/demo.html` verwenden.

Öffentliche URLs haben das Format `http://localhost:8081/pages/<32-stellige-id>.html`. Die öffentliche Root-URL liefert absichtlich 404.

`init.sh` erzeugt `.env`, ein zufälliges 64-Zeichen-Hex-Passwort und Datenverzeichnisse für UID/GID 65532. Compose-Dateisecrets behalten Host-Dateirechte. Deshalb hat die Passwortdatei Leserechte; der Host-Elternordner `secrets/` ist mit Modus 0700 geschützt. Passwort, Konfiguration und Uploads gehören nicht in Git.

Ports binden standardmäßig nur an `127.0.0.1`. Bei SSH-Tunnel beide Ports weiterleiten und im Browser dieselben localhost-URLs verwenden. `localhost` und `127.0.0.1` sind unterschiedliche Origins: Die Browser-URL muss exakt zu `ADMIN_ORIGIN` passen.

## Images und CVE-Prüfung

| Aufgabe | Default zur Digest-Auflösung |
|---|---|
| Go-Builder/Testphase | `cgr.dev/chainguard/go:latest-dev` |
| Backend-Runtime | `cgr.dev/chainguard/static:latest` |
| NGINX-Runtime | `cgr.dev/chainguard/nginx:latest` |
| CI-Scanner | `aquasec/trivy:latest` |

Das statische Go-Backend nutzt nur die Standardbibliothek und keine externen Go-Module. Builder und Scanner werden nicht Teil der ausgelieferten Anwendung. Die Runtime-Images sind auf minimale Basen ausgelegt.

`lock-images.sh` liest echte Registry-Digests via Docker Buildx und schreibt `images.lock.env`. Das Paket enthält bewusst keine erfundenen oder historisch geratenen Digests. `compose.sh` lädt zuerst `.env`, danach die Lock-Datei; Digest-Referenzen haben Vorrang. Die Lock-Datei nach Prüfung in Git committen.

Bei Registry-Zugriffsproblemen die passende Anmeldung/Katalogberechtigung einrichten. Für eine eigene Registry/Mirror die Referenzen im Lock-Skript ändern und neu auflösen; keine ungeprüften Alternativen automatisch einsetzen.

Nach dem Build und Start:

```bash
./scripts/test.sh
./scripts/smoke.sh
./scripts/scan.sh
```

Trivy scannt gespeicherte Archive beider endgültigen Runtime-Images ohne Docker-Socket-Zugriff. JSON-Berichte und CycloneDX-SBOMs landen in `artifacts/`. Archive werden bei regulärem Abschluss entfernt. Die CVE-Datenbank wird aktuell geladen; Scanner und Images sind per Digest fixiert.

Default-Gate: Jeder gemeldete Befund in `UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL` blockiert, auch ohne verfügbaren Fix. Ein bewusst weniger strenger Gate ist möglich:

```bash
TRIVY_SEVERITY=HIGH,CRITICAL ./scripts/scan.sh
```

Dieser alternative Gate ist kein Null-CVE-Gate. Ein leerer Report belegt nur den Scanstand für den konkreten Digest, die Plattform, den Scanner und seine Datenbank. Dauerhafte oder allgemeine CVE-Freiheit wird nicht zugesichert. Go-Standardbibliothek, Build-Images und CI-Werkzeuge ebenfalls aktuell halten und zusätzlich absichern. SBOMs ersetzen keine Signatur oder Provenance.

Beide Runtime-Container laufen als UID/GID 65532 mit read-only Rootfilesystem, entfernten Capabilities und `no-new-privileges`. Schreibbar sind Backend-Daten und NGINX `/tmp`. Ressourcen-, PID- und Loglimits sind konfiguriert.

## HTTPS und Serverbetrieb

Basic Authentication außerhalb von localhost ausschließlich über HTTPS nutzen. Das Paket enthält keine automatische TLS-Terminierung/Zertifikatsverwaltung.

Empfohlen: Zwei DNS-Namen hinter einem bestehenden HTTPS-Reverse-Proxy:

```dotenv
ADMIN_ORIGIN=https://html-admin.example.test
PUBLIC_URL=https://pages.example.test
BIND_ADDRESS=127.0.0.1
```

Den Admin-VHost an `127.0.0.1:8080`, den Public-VHost an `127.0.0.1:8081` weiterleiten. `Origin` und `Authorization` unverändert weitergeben. TLS und HSTS am äußeren Proxy konfigurieren. Admin-Zugang möglichst auf VPN/interne Netze beschränken. Per-Client-Rate-Limits am äußeren Proxy konfigurieren, da das interne NGINX dessen IP sieht.

```bash
./scripts/compose.sh up -d --force-recreate
```

`BIND_ADDRESS=0.0.0.0` macht die HTTP-Ports im Netzwerk erreichbar. Nicht ohne bewusst eingerichteten TLS-Vorschaltproxy und Firewall nutzen. Alle veröffentlichten Seiten sind anonym lesbar; zufällige IDs sind keine Zugriffskontrolle. Keine vertraulichen Dokumente veröffentlichen. Für private Seiten auch den öffentlichen Listener authentifizieren.

## HTML-Kompatibilität

Unterstützt werden einzelne UTF-8-Dateien mit `<!doctype html>` oder `<html>`. CSS, Bilder, JavaScript-Bundles und ZIP-Inhalte werden nicht separat importiert. Im Default Inline-CSS und eingebettete `data:`-Bilder verwenden.

JavaScript, Formulare und externe CDN-Ressourcen funktionieren mit der Standard-CSP bewusst nicht. Eine Lockerung der Public-CSP in `nginx/nginx.conf` ist eine sicherheitsrelevante Änderung und erfordert eigene Tests. Nicht pauschal `allow-same-origin`, Formularzugriffe oder externe Ziele freischalten; die Admin-CSP nicht lockern. Die Anwendung bereinigt HTML nicht.

## Konfiguration

| Variable | Default | Hinweis |
|---|---|---|
| `ADMIN_USER` | `admin` | Ein gemeinsamer Administrator, keine Rollen |
| `ADMIN_ORIGIN` | `http://localhost:8080` | Exakt eine Origin ohne abschließenden Slash |
| `PUBLIC_URL` | `http://localhost:8081` | Andere Origin, ohne Pfad/Slash |
| `MAX_UPLOAD_BYTES` | `5242880` | Bei Erhöhung auch NGINX `client_max_body_size` anpassen |
| `MAX_PAGES` | `500` | Maximale Zahl aktiver Veröffentlichungen |
| `BIND_ADDRESS` | `127.0.0.1` | Host-Bind-Adresse |

Passwortwechsel: Datei durch ein neues Secret mit mindestens 16 Byte ersetzen, Leserechte wieder setzen und den Uploader mit `--force-recreate` neu starten. Passwort wird nur beim Start eingelesen. Alte Browser-Anmeldedaten ggf. durch ein Privatfenster/neue Sitzung ersetzen.

## API

HTTP Basic für Verwaltungsrouten. Änderungen erfordern zusätzlich `Origin: <ADMIN_ORIGIN>`, auch bei CLI-Clients. Die Origin-Prüfung schützt Browser vor CSRF, ersetzt aber keine Authentifizierung; CLI-Clients können Header frei setzen.

| Methode | Pfad | Funktion |
|---|---|---|
| GET | `/api/pages` | Metadaten und URLs als JSON |
| POST | `/api/upload` | Roher HTML-Body, `X-File-Name` mit URL-kodiertem Dateinamen |
| DELETE | `/api/pages/<id>` | Datei und Metadaten löschen |
| GET | `/healthz` | Interner Backend-Healthcheck; am Admin-NGINX gesperrt |

Upload verwendet keinen Multipart-Parser. Die UI setzt den Header automatisch. API-Fehler liefern JSON mit `error`; NGINX-Fehler können eine HTML-Fehlerseite liefern. Statuscodes: 201 erstellt, 204 gelöscht, 400 Eingabe ungültig, 401 Auth fehlt/falsch, 403 Origin falsch, 409 Seitenlimit, 413 Größe, 429 paralleler Import/Rate-Limit, 500 Speicherfehler.

## Git-Struktur

```text
nginx-html-manager/
├── cmd/server/
│   ├── main.go
│   ├── main_test.go              # 14 Go-Tests
│   └── web/                     # Oberfläche
├── nginx/
├── scripts/
├── tests/ui.test.cjs             # 6 UI-Logiktests
├── examples/demo.html
├── .github/workflows/ci.yml
├── compose.yaml
├── Dockerfile
├── go.mod
├── .env.example
├── images.lock.env              # Nach Digest-Auflösung erzeugt
├── renovate.json
├── Makefile
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
├── VALIDATION.md
├── validation-results.json
├── LICENSE
└── VERSION
```

Repository nach erfolgreichem Lock/Test/Scan initialisieren:

```bash
git init -b main
git add .
git status --short             # Keine Secrets/Uploads/.env im Index!
git commit -m "Initial HTML manager 1.0.1"
git tag -a v1.0.1 -m "HTML manager 1.0.1"
```

Das Paket enthält kein Remote-Repository und keinen bereits ausgeführten Commit. `images.lock.env` ist absichtlich versionierbar; ohne committete echte Lock-Datei blockiert die CI. Branch Protection und verpflichtende CI-Prüfung im eigenen Repository aktivieren.

Die GitHub-Actions-Pipeline baut, führt Go-Tests und `go vet`, UI-Tests, NGINX-Syntaxprüfung, Stack-Smoke-Test und Trivy-Gates aus. Berichte werden als Artefakte gespeichert; zusätzlich wöchentlicher Scan montags. Kein automatischer Image-Push oder Deployment. Die Starter-Refs `actions/checkout@v4` und `actions/upload-artifact@v4` vor Produktion auf geprüfte Commit-SHAs pinnen.

Renovate berücksichtigt die Lock-Datei. Digest-Updates erst nach Build, Tests und Scan freigeben. Für Harbor/Air-Gap Images und Scanner spiegeln; die Trivy-Datenbank ebenfalls kontrolliert offline bereitstellen. Eine vollständige Air-Gap-Anleitung ist nicht enthalten.

## Betrieb und Grenzen

```bash
make logs
make down                      # Daten bleiben erhalten
make lock build up test smoke scan
```

Genau eine Backend-Instanz pro Datenverzeichnis betreiben. Prozesslokale Sperren sind kein verteiltes Locking. HTML wird nach vollständigem Schreiben per Rename veröffentlicht; Metadaten werden separat gespeichert. Das ist keine vollständig transaktionale Datenbank. Ein Abbruch kann verwaiste Metadaten hinterlassen, die ohne öffentliche Datei nicht aufgelistet werden. Bereinigung nur nach Backup und Prüfung.

Stack für konsistentes Datenbackup stoppen, `data/` sichern und Secret separat verschlüsselt sichern. Bei Restore UID/GID 65532 beachten. Host-Disk-Quotas, Monitoring und Retention ergänzen. Default maximal etwa 2,5 GiB Nutzdaten plus Metadaten; Anwendungslimits ersetzen keine Host-Quota. Sehr langsame Uploads können durch HTTP-Zeitlimits abbrechen.

## Nachweisstand

Tatsächlich ausgeführte Prüfungen stehen in `VALIDATION.md` und `validation-results.json`. Go, Docker und NGINX waren bei der Erstellung nicht installiert. Daher sind erfolgreiche Go-Kompilierung, Containerlauf, NGINX-Syntax und aktuelle CVE-freie Images nicht nachgewiesen. Bereitgestellte Prüfskripte/CI auf dem Zielsystem vor Produktivbetrieb ausführen.
