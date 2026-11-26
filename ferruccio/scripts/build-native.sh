#!/bin/sh
# Build only the active frontend; Game owns its provenance stamp.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
game=${GAME_BIN:-game5}
case "$game" in
    */*) game=$(cd "$(dirname "$game")" && pwd)/$(basename "$game") ;;
    *) game=$(command -v "$game") ;;
esac
export GOWORK=off
sh "$root/ferruccio/scripts/patch-gogpu.sh"
mkdir -p "$root/ferruccio/native/bin" "$root/bin"
helper="$root/ferruccio/native/bin/.ferruccio-stage-next"
trap 'rm -f "$helper"' EXIT HUP INT TERM
(cd "$root/ferruccio/native" && CGO_ENABLED=1 "$game" build)
cp "$root/ferruccio/native/bin/ferruccio-stage" "$helper"
(cd "$root/ferruccio/native" && CGO_ENABLED=0 "$game" build)
mv "$helper" "$root/ferruccio/native/bin/ferruccio-stage"
mkdir -p "$root/bin"
cp "$root/ferruccio/native/bin/ferruccio" "$root/bin/.ferruccio-next"
cp "$root/ferruccio/native/bin/ferruccio-stage" "$root/bin/.ferruccio-stage-next"
mv "$root/bin/.ferruccio-next" "$root/bin/ferruccio"
mv "$root/bin/.ferruccio-stage-next" "$root/bin/ferruccio-stage"
"$root/bin/ferruccio" --age
"$root/bin/ferruccio-stage" --age
