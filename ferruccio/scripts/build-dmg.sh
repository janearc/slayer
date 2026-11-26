#!/bin/sh
# Assemble a portable image from this checkout's app, without mounting it.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root"
# the pinned game, as build-app.sh builds it, builds the app
mkdir -p bin
GOWORK=off go build -o bin/game github.com/janearc/game/cmd/game
"$root/bin/game" build ferruccio
out=bin/Ferruccio.dmg
if [ -e "$out" ]; then
    recorded=$(cat "$out.sha256" 2>/dev/null || true)
    actual=$(shasum -a 256 "$out" | cut -d ' ' -f 1)
    if [ -z "$recorded" ] || [ "$recorded" != "$actual" ]; then
        echo "Refusing to replace an image not made by this build: $out" >&2; exit 1
    fi
fi
tmp=$(mktemp -d bin/.ferruccio-dmg.XXXXXXXX)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
mkdir "$tmp/content"
cp -R bin/Ferruccio.app "$tmp/content/Ferruccio.app"
cp doc/applications/ferruccio/howto-run-dis.md "$tmp/content/howto-run-dis.md"
ln -s /Applications "$tmp/content/Applications"
# on GitHub's macOS runners, hdiutil create fails now and then with
# "Resource busy" while the system scans the new files; a second try
# after a pause gets through
attempt=1
until hdiutil create -quiet -ov -format UDZO -volname Ferruccio -srcfolder "$tmp/content" "$tmp/image.dmg"; do
    if [ "$attempt" -ge 3 ]; then
        echo "build-dmg: hdiutil create failed $attempt times" >&2; exit 1
    fi
    echo "build-dmg: hdiutil create failed on attempt $attempt; trying again in 10 s" >&2
    attempt=$((attempt + 1))
    sleep 10
done
shasum -a 256 "$tmp/image.dmg" | cut -d ' ' -f 1 > "$tmp/checksum"
mv -f "$tmp/image.dmg" "$out"
mv -f "$tmp/checksum" "$out.sha256"
printf 'Built %s\n' "$out"
