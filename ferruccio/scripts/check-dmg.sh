#!/bin/sh
# Check a built Ferruccio image: the image itself, its recorded checksum,
# the app's checksums and study, the architecture and deployment target of
# each executable, their source revision and the pinned Game version.
# The Ferruccio workflow and tools/ship-ready both run this check.
#
#   check-dmg.sh IMAGE MOUNTPOINT
set -eu
if [ "$#" -ne 2 ]; then
    echo "usage: check-dmg.sh IMAGE MOUNTPOINT" >&2; exit 2
fi
image=$1
mount=$2
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
expected_revision=$(git -C "$root" rev-parse --verify HEAD)

# Read the embedded Go build info without running code from the image.
check_revision() {
    executable=$1
    build=$(go version -m "$app/$executable")
    revision=$(printf '%s\n' "$build" | awk -F= '$1 ~ /[[:space:]]vcs[.]revision$/ {print $2}')
    modified=$(printf '%s\n' "$build" | awk -F= '$1 ~ /[[:space:]]vcs[.]modified$/ {print $2}')
    if [ "$revision" != "$expected_revision" ]; then
        echo "check-dmg: $executable revision ${revision:-unknown}; expected $expected_revision" >&2
        return 1
    fi
    if [ "$modified" != false ]; then
        echo "check-dmg: $executable was not built from clean sources (vcs.modified=${modified:-unknown})" >&2
        return 1
    fi
    printf '%s: clean revision %s\n' "$executable" "$revision"
}

hdiutil verify "$image"
recorded=$(cat "$image.sha256")
actual=$(shasum -a 256 "$image" | cut -d ' ' -f 1)
if [ "$recorded" != "$actual" ]; then
    echo "check-dmg: $image.sha256 says $recorded, the image is $actual" >&2; exit 1
fi
printf '%s  %s\n' "$actual" "$image"

mkdir -p "$mount"
hdiutil attach -readonly -nobrowse -mountpoint "$mount" "$image" >/dev/null
trap 'hdiutil detach "$mount" >/dev/null' EXIT
app="$mount/Ferruccio.app"
minimum=13.0
declared=$(plutil -extract LSMinimumSystemVersion raw -o - "$app/Contents/Info.plist") || {
    echo "check-dmg: Info.plist must declare LSMinimumSystemVersion $minimum" >&2; exit 1
}
if [ "$declared" != "$minimum" ]; then
    echo "check-dmg: Info.plist declares macOS $declared; expected $minimum" >&2; exit 1
fi
(cd "$app" && shasum -a 256 -c Contents/Resources/SHA256SUMS)
notice_go=$(go version "$app/Contents/MacOS/ferruccio" | awk '{print $NF}')
if ! printf '%s\n' "$notice_go" | grep -Eq '^go[0-9]+\.[0-9]+(\.[0-9]+)?$'; then
    echo "check-dmg: cannot identify the app's Go release: ${notice_go:-unknown}" >&2
    exit 1
fi
notices=$(sh "$root/ferruccio/scripts/app-notices.sh" "$notice_go")
printf '%s\n' "$notices" | while IFS='|' read -r file source; do
    resource="Contents/Resources/licenses/$file"
    if [ ! -f "$app/$resource" ] || [ -L "$app/$resource" ] || ! cmp -s "$source" "$app/$resource"; then
        echo "check-dmg: missing or changed notice: $resource" >&2; exit 1
    fi
    sum=$(cd "$app" && shasum -a 256 "$resource")
    if ! grep -Fxq "$sum" "$app/Contents/Resources/SHA256SUMS"; then
        echo "check-dmg: SHA256SUMS does not list $resource" >&2; exit 1
    fi
