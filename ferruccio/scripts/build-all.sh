#!/bin/sh
# Let root Game sync and build first, then stamp the nested frontend.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
game=${GAME_BIN:-game5}
case "$game" in
    */*) game=$(cd "$(dirname "$game")" && pwd)/$(basename "$game") ;;
    *) game=$(command -v "$game") ;;
esac
export GAME_BIN="$game" GOWORK=off
(cd "$root" && CGO_ENABLED=1 "$game" build)
/bin/sh "$root/ferruccio/scripts/build-native.sh"
