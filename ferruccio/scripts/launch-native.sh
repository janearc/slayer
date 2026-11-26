#!/bin/sh
# Open against the shared renderer; never substitute this checkout's backend.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
evidence=${FERRUCCIO_EVIDENCE:-}
slayer=${FERRUCCIO_SLAYER:-}
if [ -n "${HOME:-}" ]; then
    if [ -z "$evidence" ]; then evidence="$HOME/mesh/the-vorrh/renders/frames"; fi
    if [ -z "$slayer" ]; then slayer="$HOME/mesh/the-vorrh/renders/applications/slayer/bin/slayer"; fi
fi
game=${GAME_BIN:-game5}
case "$game" in
    */*) game=$(cd "$(dirname "$game")" && pwd)/$(basename "$game") ;;
    *) game=$(command -v "$game") ;;
esac
exec "$root/bin/ferruccio" open --preview-transport azazel --azazel "${FERRUCCIO_AZAZEL:-}" --azazel-queue "${FERRUCCIO_AZAZEL_QUEUE:-}" --starter "${FERRUCCIO_STARTER:-}" --evidence "$evidence" --agent "${FERRUCCIO_AGENT:-ferruccio}" --game "$game" --slayer "$slayer" --examples "$root/ferruccio/examples" --game-home "${GAME_HOME:-}" "$@"