done
for file in oakland-larson/set.yaml oakland-larson/two-metre-human.gltf \
    oakland-larson/two-metre-human.bin oakland-larson/LICENSE.txt \
    ferruccio-playground.yaml larson-camp.yaml larson-camp-visitor.glb larson-camp-visitor-LICENSE.txt \
    library/bioforms/adult-human-neutral.yaml library/bioforms/adult-human-reaching.yaml \
    library/bioforms/adult-human-male.yaml; do
    case "$file" in
        library/bioforms/*) source="$root/sets/${file#library/}" ;;
        *) source="$root/ferruccio/examples/$file" ;;
    esac

    resource="Contents/Resources/examples/$file"
    if [ ! -f "$app/$resource" ] || [ -L "$app/$resource" ]; then
        echo "check-dmg: app must contain a regular file at $resource" >&2; exit 1
    fi
    if ! cmp -s "$source" "$app/$resource"; then
        echo "check-dmg: app resource differs from source: $resource" >&2; exit 1
    fi
    sum=$(cd "$app" && shasum -a 256 "$resource")
    if ! grep -Fxq "$sum" "$app/Contents/Resources/SHA256SUMS"; then
        echo "check-dmg: SHA256SUMS does not list $resource" >&2; exit 1
    fi
done
for executable in Contents/MacOS/ferruccio Contents/MacOS/ferruccio-stage Contents/Resources/engine/slayer Contents/Resources/engine/azazel Contents/Resources/engine/game5; do
    built_go=$(go version "$app/$executable" | awk '{print $NF}')
    if [ "$built_go" != "$notice_go" ]; then
        echo "check-dmg: $executable uses ${built_go:-unknown}; Go notices are from $notice_go" >&2
        exit 1
    fi
    if ! grep -q "  $executable\$" "$app/Contents/Resources/SHA256SUMS"; then
        echo "check-dmg: SHA256SUMS does not list $executable" >&2; exit 1
    fi
    # the runner's lipo reads every argument after -verify_arch as an
    # architecture, so the file comes first
    lipo "$app/$executable" -verify_arch arm64
    linked=$(xcrun vtool -arch arm64 -show-build "$app/$executable" | awk '
        $1 == "platform" {macos = ($2 == "MACOS")}
        macos && $1 == "minos" {print $2}
    ')
    if ! awk -v linked="$linked" -v floor="$minimum" 'BEGIN {
        if (linked !~ /^[0-9]+([.][0-9]+){1,2}$/) exit 1
        split(linked, actual, ".")
        split(floor, supported, ".")
        for (part = 1; part <= 3; part++) {
            difference = actual[part] - supported[part]
            if (difference != 0) exit (difference > 0)
        }
    }'; then
        echo "check-dmg: $executable links for macOS ${linked:-unknown}; app declares $minimum" >&2
        exit 1
    fi
    printf '%s: arm64, linked for macOS %s\n' "$executable" "$linked"
    if [ "$executable" != Contents/Resources/engine/game5 ]; then
        check_revision "$executable"
    fi
done
pinned=$(cd "$root" && GOWORK=off go list -m -f '{{.Version}}' github.com/janearc/game)
if [ -z "$pinned" ]; then
    echo "check-dmg: go.mod names no version of github.com/janearc/game" >&2; exit 1
fi
bundled=$(go version -m "$app/Contents/Resources/engine/game5" | awk '$1 == "mod" && $2 == "github.com/janearc/game" { print $3 }')
if [ "$bundled" != "$pinned" ]; then
    echo "check-dmg: the bundled game is ${bundled:-unknown}, go.mod pins $pinned" >&2; exit 1
fi
printf 'engine/game5: github.com/janearc/game %s, as go.mod pins\n' "$bundled"
for module in github.com/gogpu/gogpu github.com/gogpu/gpucontext github.com/gogpu/gputypes; do
    pinned=$(cd "$root/ferruccio/native" && GOWORK=off go list -m -f '{{.Version}}' "$module")
    for executable in Contents/MacOS/ferruccio Contents/MacOS/ferruccio-stage; do
        bundled=$(go version -m "$app/$executable" |
            awk -v module="$module" '$1 == "dep" && $2 == module {print $3}')
        if [ -z "$pinned" ] || [ "$bundled" != "$pinned" ]; then
            echo "check-dmg: $executable uses $module ${bundled:-unknown}; the notice is from $pinned" >&2
            exit 1
        fi
    done
done
resource=Contents/Resources/licenses/Linked-Go-Modules.txt
expected=$(mktemp)
trap 'rm -f "$expected"; hdiutil detach "$mount" >/dev/null' EXIT
sh "$root/ferruccio/scripts/linked-module-notices.sh" "$app" > "$expected"
if [ ! -f "$app/$resource" ] || [ -L "$app/$resource" ] || ! cmp -s "$expected" "$app/$resource"; then
    echo "check-dmg: missing or changed notice: $resource" >&2; exit 1
fi
sum=$(cd "$app" && shasum -a 256 "$resource")
if ! grep -Fxq "$sum" "$app/Contents/Resources/SHA256SUMS"; then
    echo "check-dmg: SHA256SUMS does not list $resource" >&2; exit 1
fi
for want in Ferruccio.app howto-run-dis.md Applications "Ferruccio Quick Start.pdf"; do
    if [ ! -e "$mount/$want" ]; then
        echo "check-dmg: the image has no $want" >&2; exit 1
    fi
done
guide="Ferruccio Quick Start.pdf"
if ! cmp -s "$root/doc/applications/ferruccio/$guide" "$mount/$guide"; then
    echo "check-dmg: $guide differs from source" >&2; exit 1
fi
printf 'image holds Ferruccio.app, howto-run-dis.md, Applications and %s\n' "$guide"
