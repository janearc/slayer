#!/bin/sh
set -eu
cd "$(dirname "$0")"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
scene=${SLAYER_SCENE:-scene.yaml}
export SLAYER_CACHE="$work/cache"
mkdir "$SLAYER_CACHE"

# Standard input is one model edit. Slayer reads the resulting scene.
edit=$(jq -esc 'if length == 1 then .[0] else error("expected one JSON edit") end')
printf '%s\n' "$edit" | jq -ef valid-edit.jq >/dev/null
yq -o=json "$scene" | jq --argjson edit "$edit" -f apply-edit.jq |
  yq -P > "$work/scene.yaml"

"${SLAYER_BIN:-slayer}" assess "$work/scene.yaml"
if [ -n "${SLAYER_PREVIEW:-}" ]; then
  "${SLAYER_BIN:-slayer}" draw "$work/scene.yaml" --camera front --renderer cpu --out "$SLAYER_PREVIEW"
  test -s "$SLAYER_PREVIEW"
fi
