# Builder-Logo-Kit: original

Farben: Light #EE6E49; Dark #F69370. Der i-Punkt nutzt jeweils die Roboter-Akzentfarbe.

- logos/*-1400x700.png: transparentes Logo mit Slogan, für Website-Header und Dokumentation.
- logos/*-700x350.png: kleinere transparente Variante für Web und Präsentationen.
- icons/app-*-512.png und *-192.png: quadratisches App-/PWA-Icon.
- favicons/favicon.ico und favicon-16/32/48.png: Browser-Tab.
- social/og-*.png: Vorschau für Social Sharing (1200 x 630).
- web/manifest.webmanifest: Beispiel für PWA; Pfade nach dem Entpacken bei Bedarf anpassen.

Light-Grafiken auf hellen Oberflächen, Dark-Grafiken auf dunklen Oberflächen einsetzen. Logos sind transparent; Icons und Social-Previews haben absichtlich einen Hintergrund. Alle Grafiken sind Rasterdateien und keine vektorisierten SVGs.

## Bash-Skript
## Verwendung

```bash
chmod +x png-als-text.sh

# Erzeugt logo.png.txt:
./png-als-text.sh logo.png

# Alternativ mit eigenem Ausgabedateinamen:
./png-als-text.sh logo.png upload.txt
```

Lade anschließend die erzeugte TXT-Datei hier als Anhang hoch. Das Skript verändert das PNG nicht und überschreibt keine vorhandene Ausgabedatei.

## Zurückwandeln und prüfen

Unter Linux kannst du die Datei so rekonstruieren und überprüfen:

```bash
jq -r '.base64' logo.png.txt | base64 --decode > wiederhergestellt.png

expected="$(jq -r '.sha256' logo.png.txt)"
actual="$(sha256sum < wiederhergestellt.png)"
actual="${actual%% *}"

if [[ "$actual" == "$expected" ]]; then
    printf 'OK: Die rekonstruierte PNG entspricht dem Original.\n'
else
    printf 'FEHLER: Die Prüfsummen unterscheiden sich.\n' >&2
fi
```

Die Prüfung stellt sicher, dass die rekonstruierten Dateibytes mit dem verpackten Original übereinstimmen – einschließlich der darin enthaltenen Transparenz.
