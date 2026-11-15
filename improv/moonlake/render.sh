#!/bin/sh
# render.sh: moonlake's show for a stretch of the song, frame by frame,
# with the song under it. improv.
#
#   render.sh FROM TO FPS CUES OUT
#
# FROM and TO are seconds into the song; CUES is a cue sheet; OUT is a
# directory, which gets frames/ and show.mp4. The tools, the plates and
# the song come from the environment:
#
#   BIN    the built tools: cueshow lightshow seawater seamatte mistlake
#          develop, and slate when SLATE is set
#   PLATE  the scene's raw (g5.pfm); MASK its silhouette raw (mask.pfm)
#   SCORE  score.yaml; SONG the song (moon-lake.m4a)
#
# With MOON_DISC (the disc alone, at MOON_DISC_ALT), PLATE is the scene
# without its disc and lightshow lays the disc in where the cues stand
# it; the tools take the moon's light from the disc pass, not the plate.
# FIRE_FRAMES passes through to lightshow.
#
# MOONLAKE_FRAME_HEIGHT is the frame's height in mm, 39.61 for the 16:9
# crop of the 15/70 aperture unless told, 51.61 for the whole of it; the
# plates must be rendered to the same. BLOOM is develop's, in pixels,
# 0.8 unless told: twice that at twice the size keeps the same look.
#
# For a long render: LOAD_GATE holds each frame while the load is at or
# over it, or while swap used is over SWAP_GATE GB (8 unless told);
# RESUME keeps the frames already in OUT/frames, so a stopped render
# starts where it stopped (a frame is moved there only when whole); and
# ENCODE=no leaves show.mp4 for whoever puts the frames together.
set -eu
from=$1 to=$2 fps=$3 cues=$4 out=$5
: "${BIN:?}" "${PLATE:?}" "${MASK:?}" "${SCORE:?}" "${SONG:?}"
mkdir -p "$out/frames" "$out/work"
work=$out/work
export MOONREF="${MOON_DISC:-$PLATE}"
# the frame's edges and the far shore, in degrees, from the camera the
# tools are told (MOONLAKE_EYE, MOONLAKE_LOOK, MOONLAKE_FOCAL), the first
# view's when unset; WATERMASK and PASSES pass through to the tools
read -r top bottom shore <<EOF
$(awk -v e="${MOONLAKE_EYE:-53.6}" -v l="${MOONLAKE_LOOK:--0.5}" \
	-v f="${MOONLAKE_FOCAL:-150}" -v h="${MOONLAKE_FRAME_HEIGHT:-39.61}" 'BEGIN{
	half = atan2(h/2, f) * 180 / 3.141592653589793
	printf "%.3f %.3f %.3f", l + half, l - half,
		atan2(-e, 1660) * 180 / 3.141592653589793}')
