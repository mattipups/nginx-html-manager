#!/usr/bin/env bash
set -euo pipefail

# Verwendung:
#   ./png-als-text.sh logo.png
#   ./png-als-text.sh logo.png upload.txt

die() {
    printf 'Fehler: %s\n' "$*" >&2
    exit 1
}

if (( $# < 1 || $# > 2 )); then
    printf 'Verwendung: %s INPUT.png [OUTPUT.txt]\n' "$0" >&2
    exit 1
fi

input="$1"
output="${2:-${input}.txt}"

[[ -f "$input" && -r "$input" ]] \
    || die "Eingabedatei ist keine lesbare reguläre Datei: $input"

[[ ! -e "$output" && ! -L "$output" ]] \
    || die "Ausgabedatei existiert bereits: $output"

for cmd in base64 jq od tr wc mktemp; do
    command -v "$cmd" >/dev/null 2>&1 \
        || die "Benötigtes Programm fehlt: $cmd"
done

# PNG-Signatur prüfen, nicht nur die Dateiendung.
signature="$(LC_ALL=C od -An -tx1 -N8 < "$input" | tr -d '[:space:]')"
[[ "$signature" == "89504e470d0a1a0a" ]] \
    || die "Die Eingabedatei besitzt keine gültige PNG-Signatur."

if command -v sha256sum >/dev/null 2>&1; then
    checksum="$(sha256sum < "$input")"
elif command -v shasum >/dev/null 2>&1; then
    checksum="$(shasum -a 256 < "$input")"
else
    die "Weder sha256sum noch shasum gefunden."
fi
checksum="${checksum%% *}"

size="$(wc -c < "$input" | tr -d '[:space:]')"
filename="${input##*/}"

# Temporäre Datei im Zielverzeichnis; bei Fehlern aufräumen.
tmp="$(mktemp "${output}.tmp.XXXXXX")"
trap 'rm -f -- "$tmp"' EXIT

# Streaming-Ausgabe: Base64 wird nicht als großes Shell-Argument übergeben.
{
    printf '{\n'
    printf '  "filename": %s,\n' \
        "$(printf '%s' "$filename" | jq -Rs '.')"
    printf '  "mime_type": "image/png",\n'
    printf '  "byte_length": %s,\n' "$size"
    printf '  "sha256": "%s",\n' "$checksum"
    printf '  "base64": "'
    base64 < "$input" | tr -d '\r\n'
    printf '"\n}\n'
} > "$tmp"

# Ohne Überschreiben einer inzwischen angelegten Zieldatei veröffentlichen.
ln -- "$tmp" "$output" \
    || die "Ausgabedatei konnte nicht angelegt werden: $output"

printf 'Erstellt: %s\n' "$output"
printf 'Originalgröße: %s Bytes\n' "$size"
printf 'SHA-256: %s\n' "$checksum"
