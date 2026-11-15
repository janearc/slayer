#!/bin/sh
# parallel.sh: moonlake, the whole film or a stretch of it, in N slices
# side by side, then one sequence encoded once with the song under it.
# improv.
#
#   parallel.sh N [FROM TO [OUT]]
#
# FROM and TO are seconds into the song, the whole song unless given;
# OUT is the directory, film/ beside this script unless given. The
# environment is render.sh's (BIN, PLATE, MASK, SCORE, SONG and the
# tools' own), plus RAWS.
#
# Each slice is render.sh over an exact run of frames, into OUT/part-K;
# then OUT/frames holds every frame in order, linked from the parts, and
# OUT/show.mp4 is encoded from it with the song from FROM. A slice that
# fails stops it: the others finish, nothing is assembled, and the
# failing slice is named.
set -eu
n=$1
here=$(cd "$(dirname "$0")" && pwd)
start=${2:-0}
end=${3:-235.2}
film=${4:-$here/film}
fps=24
total=$(awk -v a="$start" -v b="$end" -v f="$fps" 'BEGIN{print int((b-a)*f+0.5)}')
each=$(( (total + n - 1) / n ))
mkdir -p "$film"
pids=""
k=0
while [ "$k" -lt "$n" ]; do
	first=$((k * each))
	last=$(( (k + 1) * each ))
	[ "$last" -gt "$total" ] && last=$total
	from=$(awk -v s="$start" -v a="$first" -v f="$fps" 'BEGIN{printf "%.6f", s+a/f}')
	to=$(awk -v s="$start" -v a="$last" -v f="$fps" 'BEGIN{printf "%.6f", s+a/f}')
	echo "slice $k: frames $first to $last, $from s to $to s"
	GOMAXPROCS=2 nice "$here/render.sh" "$from" "$to" "$fps" "$here/cues.yaml" \
		"$film/part-$k" >"$film/part-$k.log" 2>&1 &
	pids="$pids $!"
	k=$((k + 1))
done
failed=""
k=0
for pid in $pids; do
	if ! wait "$pid"; then
		failed="$failed $k"
	fi
	k=$((k + 1))
done
if [ -n "$failed" ]; then
	echo "slices failed:$failed; see $film/part-K.log" >&2
	exit 1
fi
# every frame in order: each slice's frames are numbered from nought
mkdir -p "$film/frames"
k=0
while [ "$k" -lt "$n" ]; do
	first=$((k * each))
	for f in "$film/part-$k/frames/"*.png; do
		i=$(basename "$f" .png)
		ln -f "$f" "$film/frames/$(printf %05d $((first + 10#$i)))".png
	done
	k=$((k + 1))
done
ffmpeg -loglevel error -y -framerate "$fps" -i "$film/frames/%05d.png" \
	-ss "$start" -i "$SONG" -map 0:v -map 1:a -c:v libx264 -crf 16 \
	-preset slow -pix_fmt yuv420p -c:a aac -b:a 256k -shortest "$film/show.mp4"
echo "film: $(ls "$film/frames" | wc -l) frames, $film/show.mp4"
