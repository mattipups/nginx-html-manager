#!/usr/bin/env bash
set -euo pipefail

echo "=== Starte Backup- und Wiederherstellungsprüfung ==="

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

DATA_DIR="$WORKDIR/data"
BACKUP_ARCHIVE="$WORKDIR/backup.tar.gz"
RESTORE_DIR="$WORKDIR/restore_data"

mkdir -p "$DATA_DIR/meta" "$DATA_DIR/public" "$RESTORE_DIR"

ID1="11111111111111111111111111111111"
CONTENT1="<!doctype html><html><head><title>App1</title></head><body>Inhalt 1</body></html>"
SHA1=$(printf '%s' "$CONTENT1" | sha256sum | awk '{print $1}')
SIZE1=${#CONTENT1}

echo -n "$CONTENT1" > "$DATA_DIR/public/${ID1}.html"
cat <<EOF > "$DATA_DIR/meta/${ID1}.json"
{
  "id": "${ID1}",
  "name": "app1.html",
  "size": ${SIZE1},
  "sha256": "${SHA1}",
  "created": "2026-10-06T20:00:00Z",
  "url": "http://localhost:8081/pages/argocd-builder.html",
  "canonicalUrl": "http://localhost:8081/pages/${ID1}.html",
  "slug": "argocd-builder",
  "profile": "interactive-api"
}
EOF

ln "$DATA_DIR/public/${ID1}.html" "$DATA_DIR/public/argocd-builder.html"

INODE_ORIG=$(stat -c %i "$DATA_DIR/public/${ID1}.html" 2>/dev/null || stat -f %i "$DATA_DIR/public/${ID1}.html")
INODE_SLUG=$(stat -c %i "$DATA_DIR/public/argocd-builder.html" 2>/dev/null || stat -f %i "$DATA_DIR/public/argocd-builder.html")

if [[ "$INODE_ORIG" != "$INODE_SLUG" ]]; then
  echo "FEHLER: Hardlink teilt nicht dieselbe Inode!" >&2
  exit 1
fi
echo "[OK] Hardlink teilt Inode $INODE_ORIG"

echo "Erstelle Backup-Archiv..."
tar -czf "$BACKUP_ARCHIVE" -C "$DATA_DIR" meta public

echo "Simuliere Desaster: Lösche Originaldaten..."
rm -rf "$DATA_DIR/meta" "$DATA_DIR/public"

echo "Führe Wiederherstellung (Restore) aus dem Backup durch..."
tar -xzf "$BACKUP_ARCHIVE" -C "$RESTORE_DIR"

echo "Validiere wiederhergestellten Datenbestand..."
RESTORED_ORIG="$RESTORE_DIR/public/${ID1}.html"
RESTORED_SLUG="$RESTORE_DIR/public/argocd-builder.html"
RESTORED_META="$RESTORE_DIR/meta/${ID1}.json"

[[ -f "$RESTORED_ORIG" ]] || { echo "FEHLER: Originaldatei fehlt nach Restore!" >&2; exit 1; }
[[ -f "$RESTORED_SLUG" ]] || { echo "FEHLER: Eigener Link (Slug) fehlt nach Restore!" >&2; exit 1; }
[[ -f "$RESTORED_META" ]] || { echo "FEHLER: Metadaten fehlen nach Restore!" >&2; exit 1; }

INODE_RESTORED_ORIG=$(stat -c %i "$RESTORED_ORIG" 2>/dev/null || stat -f %i "$RESTORED_ORIG")
INODE_RESTORED_SLUG=$(stat -c %i "$RESTORED_SLUG" 2>/dev/null || stat -f %i "$RESTORED_SLUG")

if [[ "$INODE_RESTORED_ORIG" != "$INODE_RESTORED_SLUG" ]]; then
  echo "FEHLER: Wiederhergestellter Link ist kein Hardlink mehr (Inodes: $INODE_RESTORED_ORIG vs $INODE_RESTORED_SLUG)!" >&2
  exit 1
fi
echo "[OK] Wiederhergestellter Hardlink teilt Inode: $INODE_RESTORED_ORIG"

RESTORED_HASH=$(sha256sum "$RESTORED_ORIG" | awk '{print $1}')
if [[ "$RESTORED_HASH" != "$SHA1" ]]; then
  echo "FEHLER: SHA-256 Prüfsumme stimmt nach Restore nicht überein!" >&2
  exit 1
fi
echo "[OK] SHA-256 Prüfsumme ($RESTORED_HASH) bytegenau validiert."

RESTORED_SIZE=$(wc -c < "$RESTORED_ORIG" | tr -d ' ')
if [[ "$RESTORED_SIZE" -ne "$SIZE1" ]]; then
  echo "FEHLER: Dateigröße weicht nach Restore ab!" >&2
  exit 1
fi
echo "[OK] Dateigröße ($RESTORED_SIZE Bytes) stimmt exakt überein."

echo "=== Backup- und Wiederherstellungstest erfolgreich abgeschlossen! ==="
