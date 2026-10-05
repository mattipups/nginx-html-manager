# Projektpflege

1. Feature-Branch anlegen: `git switch -c feat/meine-aenderung`.
2. Code und Tests anpassen. Go lokal mit `gofmt -w cmd/server/*.go` formatieren.
3. `make build test up smoke scan` ausführen. Docker/Buildx, Node.js, Python 3, curl und Internetzugriff für Images und Trivy-Datenbank erforderlich.
4. Änderungen in CHANGELOG.md dokumentieren; keine Secrets und keine Uploads committen.
5. Pull Request öffnen. Erfolgreiche CI und mindestens ein Review verlangen.
6. Version in VERSION aktualisieren und annotiertes Release-Tag erstellen.

`images.lock.env` gehört in Git. Nach `make lock` sowohl Images bauen als auch testen und scannen. Renovate erzeugt kontrollierbare Digest-Update-PRs. Wöchentliche CI-Scans prüfen unveränderte Images gegen neue CVE-Datenbanken.

Empfohlene Repository-Regeln: geschützter main-Branch, verpflichtender CI-Check, kein direktes Pushen nach main, keine automatischen Security-Ausnahmen. CI veröffentlicht absichtlich keine Images und führt kein automatisches Deployment aus.
