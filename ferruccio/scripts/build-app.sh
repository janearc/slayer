#!/bin/sh
# Build a private app in bin; never replace an app we did not make.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root"
# the game the app carries, and the one that builds its parts, is the one
# go.mod pins as a tool: never whichever game is first on the builder's PATH
mkdir -p bin
GOWORK=off go build -o bin/game github.com/janearc/game/cmd/game
game=$root/bin/game
export GAME_BIN="$game"
"$game" build ferruccio-binary
"$game" build slayer
"$game" build azazel-binary
out=bin/Ferruccio.app
if [ -e "$out" ] && [ ! -f "$out/Contents/Resources/ferruccio-build-owned" ]; then
    echo "Refusing to replace an app not made by this build: $out" >&2; exit 1
fi
tmp=$(mktemp -d bin/.ferruccio-app.XXXXXXXX)
cleanup() {
    if [ -e "$tmp/old.app" ] && [ ! -e "$out" ]; then mv "$tmp/old.app" "$out"; fi
    rm -rf "$tmp"
}
trap cleanup EXIT HUP INT TERM
app="$tmp/Ferruccio.app"
resources="$app/Contents/Resources"
mkdir -p "$app/Contents/MacOS" "$resources/engine" "$resources/examples/library" "$resources/Ferruccio.iconset"
for size in 16 32 128 256 512; do
    sips -z "$size" "$size" ferruccio/assets/icon.png --out "$resources/Ferruccio.iconset/icon_${size}x${size}.png" >/dev/null
    double=$((size * 2))
    sips -z "$double" "$double" ferruccio/assets/icon.png --out "$resources/Ferruccio.iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$resources/Ferruccio.iconset" -o "$resources/Ferruccio.icns"
rm -rf "$resources/Ferruccio.iconset"
cp bin/ferruccio bin/ferruccio-stage "$app/Contents/MacOS/"
cp bin/slayer bin/azazel "$resources/engine/"
cp "$game" "$resources/engine/game5"
cp ferruccio/examples/*.yaml "$resources/examples/"
cp sets/*.yaml "$resources/examples/library/"
for profile in \
    manufacture/plants/seasonal/profiles.yaml \
    manufacture/plants/conifers/conifer-profiles.yaml \
    manufacture/plants/ferns/fern-profiles.yaml \
    manufacture/animals/vertebrates/mammals/pinnipeds/profiles.yaml \
    manufacture/animals/vertebrates/mammals/whales/profiles.yaml; do
    mkdir -p "$resources/examples/$(dirname "$profile")"
    cp "$profile" "$resources/examples/$profile"
done
cp sources.yaml "$resources/examples/sources.yaml"
cat > "$app/Contents/MacOS/launch" <<'LAUNCH'
#!/bin/sh
set -eu
runtime="$(dirname "$0")/../Resources"
exec "$(dirname "$0")/ferruccio" open --preview-transport azazel --azazel "${FERRUCCIO_AZAZEL:-$runtime/engine/azazel}" --azazel-queue "${FERRUCCIO_AZAZEL_QUEUE:-}" --game "$runtime/engine/game5" --slayer "${FERRUCCIO_SLAYER:-$runtime/engine/slayer}" --examples "$runtime/examples" "$@"
LAUNCH
chmod 755 "$app/Contents/MacOS/launch"
cat > "$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>launch</string>
<key>CFBundleIdentifier</key><string>com.janearc.slayer.ferruccio</string>
<key>CFBundleName</key><string>Ferruccio</string>
<key>CFBundleDisplayName</key><string>Ferruccio</string>
<key>CFBundleIconFile</key><string>Ferruccio</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleVersion</key><string>1</string>
<key>CFBundleShortVersionString</key><string>0.1</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
: > "$resources/ferruccio-build-owned"
(cd "$app" && shasum -a 256 Contents/MacOS/ferruccio Contents/MacOS/ferruccio-stage Contents/Resources/engine/slayer Contents/Resources/engine/azazel Contents/Resources/engine/game5 > Contents/Resources/SHA256SUMS)
if [ -e "$out" ]; then mv "$out" "$tmp/old.app"; fi
if ! mv "$app" "$out"; then
    if [ -e "$tmp/old.app" ]; then mv "$tmp/old.app" "$out"; fi
    exit 1
fi
printf 'Built %s\n' "$out"
