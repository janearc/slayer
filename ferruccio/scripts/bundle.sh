#!/bin/sh
set -eu

# Package only the binary already stamped by Game. No host installation.
test -x bin/atelier
bundle='bin/Atelier Prototype.app/Contents'
mkdir -p "$bundle/MacOS" "$bundle/Resources"
cp bin/atelier "$bundle/MacOS/.atelier-next"
mv "$bundle/MacOS/.atelier-next" "$bundle/MacOS/atelier"
mkdir -p "$bundle/Resources/engine" "$bundle/Resources/examples"
cp -R bin/engine/. "$bundle/Resources/engine/"
cp examples/*.yaml "$bundle/Resources/examples/"
cat > "$bundle/MacOS/launch" <<'LAUNCH'
#!/bin/sh
runtime="$(dirname "$0")/../Resources"
exec "$(dirname "$0")/atelier" open --game "$runtime/engine/game5" --slayer "$runtime/engine/slayer" --game-home "$runtime/engine/gamehome" --examples "$runtime/examples"
LAUNCH
chmod 755 "$bundle/MacOS/launch"
cat > "$bundle/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>launch</string>
<key>CFBundleIdentifier</key><string>org.slayerscenes.atelier.prototype</string>
<key>CFBundleName</key><string>Atelier Prototype</string>
<key>CFBundleDisplayName</key><string>Atelier Prototype</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleVersion</key><string>1</string>
<key>CFBundleShortVersionString</key><string>0.1</string>
<key>NSHighResolutionCapable</key><true/>
<key>NSSupportsAutomaticGraphicsSwitching</key><true/>
</dict></plist>
PLIST
"$bundle/MacOS/atelier" --age
