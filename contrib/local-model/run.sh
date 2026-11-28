#!/bin/sh
set -eu
cd "$(dirname "$0")"
endpoint=${LOCAL_ENDPOINT:-http://127.0.0.1:8080}
port=${endpoint#http://127.0.0.1:}
case "$port" in
  ''|*[!0-9]*) printf '%s\n' 'Use a loopback OpenAI-compatible endpoint.' >&2; exit 2 ;;
esac
out=$(mktemp -d "$PWD/run.XXXXXX")
printf 'Run files: %s\n' "$out" >&2
curl -fsS --max-time 10 "$endpoint/v1/models" > "$out/models.json"
model=${LOCAL_MODEL:-$(jq -er '.data[0].id' "$out/models.json")}
scene=$(yq -o=json "${SLAYER_SCENE:-scene.yaml}" | jq -ec -f scene-context.jq)
example_scene=$(yq -o=json scene.yaml | jq -ec -f scene-context.jq)
ollama=false
if curl -fsS --max-time 2 "$endpoint/api/version" >/dev/null 2>&1; then ollama=true; fi
format=$(cat response-format.json)
training=train.json
action_prompt=''
if [ "${SLAYER_ACTIONS:-0}" = 1 ]; then
  format=$(cat action-format.json)
  training=action-train.json
  action_prompt=$(cat action-prompt.txt)
fi
examples='[]'
if [ "${WITH_EXAMPLES:-0}" = 1 ]; then examples=$(cat "$training"); fi
jq -c '.[]' "${SLAYER_CASES:-eval.json}" | while IFS= read -r task; do
  id=$(printf '%s' "$task" | jq -r .id)
  printf '%s' "$task" | jq --arg model "$model" --argjson examples "$examples" \
    --argjson scene "$scene" --argjson example_scene "$example_scene" \
    --argjson format "$format" --argjson ollama "$ollama" --arg action_prompt "$action_prompt" \
    -f request.jq > "$out/$id.request.json"
  elapsed=$(curl --fail-with-body -sS --max-time 300 -w '%{time_total}' \
    -H 'Content-Type: application/json' --data-binary "@$out/$id.request.json" \
    -o "$out/$id.response.json" "$endpoint/v1/chat/completions")
  jq -er '.choices[0].message.content | fromjson | select(type == "object")' \
    "$out/$id.response.json" > "$out/$id.output.json" 2> "$out/$id.assess.txt" || true
  : > "$out/$id.edit.json"
  if [ -s "$out/$id.output.json" ]; then
    if [ -n "$action_prompt" ]; then
      jq -e --argjson scene "$scene" -f resolve-action.jq "$out/$id.output.json" \
        > "$out/$id.edit.json" 2> "$out/$id.assess.txt" || true
    else
      cp "$out/$id.output.json" "$out/$id.edit.json"
    fi
  fi
  accepted=false
  if [ -s "$out/$id.edit.json" ] &&
     SLAYER_PREVIEW="$out/$id.png" sh assess.sh < "$out/$id.edit.json" > "$out/$id.assess.txt" 2>&1; then
    accepted=true
  fi
  printf '%s' "$task" | jq -c --slurpfile edit "$out/$id.edit.json" \
    --slurpfile response "$out/$id.response.json" --argjson accepted "$accepted" \
    --argjson seconds "$elapsed" '{
      id, accepted: $accepted, matches: (.expected == $edit[0]),
      pass: ($accepted and .expected == $edit[0]),
      seconds: $seconds, usage: $response[0].usage,
      timings: $response[0].timings
    }' | tee -a "$out/results.jsonl"
done
