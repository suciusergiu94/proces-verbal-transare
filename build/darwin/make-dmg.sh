#!/usr/bin/env bash
#
# Împachetează bundle-ul .app într-un fișier .dmg cu simbolic link către
# /Applications, folosind doar hdiutil (fără dependențe externe).
#
#   ./make-dmg.sh <cale/catre/App.app> <cale/catre/iesire.dmg> "<Nume Volum>"
#
# Semnare opțională (dacă nu sunt setate, aplicația rămâne nesemnată):
#   APPLE_SIGNING_IDENTITY  — identitate "Developer ID Application: ..."
#   APPLE_NOTARY_PROFILE    — profil salvat cu `xcrun notarytool store-credentials`
#
set -euo pipefail

APP_PATH=${1:?Lipsește calea către .app}
DMG_PATH=${2:?Lipsește calea către .dmg}
VOLUME_NAME=${3:?Lipsește numele volumului}

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
READ_ME="$SCRIPT_DIR/CITEȘTE-MĂ.txt"

[ -d "$APP_PATH" ] || { echo "Eroare: nu există bundle-ul $APP_PATH" >&2; exit 1; }

STAGE_DIR=$(mktemp -d)
trap 'rm -rf "$STAGE_DIR"' EXIT

echo "==> Pregătesc conținutul imaginii"
cp -R "$APP_PATH" "$STAGE_DIR/"
ln -s /Applications "$STAGE_DIR/Applications"
[ -f "$READ_ME" ] && cp "$READ_ME" "$STAGE_DIR/"

STAGED_APP="$STAGE_DIR/$(basename "$APP_PATH")"

if [ -n "${APPLE_SIGNING_IDENTITY:-}" ]; then
  echo "==> Semnez aplicația cu identitatea $APPLE_SIGNING_IDENTITY"
  codesign --force --deep --timestamp --options runtime \
    --sign "$APPLE_SIGNING_IDENTITY" "$STAGED_APP"
  codesign --verify --strict --verbose=2 "$STAGED_APP"
else
  echo "==> Fără identitate de semnare (APPLE_SIGNING_IDENTITY nesetat): aplicație nesemnată"
fi

echo "==> Creez $DMG_PATH"
mkdir -p "$(dirname "$DMG_PATH")"
rm -f "$DMG_PATH"
hdiutil create \
  -volname "$VOLUME_NAME" \
  -srcfolder "$STAGE_DIR" \
  -fs HFS+ \
  -format UDZO \
  -imagekey zlib-level=9 \
  -quiet \
  "$DMG_PATH"

if [ -n "${APPLE_SIGNING_IDENTITY:-}" ]; then
  codesign --force --timestamp --sign "$APPLE_SIGNING_IDENTITY" "$DMG_PATH"
fi

if [ -n "${APPLE_NOTARY_PROFILE:-}" ]; then
  echo "==> Trimit imaginea la notarizare (profil $APPLE_NOTARY_PROFILE)"
  xcrun notarytool submit "$DMG_PATH" \
    --keychain-profile "$APPLE_NOTARY_PROFILE" --wait
  xcrun stapler staple "$DMG_PATH"
  echo "==> Notarizare finalizată"
else
  echo "==> Fără notarizare (APPLE_NOTARY_PROFILE nesetat)"
fi

echo "==> Gata: $DMG_PATH"
