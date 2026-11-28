#!/bin/bash
# Background coordinator only: submits the dependent print, never a worker.
set -euo pipefail
folder=$(cd -- "$(dirname -- "$0")" && pwd)
checkout=$(cd -- "$folder/../../../.." && pwd)
azazel=${AZAZEL_BINARY:?set AZAZEL_BINARY}
queue=${AZAZEL_QUEUE:?set AZAZEL_QUEUE}
frame_prefix=${CAPRICA_FRAME_PREFIX:-"$checkout/out/2026-11-27-system-half-"}
book=${CAPRICA_BOOK_DIRECTORY:-"${frame_prefix}portrait-book"}
source="$folder/source-manifest.json"
generator=${CAPRICA_LOOKBOOK_BINARY:-"$checkout/bin/caprica-lookbook"}
print_request=${CAPRICA_PRINT_REQUEST:?set CAPRICA_PRINT_REQUEST}
first_sequence=${CAPRICA_FIRST_SEQUENCE:?set CAPRICA_FIRST_SEQUENCE}
[[ "$first_sequence" =~ ^[1-9][0-9]*$ ]] || { echo "CAPRICA_FIRST_SEQUENCE must be positive" >&2; exit 2; }
state=${CAPRICA_COORDINATOR_STATE:-"$book/coordination/$first_sequence"}
names=(01-binary-transfer 02-amber-crown 03-azure 04-azure-flock 05-fresno-halo 06-fresno-close 07-caprica-quarter 08-caprica-night 09-caprica-deck 10-caprica-terminator 11-caprica-fresno 12-amber)
pins=(02afa87e9 02afa87e9 02afa87e9 0e7ed2a60 02afa87e9 701507dec 701507dec 701507dec 701507dec 701507dec 02afa87e9 02afa87e9)
mkdir -p "$state"
log() { printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*" | tee -a "$state/receipt.log"; }
fail() { log "FAILED: $*"; printf '%s\n' "$*" > "$state/FAILED"; exit 1; }
trap 'fail "coordinator command failed at line $LINENO"' ERR

[[ -x "$azazel" && -x "$generator" && -r "$source" && -r "$print_request" ]] || fail 'missing queue client, book generator, manifest, or print request'
log "Waiting for draw jobs $first_sequence–$((first_sequence + 11))."
print_sequence=""
[[ ! -s "$state/print-sequence" ]] || read -r print_sequence < "$state/print-sequence"
errors=0
while :; do
  if ! status=$("$azazel" status --queue "$queue"); then
    errors=$((errors + 1))
    log "Queue status unavailable ($errors consecutive attempts)."
    (( errors < 10 )) || fail 'queue status unavailable ten times'
    sleep 15
    continue
  fi
  errors=0
  complete=1
  for i in "${!names[@]}"; do
    seq=$((first_sequence + i))
    expected="system-half-${names[$i]}-${pins[$i]}"
    key=$(jq -r --argjson n "$seq" '.jobs[] | select(.sequence == $n) | .request.key' <<< "$status")
    [[ "$key" == "$expected" ]] || fail "job $seq missing or key mismatch: $key"
    phase=$(jq -r --argjson n "$seq" '.jobs[] | select(.sequence == $n) | .state' <<< "$status")
    case "$phase" in
      succeeded) ;;
      failed|cancelled|interrupted) fail "job $seq ($expected) ended $phase" ;;
      queued|running) complete=0 ;;
      *) fail "job $seq has unexpected state $phase" ;;
    esac
    if [[ "$i" == 10 && "$phase" == succeeded && -z "$print_sequence" ]]; then
      raw="${frame_prefix}11-caprica-fresno/light.pfm"
      [[ -s "$raw" ]] || fail 'slot 11 draw succeeded but its raw is missing or empty'
      ack=$("$azazel" submit --queue "$queue" "$print_request")
      print_sequence=$(jq -er '.sequence | select(type == "number" and . > 0)' <<< "$ack")
      printf '%s\n' "$print_sequence" > "$state/print-sequence"
      printf '%s\n' "$ack" > "$state/print-ack.json"
      log "Submitted slot 11 development as job $print_sequence."
    fi
  done
  if [[ -n "$print_sequence" ]]; then
    pstate=$(jq -r --argjson n "$print_sequence" '.jobs[] | select(.sequence == $n) | .state' <<< "$status")
    case "$pstate" in
      succeeded) ;;
      failed|cancelled|interrupted) fail "slot 11 development $print_sequence ended $pstate" ;;
      queued|running|"") complete=0 ;;
      *) fail "slot 11 development $print_sequence has unexpected state $pstate" ;;
    esac
  else
    complete=0
  fi
  (( complete == 0 )) || break
  sleep 15
done

for n in "${names[@]}"; do
  dir="${frame_prefix}$n"
  [[ -s "$dir/proof.png" && -s "$dir/light.pfm" ]] || fail "missing or empty final PNG/PFM for $n"
done

mkdir -p "$book"
jq --arg root "${frame_prefix}" \
  --argjson names '["01-binary-transfer","02-amber-crown","03-azure","04-azure-flock","05-fresno-halo","06-fresno-close","07-caprica-quarter","08-caprica-night","09-caprica-deck","10-caprica-terminator","11-caprica-fresno","12-amber"]' \
  '.Introduction += " This 1080×960 edition is provisional pending visual assessment of its new framing." | .Frames |= [to_entries[] | .value.Directory = ($root + $names[.key]) | .value.Image = "proof.png" | .value.Raw = "light.pfm" | .value]' \
  "$source" > "$book/manifest.json"
"$generator" make --manifest "$book/manifest.json" --out "$book/index.pending.html"
cp "${frame_prefix}03-azure/proof.png" "$book/cover-azure.png"
cp "${frame_prefix}12-amber/proof.png" "$book/cover-amber.png"
mv "$book/index.pending.html" "$book/index.html"
if [[ -n "${CAPRICA_GALLERY_NOTES:-}" ]]; then
  entry=${CAPRICA_GALLERY_ENTRY:?set CAPRICA_GALLERY_ENTRY to the book directory relative to the gallery}
  if ! grep -Fq "$entry"$'\t' "$CAPRICA_GALLERY_NOTES"; then
    printf '%s\t%s\n' "$entry" 'CAPRICA SYSTEM — 1080×960 edition, 9 samples per pixel; new framing awaits visual assessment; click cover.' >> "$CAPRICA_GALLERY_NOTES"
  fi
fi
if [[ -n "${CAPRICA_GALLERY_REFRESH:-}" ]]; then
  bash "$CAPRICA_GALLERY_REFRESH"
fi
log "Wrote $book/index.html; visual assessment remains pending."
printf '%s\n' "$book/index.html" > "$state/SUCCEEDED"
