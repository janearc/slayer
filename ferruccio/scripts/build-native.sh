#!/bin/sh
# Build only the active frontend; Game owns its provenance stamp.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
game=${GAME_BIN:-game5}
case "$game" in
    */*) game=$(cd "$(dirname "$game")" && pwd)/$(basename "$game") ;;
    *) game=$(command -v "$game") ;;
esac
export CGO_ENABLED=0 GOWORK=off
sh "$root/ferruccio/scripts/patch-gogpu.sh"
(cd "$root/ferruccio/native" && "$game" build)
mkdir -p "$root/bin"
cp "$root/ferruccio/native/bin/ferruccio" "$root/bin/.ferruccio-next"
mv "$root/bin/.ferruccio-next" "$root/bin/ferruccio"
"$root/bin/ferruccio" --age