EOF
frames=$(awk -v a="$from" -v b="$to" -v f="$fps" 'BEGIN{print int((b-a)*f+0.5)}')
# the water is drawn at the plate's own size: its pfm's second line
read -r width height <<EOF
$(awk 'NR == 2 {print $1, $2; exit}' "$PLATE")
EOF
i=0
while [ "$i" -lt "$frames" ]; do
	t=$(awk -v a="$from" -v i="$i" -v f="$fps" 'BEGIN{printf "%.4f", a+i/f}')
	name=$(printf %05d "$i")
	if [ -n "${RESUME:-}" ] && [ -s "$out/frames/$name.png" ]; then
		i=$((i + 1))
		continue
	fi
	if [ -n "${LOAD_GATE:-}" ]; then
		while [ "$(sysctl -n vm.loadavg | awk '{print int($2)}')" -ge "$LOAD_GATE" ] ||
			[ "$(sysctl -n vm.swapusage | awk '{print int($6 / 1024)}')" -gt "${SWAP_GATE:-8}" ]; do
			sleep 30
		done
	fi
	# Under 30 GB free, everything stops. Each second of
	# frames, the disk the frames go to is checked, and under DISK_FLOOR
	# GB (35 unless told) the render stops, saying where it stopped
	if [ $((i % fps)) -eq 0 ]; then
		free=$(df -g "$out" | awk 'NR == 2 {print $4}')
		if [ "$free" -lt "${DISK_FLOOR:-35}" ]; then
			echo "disk: ${free} GB free, under ${DISK_FLOOR:-35}; stopped at frame $name ($t s)" >&2
			exit 3
		fi
	fi
	"$BIN/cueshow" "$SCORE" "$cues" "$t" "$fps" >"$work/cue.env"
	mist=$(sed -n 's/^mist=//p' "$work/cue.env")
	master=$(sed -n 's/^master=//p' "$work/cue.env")
	overhead=$(sed -n 's/^overhead=//p' "$work/cue.env")
	exposure=$(sed -n 's/^exposure=//p' "$work/cue.env")
	# the camera's exposure: the moon's 23 stops, moved by the cue's.
	# lightshow rolls the fire off against it (STOPS)
	stops=$(awk -v e="$exposure" 'BEGIN{printf "%.3f", 23+e}')
	level=$(CUES="$work/cue.env" STOPS="$stops" "$BIN/lightshow" "$PLATE" \
		"$MASK" "$work/k.pfm" "$work/n.pfm" "$t")
	"$BIN/seawater" "$work/s.pfm" "$width" "$height" "$t" 8 "$work/k.pfm"
	FRONT="$work/n.pfm" "$BIN/seamatte" "$work/k.pfm" "$work/s.pfm" \
		"$work/o.pfm" "$top" "$bottom" "$shore" 1.554
	# the mist: its cue under the master, the strobes' flashes, and the
	# overhead's daylight on it (OVERHEAD_MIST times the moon's, at full)
	strength=$(awk -v m="$mist" -v l="$level" -v a="$master" -v o="$overhead" \
		-v g="${OVERHEAD_MIST:-2500}" 'BEGIN{printf "%.3f", m*a+0.25*l+o*g}')
	"$BIN/mistlake" "$work/o.pfm" "$work/m.pfm" "$top" "$bottom" "$shore" \
		"$strength" 3 "$t"
	"$BIN/develop" -exposure "$stops" -bloom "${BLOOM:-0.8}" \
		-out "$work/frame.png" "$work/m.pfm" >/dev/null
	# SLATE writes the frame's timing over it: its number in the film,
	# the song's time, the bar, and the fire's strikes
	if [ -n "${SLATE:-}" ]; then
		"$BIN/slate" "$work/cue.env" "$t" "$fps" "$work/frame.png"
	fi
	mv "$work/frame.png" "$out/frames/$name.png"
	# RAWS keeps each frame's light before develop, as half-float OpenEXR
	# with lossless zip, so the film can be graded again without a render.
	# The light is first scaled up by the moon's 23 stops, since the raw
	# sits below half-float's precision; the frame's own stops go beside
	# it, so its develop is -exposure (stops - 23) on the kept light
	if [ -n "${RAWS:-}" ]; then
		mkdir -p "$out/raw"
		"$BIN/scalepfm" "$work/m.pfm" "$work/m23.pfm" 23
		ffmpeg -loglevel error -y -i "$work/m23.pfm" -c:v exr -format half \
			-compression zip16 "$out/raw/$name.exr"
		printf '%s %s\n' "$name" "$stops" >>"$out/raw/exposures.txt"
	fi
	i=$((i + 1))
done
rm -r "${work:?}"
if [ "${ENCODE:-yes}" = no ]; then
	echo "$out/frames: $frames frames"
	exit 0
fi
ffmpeg -loglevel error -y -framerate "$fps" -i "$out/frames/%05d.png" \
	-ss "$from" -t "$(awk -v a="$from" -v b="$to" 'BEGIN{print b-a}')" \
	-i "$SONG" -map 0:v -map 1:a -c:v libx264 -crf 18 -pix_fmt yuv420p \
	-c:a aac -b:a 192k -shortest "$out/show.mp4"
echo "$out/show.mp4: $frames frames"
