#!/usr/bin/env bash
# Prepare local build inputs; leave system tools and existing Game config alone.
set -euo pipefail

usage() {
    cat <<'HELP'
Usage: ./bootstrap.sh [--check]

--check inspects prerequisites without downloads, builds or file writes.
Without it, prepare a private Game home, download Go dependencies, fetch
and verify the source shelf, then build Slayer through Game. If needed,
clone and bootstrap Game locally (including Game's own checks).

Install Git, Go and a C compiler first. On macOS, use the Xcode command-line
tools. ffmpeg is optional for pictures, required for movie/audio workflows.

Environment:
  GAME                    existing compatible Game executable
  SLAYER_BOOTSTRAP_DIR     absolute local setup directory outside this checkout
  SLAYER_SHELF             absolute source shelf outside this checkout

No system packages are installed and no personal Game config is rewritten.
Read doc/building.md for setup limitations.
HELP
}

mode=setup
case "${1:-}" in
    --help|-h) usage; exit 0 ;;
    --check) mode=check ;;
    '') ;;
    *) usage >&2; exit 2 ;;
esac
if (( $# > 1 )); then usage >&2; exit 2; fi

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
cd "$root"
state=${SLAYER_BOOTSTRAP_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/slayer-bootstrap}
shelf=${SLAYER_SHELF:-$state/sources}
# Resolve existing parents without creating them, including macOS /tmp aliases.
physical_location() {
    local parent=$1 tail=
    while [[ ! -d "$parent" ]]; do
        tail="/$(basename -- "$parent")$tail"
        parent=$(dirname -- "$parent")
    done
    printf '%s%s\n' "$(cd -- "$parent" && pwd -P)" "$tail"
}
for location in "$state" "$shelf"; do
    case "$location" in
        /*) ;;
        *) printf 'Use an absolute bootstrap/shelf path: %s\n' "$location" >&2; exit 2 ;;
    esac
    if [[ "$location" == *$'\n'* || "$location" == *$'\r'* ]]; then
        printf 'Setup paths cannot contain line breaks.\n' >&2; exit 2
    fi
    case "$location/" in
        */../*|*/./*) printf 'Use a normalized setup path: %s\n' "$location" >&2; exit 2 ;;
    esac
done
state=$(physical_location "$state")
shelf=$(physical_location "$shelf")
for location in "$state" "$shelf"; do
    case "$location/" in
        /|//|"$root/"*) printf 'Keep setup inputs outside this checkout: %s\n' "$location" >&2; exit 2 ;;
    esac
done

failed=0
for tool in git go cc; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        printf 'Missing required tool: %s (see doc/building.md).\n' "$tool" >&2
        failed=1
    fi
done
if ! command -v ffmpeg >/dev/null 2>&1; then
    printf 'Optional tool missing: ffmpeg; install it before movie/audio work.\n'
fi

# Inspect the current single-line local replacements before doing network work.
# Never rewrite go.mod behind the user.
local_replacements=$(awk '$1 == "replace" && $(NF-1) == "=>" {p=$NF; if (substr(p,1,1)=="/" || substr(p,1,1)==".") print p}' go.mod)
while IFS= read -r replacement; do
    [[ -n "$replacement" ]] || continue
    if [[ ! -d "$replacement" ]]; then
        printf 'Missing local module replacement: %s\n' "$replacement" >&2
        printf 'Create that folder or remove the replace line from go.mod.\n' >&2
        failed=1
    fi
done <<< "$local_replacements"

game_bin=${GAME:-}
if [[ -n "$game_bin" ]]; then
    if ! game_bin=$(command -v "$game_bin"); then
        printf 'GAME does not name an executable.\n' >&2; exit 1
    fi
else
    for candidate in game game5 "$state/game/bin/game"; do
        if game_bin=$(command -v "$candidate" 2>/dev/null); then break; fi
        game_bin=
    done
fi

required_go=$(awk '$1 == "go" {print $2; exit}' go.mod)
printf 'Checkout requests Go %s and Game %s.\n' "$required_go" "$(awk '$1 == "game" {print $2; exit}' .game)"
if command -v go >/dev/null 2>&1; then GOTOOLCHAIN=local go version; fi
printf 'Setup directory: %s\nSource shelf: %s\n' "$state" "$shelf"
if [[ -n "$game_bin" ]]; then
    "$game_bin" --age
else
    printf 'Game is not installed locally; setup will clone its source and run its bootstrap.\n'
    [[ "$mode" != check ]] || failed=1
fi
if (( failed )); then exit 1; fi
if [[ "$mode" == check ]]; then
    printf 'Prerequisite inspection passed. Module downloads, source hashes and Metal device support still need their actual checks.\n'
    exit 0
fi

export GOMAXPROCS=2
export GOFLAGS=-p=1
export GOWORK=off
export SLAYER_SHELF="$shelf"
# This is a script-owned isolated home, never the caller's personal config.
export GAME_HOME="$state/game-home"
mkdir -p "$GAME_HOME/.config/game" "$shelf"
config_file="$GAME_HOME/.config/game/config"
marker='# managed by Slayer bootstrap'
if [[ -e "$config_file" ]] && ! [[ "$(head -n 1 "$config_file")" == "$marker" ]]; then
    printf 'Refusing to replace an unmanaged config: %s\n' "$config_file" >&2; exit 1
fi
printf '%s\ngomaxprocs 2\nproject slayer\n    set SLAYER_SHELF %s\n' "$marker" "$shelf" > "$config_file"

if [[ -z "$game_bin" ]]; then
    if [[ -e "$state/game" ]]; then
        printf 'Incomplete Game checkout exists at %s; inspect it before retrying.\n' "$state/game" >&2; exit 1
    fi
    git clone https://github.com/janearc/game.git "$state/game"
    (cd "$state/game" && sh bootstrap.sh)
    game_bin="$state/game/bin/game"
fi

"$game_bin" lane tests -- go mod download
"$game_bin" lane tests -- "$game_bin" run sources-fetch
"$game_bin" lane tests -- "$game_bin" run sources-verify
"$game_bin" lane tests -- "$game_bin" build
printf '\nBuilt Slayer. For later Game commands in this shell:\n'
printf 'export GAME_HOME=%q\n' "$GAME_HOME"
printf 'export SLAYER_SHELF=%q\n' "$shelf"
printf 'export PATH=%q:"$PATH"\n' "$(dirname "$game_bin")"
printf 'Runner: %s (use this executable for the guide commands).\n' "$game_bin"
printf 'Then follow doc/getting-started.md. No Slayer render suite was run.\n'
