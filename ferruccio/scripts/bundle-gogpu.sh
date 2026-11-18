#!/bin/sh
# A separate live-viewport candidate, using the shared renderer and supplied starter.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root"
game=${GAME_BIN:-game5}
case "$game" in
    */*) game=$(cd "$(dirname "$game")" && pwd)/$(basename "$game") ;;
    *) game=$(command -v "$game") ;;
esac
slayer=${FERRUCCIO_SLAYER:-}
if [ -z "$slayer" ] && [ -n "${HOME:-}" ]; then
    slayer="$HOME/mesh/the-vorrh/renders/applications/slayer/bin/slayer"
fi
if [ -z "$slayer" ] || [ ! -x "$slayer" ]; then
    echo "Set FERRUCCIO_SLAYER to the external shared Slayer executable" >&2
    exit 1
fi
slayer=$(cd "$(dirname "$slayer")" && pwd)/$(basename "$slayer")
case "$slayer" in
    "$root"/*) echo "Live package requires the external shared Slayer renderer" >&2; exit 1 ;;
esac
starter=${FERRUCCIO_STARTER:?set FERRUCCIO_STARTER to the populated starting scene directory}
for resource in source.yaml preview.png receipt.json; do
    if [ ! -s "$starter/$resource" ]; then
        echo "Starter needs $resource" >&2
        exit 1
    fi
done
for program in bin/ferruccio "$game"; do
    if [ ! -x "$program" ]; then
        echo "Live package needs Game-built executable: $program" >&2
        exit 1
    fi
done
bundle_app=${FERRUCCIO_BUNDLE:-bin/Ferruccio-Live.app}
if [ -e "$bundle_app" ]; then
    echo "Candidate already exists; select a new FERRUCCIO_BUNDLE path" >&2
    exit 1
fi
bundle="$bundle_app/Contents"
mkdir -p "$bundle/MacOS" "$bundle/Resources/engine/gamehome/.config/game" "$bundle/Resources/examples" "$bundle/Resources/starter"
cp bin/ferruccio "$bundle/MacOS/ferruccio"
cp "$game" "$bundle/Resources/engine/game5"
printf 'author ferruccio\ngomaxprocs 2\n' > "$bundle/Resources/engine/gamehome/.config/game/config"
cp ferruccio/examples/*.yaml "$bundle/Resources/examples/"
mkdir -p "$bundle/Resources/examples/library"
cp sets/*.yaml "$bundle/Resources/examples/library/"
cp sources.yaml "$bundle/Resources/examples/sources.yaml"
cp "$starter/source.yaml" "$starter/preview.png" "$starter/receipt.json" "$bundle/Resources/starter/"
# shell_word prints its argument quoted for sh, one backslash before every
# byte outside the safe set, a quoted newline, and '' for the empty string.
shell_word() {
    if [ -z "$1" ]; then
        printf "''"
        return
    fi
    printf '%s\n' "$1" | LC_ALL=C sed -e ':join' -e '$!{N;bjoin' -e '}' \
        -e 's/[^A-Za-z0-9_.,:/@-]/\\&/g' -e "s/\\\\\\n/'\\n'/g"
}
# absolute_path prints its argument as an absolute path with dot segments,
# repeated slashes and any trailing slash removed. A leading tilde slash is
# the home directory.
absolute_path() {
    case $1 in
        "~") path=${HOME:-} ;;
        "~/"*) path="${HOME:-}/${1#"~/"}" ;;
        /*) path=$1 ;;
        *) path="$PWD/$1" ;;
    esac
    result=
    saved_ifs=$IFS
    IFS=/
    set -f
    for part in $path; do
        case $part in
            ''|.) ;;
            ..) result=${result%/*} ;;
            *) result="$result/$part" ;;
        esac
    done
    set +f
    IFS=$saved_ifs
    printf '%s' "${result:-/}"
}
evidence=${FERRUCCIO_EVIDENCE:-}
if [ -z "$evidence" ] && [ -n "${HOME:-}" ]; then
    evidence="$HOME/mesh/the-vorrh/renders/frames"
fi
if [ -n "$evidence" ]; then evidence=$(absolute_path "$evidence"); fi
{
    printf '#!/bin/sh\nset -eu\n'
    printf 'slayer_default=%s\n' "$(shell_word "$slayer")"
    printf 'agent_default=%s\n' "$(shell_word "${FERRUCCIO_AGENT-ferruccio}")"
    printf 'cache_default=%s\n' "$(shell_word "${FERRUCCIO_CACHE-${SLAYER_CACHE-}}")"
    printf 'evidence_default=%s\n' "$(shell_word "$evidence")"
} > "$bundle/MacOS/launch"
cat >> "$bundle/MacOS/launch" <<'LAUNCH'
runtime="$(dirname "$0")/../Resources"
exec "$(dirname "$0")/ferruccio" open --display "${FERRUCCIO_DISPLAY:-wire}" --preview-transport slayer --starter "$runtime/starter" --evidence "${FERRUCCIO_EVIDENCE:-$evidence_default}" --agent "${FERRUCCIO_AGENT:-$agent_default}" --game "$runtime/engine/game5" --slayer "${FERRUCCIO_SLAYER:-$slayer_default}" --game-home "$runtime/engine/gamehome" --examples "$runtime/examples" --cache "${FERRUCCIO_CACHE:-$cache_default}" --shelf "${FERRUCCIO_SHELF:-$HOME/mesh/the-vorrh/data/slayer/sources}" "$@"
LAUNCH
chmod 755 "$bundle/MacOS/launch"
cat > "$bundle/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>launch</string>
<key>CFBundleIdentifier</key><string>com.janearc.slayer.ferruccio.live</string>
<key>CFBundleName</key><string>Ferruccio Live</string>
<key>CFBundleDisplayName</key><string>Ferruccio Live</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleVersion</key><string>1</string>
<key>CFBundleShortVersionString</key><string>0.1.3</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
printf '%s\n' "$slayer" > "$bundle/Resources/shared-renderer.txt"
builds="$bundle/Resources/builds.txt"
: > "$builds"
for program in "$root/bin/ferruccio" "$slayer" "$game"; do
    mark=$(wc -c < "$builds")
    if ! "$program" --age >> "$builds" 2>&1; then
        tail -c +$((mark + 1)) "$builds" >&2
        exit 1
    fi
done
shasum -a 256 "$bundle/MacOS/ferruccio" "$bundle/Resources/engine/game5" "$bundle/Resources/starter/source.yaml" "$bundle/Resources/starter/preview.png" "$bundle/Resources/starter/receipt.json" > "$bundle/Resources/SHA256SUMS"
cat "$bundle/Resources/builds.txt"
printf 'Live candidate: %s\n' "$bundle_app"
