# azazel/pages

Azazel keeps, for every job it runs, what the operator sees of a render while
it runs and when it ends. Before this, agents ran shell watchers by hand
beside each render (`latest-while.sh`, `frames-index.sh`,
`progress-while.sh`). This package does that work in Go, so that no
agent has to.

One publisher serves each queue. The standalone entry is a thin verb over
`pages.Serve`:

    azazel pages --serve --queue QUEUE --renders ROOT

The main worker starts it automatically when `--renders ROOT` or
`SLAYER_RENDERS` names the render root. The standalone command is available
for operator recovery; the queue lock prevents two publishers.
Agents run the watchers until the operator starts automatic publication
with the worker or the standalone publisher; they stop then.

On 2026-11-15: "all renders should be in the latest.html *as they are
produced* not just *when they are finished*", and, of a film page whose
link was missing until the end, "azazel should be" writing it.

## Which jobs

A job is published when its `--out` or `--output` argument lands under
`RENDERS/frames/X/` or `RENDERS/animations/X/`. `X` is the render folder.
Without one, the `outputs` of the job or set files its arguments name
are read instead, as a `draw JOB.yaml` chunk names its frames. They must
all land in one render folder. A job with none is logged once in
`pages.log` and nothing is published for it.

A relative path is read from the job's `directory`, or from the job
file's folder for its outputs. The renders root is `Settings.Renders`.

Every 10 seconds it reads the queue's ledger as `azazel status` does,
and never writes the store or touches dispatch. A job the ledger shows
running is live; one seen running that no longer is has ended. Every poll
also catches jobs that finished in the last ten minutes, including short
jobs completed between polls. A restart rebuilds publication from the disc
and ledger within that window. Older missed folders need operator `publish`.

One coordinator per queue: it holds `QUEUE/pages.lock`, and a second is
refused and told who holds it. Its log is `QUEUE/pages.log`, which it
alone writes. It does not write into the jobs' own logs.

## What it keeps

1. The render folder's page, `X/index.html`: films first, then the
   pictures, by name, with the count. Over 60 pictures it shows 48,
   evenly spaced. It is what `frames-index.sh` writes, byte for byte,
   with azazel's mark in place of the script's, and a line for each
   thing below that applies.
2. For a film (a folder under `animations/` holding numbered frames,
   `frame-000000.png` and on, in `X/frames/` or in `X` itself): the film
   so far, `X/partial.mp4`, from `frame-000000.png` up to the first gap,
   with its music (below), in hevc on VideoToolbox, tagged hvc1, BT.709,
   the height padded to a multiple of 16, so QuickTime plays it.
   `X/partial.json` says its frames and rate; the page says both.
3. `latest.html`, `frames/index.html` and `animations/index.html`, made
   as `latest.sh` and `indexes.sh` make them, from one walk of the
   renders tree.
4. Barkeep's line, `render-progress.txt`, for one render at a time.

## When

Every 10 seconds while any job is live: the folder pages, the films so
far when enough new frames have come, and barkeep's line. The gallery
pages at most once a minute. When a job ends: all of them once more, and
the last partial film if the frames have grown since the one before.

A failure to publish never changes the render job. It is recorded in
`pages.log`, naming the job and folder. Final publication retries at the next
poll, including after the render has completed. A clean shutdown makes a
bounded final ledger refresh before releasing the publisher lock.

## The music

On 2026-11-15:

    "Music is MANDATORY in every preview including partials; a silent
    movie is not a reviewable deliverable."

The partial film carries the film's music from its start, cut to the partial's
length, in aac, with a half-second fade at the cut.

The music is named by the render's own files: a score's `score.audio`
and a screenplay's `music`, with a digest in the receipt's
`music_sha256` or the score's `score.sha256`. All that are present must
agree, and the file must match its digest.

With no music named, two that disagree, a digest that does not match, or
a file that cannot be read, there is no partial film, and the page says
why. After muxing, ffprobe must find one audio stream within a frame of
the picture's length, or the film is not published. Never a silent film.

## The frame rate

The rate comes from the render's own files: a top-level `fps` in a
`.json`, `.yaml` or `.yml` file at the top of the render folder (a
receipt, a screenplay, a set), or in one of the job's declared inputs.
When none says it, or two disagree, there is no partial film and the
page says why. Azazel never guesses a rate.

The partial film is made again when a second of film, or a tenth more
frames, has come since the last one, whichever is more. Once a finished
film is in the folder (a visible `.mp4`, `.mov` or `.m4v` other than the
partial at its top),
the page links it and no more partial films are made.

## Barkeep's line

Barkeep reads `RENDERS/render-progress.txt`, one line of five whole
numbers:

    done total frames_per_minute eta_seconds epoch

It is written to a temporary file in that folder and renamed. `total` is
above nought; every number is nought or more, except `eta_seconds`,
which is -1 while it cannot be said. The pace and the time left are
taken over the last ten minutes, not since the start.

- `done` is the pictures in the render folder, counted as its page
  counts them: every `.png` at any depth, except `icon-*.png`.
- `total` is the request's `pictures`, the pictures the folder holds when
  the whole render is done. Without it the total is unknown: azazel
  writes no line for that render and says so once in `pages.log`. It
  never infers one.
- One render owns the line: the oldest live render with a known total.
  Azazel never writes over a line younger than 120 seconds that it did
  not write, such as a watcher still run by hand.
- At the end of a render that succeeded the line says `done` equal to
  `total` and nought seconds left.

## Pages made by hand

A folder page without azazel's or `frames-index.sh`'s mark was made by
hand. Azazel moves it aside once, to `index-before-azazel.html`, and its
own page links it. If that name is already taken, azazel leaves both
pages alone and says so in `pages.log`. It never overwrites a page
made by hand, and never links a file that is not there.

## Every write is whole

Pages, copies, partial films and the barkeep line are written beside
their name and renamed over it, so no reader sees part of one. The one
coordinator does all of the writing, so jobs in one folder never
overwrite each other's pages.

## For renders begun before

`azazel publish --renders ROOT --folder frames/X [--pictures N]`
publishes one render folder once, as when a job ends. It is the
operator's, for renders started before azazel published. Agents do not
run it for their jobs.
