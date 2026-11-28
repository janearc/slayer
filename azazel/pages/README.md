# Render gallery publisher

Start the worker with `--renders ROOT` or set `SLAYER_RENDERS` on the worker
to publish the gallery while jobs run. For operator recovery, run:

```sh
azazel pages --serve --queue QUEUE --renders ROOT
```

One publisher holds `QUEUE/pages.lock`. Its diagnostics are in
`QUEUE/pages.log`. Stop any external gallery or progress watchers when
automatic publication takes over.

## Which jobs appear

The publisher takes a job's `--out` or `--output` path first. Otherwise it
reads output paths from the job or set files named in the arguments. To
appear in the gallery, the paths must identify one folder under
`ROOT/frames/` or `ROOT/animations/`. A relative command-line path is
resolved from the job's working directory. Paths inside files are resolved
from those files' folders.

The publisher reads the ledger every ten seconds and can catch jobs that
completed between polls. A restart catches completed jobs from the last
ten minutes. To publish an older missed folder, use the operator command
`azazel publish --renders ROOT --folder frames/X`.

The publisher writes the render folder's `index.html`, `latest.html` and the
still and animation indexes. It refreshes the gallery indexes at most once
per minute while work is running and again when work ends. Failed final
publication retries after the next ledger poll without changing job status
or rerendering.

A hand-written folder page moves to `index-before-azazel.html` and stays
linked. If that name already exists, the publisher leaves both files
alone and logs the conflict. Files are installed by rename.

## Partial films

For an animation folder, `partial.mp4` uses the numbered PNGs from
`frame-000000.png` to the first gap. `partial.json` records the count
and frame rate. A finished movie at the top of the folder takes over
from the partial.

Provide an agreed positive `fps` in the render folder's JSON or YAML records
or the job's declared inputs. Identify the score with `score.audio` or
`music`. If a record supplies `score.sha256` or `music_sha256`, it must
match the audio file. The publisher checks the encoded audio stream.
Missing or conflicting required metadata leaves a reason on the folder page.

VideoToolbox encodes HEVC with AAC music cut to the partial's length.
While work runs, the partial refreshes when at least a second of film or
10% more frames are available, whichever is greater. When the folder's
last job ends, any new contiguous frames can trigger a final partial.

## Barkeep progress

`ROOT/render-progress.txt` contains one line for the oldest live render
with a known total:

```text
done total frames_per_minute eta_seconds epoch
```

`done` counts PNGs under the render folder except `icon-*.png`.
`total` comes from the request's `pictures` field. Azazel does not infer
it. Pace uses the last ten minutes, and `eta_seconds` is `-1` when
unavailable. A successful final update has zero seconds left only if
the observed count has reached the total. The publisher respects
another writer's line while that line is younger than 120 seconds.
