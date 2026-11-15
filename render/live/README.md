# live

The view behind `slayer watch JOB`: a job's current frame drawn small at
once and then finer, in a terminal pane, drawn again when the job file
changes or a key asks.

The package is pure: it is handed a screen, a stager, a clock and a
reader of the job, each an interface or a function, and `door` hands it
the real ones (terminal.Live, a hillside session, time.Now, the door's
own intake). Its tests hand it mocks made by mockgen.

## The parts

- `Change` is how much a change costs, in order: nothing; redevelop (the
  kept light developed again); redraw (the frame drawn again by the open
  session); restage (the session closed and the job opened again).
- `Compare(old, new)` is the change between two jobs, read from their
  canonical forms, key by key, by a table.
- A `ladder` is the rungs for a pane: the picture fitted to the pane,
  then drawn at an eighth, a quarter, a half and the whole of that size
  at one ray, then the whole at two rays and at four. A rung whose
  predicted time passes the ceiling is not started; the prediction is
  the last rung's time scaled by pixels and by rays squared.
- `fitted` is the picture's size in the pane, keeping the job's shape:
  the pane's pixels are not square (`Screen.Pixel`), so the picture is
  drawn with square pixels and then resampled onto the pane's, so a
  circle stays round. A portrait job in a wide pane is a tall picture
  with bars either side.
- `view` is the state keys change: the frame shown, the camera chosen,
  exposure in thirds of a stop, playing or not, and the mode (preview,
  help, full size). Each key is a row of a table: the key, and what it
  does to the view and what that costs.
- `watcher` is the loop. One goroutine owns the view; one worker owns
  the stage and draws one plan at a time. A key or a file change while a
  plan runs cancels its context, waits for it to stop, folds everything
  that came meanwhile into one change, and starts the ladder again.
- `foot` is the last few notes and errors, newest at the bottom, above
  the status row. A file that does not read is a line of its own that
  stays until the file reads again, and the last good picture stays up.
- The status folder is described below.

## The status folder: the seam a graphical front end attaches to

Beside the job, in `.NAME.watch/` (NAME the job file's name without its
extension), `watch` keeps three files. Each is written beside itself and
renamed into place, so a reader sees the whole of one or of the one
before, never a part. The pictures are written before the status that
counts them.

- `frame.png`: the latest picture as drawn and developed, its pixels
  square, at the rung's size (or the job's full size after `f`).
  Rewritten only when the picture changes.
- `screen.png`: the cells as the pane shows them, each pixel grown to
  the pane's pixel shape. Rewritten with `frame.png`.
- `status.json`, rewritten on every change of what is shown, and once
  more with `state: stopped` when watch quits:

| Key | Says |
|---|---|
| `schema` | `slayer-watch-status-v1`; a reader refuses another |
| `job` | the job file, absolute |
| `state` | `staging`, `drawing`, `shown` or `stopped` |
| `mode` | `preview`, `help` or `full` |
| `quality` | `preview` or `exact`, as drawn |
| `time`, `frame`, `frames`, `rate` | the shown frame's scene time in seconds, its index from nought, the job's frame count and frames a second (one for a still) |
| `playing` | whether a film is playing |
| `camera` | the camera shown by name; empty for the scene's own |
| `exposure_thirds` | the exposure the keys chose, thirds of a stop |
| `rung` | the rung shown, in words |
| `pictures` | how many `frame.png` and `screen.png` pairs were written; it moves when they change |
| `width`, `height`, `rays` | `frame.png`'s picture as drawn |
| `error` | the trouble with the job or its scene now, or empty; the last good picture stays while it stands |
| `notes` | the foot's lines, oldest first |
| `when` | when this was written, RFC 3339 with nanoseconds |

A front end polls `status.json`, reloads `frame.png` when `pictures`
moves, and edits the job file to change what is drawn. Nothing reads
keys from the folder: `watch` takes its keys from its terminal only.

## What is followed

The job file is looked at every `PollEvery`, and so are the set it names
and its titles' theme: a save of any of them reads the job again and
draws what changed (a set restages, a theme develops again). What a set
reads in its turn, a shelf's sources or terrain, is not followed; a
change there shows at the next save of the job or the set.

## What reopens the session

The session (`hillside.Open`) stages the scene with the job's camera,
times, exposure, renderer, quality and shutter, and draws a frame at any
size and rays it is asked. So:

| What changed | Cost |
|---|---|
| `film` width, height or rays (and a preset, which gives them) | redraw |
| `print`, `stock`, `titles`, a title theme's file | redevelop |
| `sound` and its files, `outputs`, `limits` | nothing: the preview plays no sound |
| the set's file, or a frames folder, changed on disk | restage |
| anything else: `scene`, `camera`, `at`, `fps`, `duration`, `start`, `scene-duration`, `shutter`, `renderer`, `quality`, `film` exposure, wide or tall | restage |

The spec hoped that a change of time or camera would be a redraw. The
session as built fixes both at Open, so they restage; `gpu.Frames` keeps
the device's scene between frames of one session, which is what makes
stepping along a film (left and right) cheap. Keys that change the camera
(`c`) or a still's moment (`[` and `]` on a still) restage too, and say
"staging..." while they do.

## Keys

Byte 0x1d (control-right-bracket) is never bound; it arrives as
`KeyHost` and is dropped.

| Key | Does | Costs |
|---|---|---|
| left, right | a film's frame back or on by one | redraw |
| `[`, `]` | a film's frame back or on by a second; a still's moment by a second | redraw; restage on a still |
| space | play or pause a film at its rate, dropping frames it cannot draw in time | redraw |
| `c` | the scene's next camera by name | restage |
| `e`, `E` | exposure down or up a third of a stop: the kept light scaled and developed again | redevelop |
| `f` | the current frame at the job's full size; any key returns | redraw |
| `s` | the moment, camera and exposure written back into the job file | nothing |
| `?` | help; any key returns | nothing |
| `q`, control-c | quit | |

`s` edits the YAML in place, comments kept: a still's `at`, a camera
chosen by `c` as `camera: {name: ...}`, and `film.exposure` as the
scene's exposure scaled by the stops chosen. A film's frame is not a
key of the job and is not written. The status row says the editor must
reload.

## What it is not

A preview is drawn at preview quality unless the job or `--quality
exact` says otherwise, and the status row says which. It is never
evidence of what the final render looks like to the bit. The tracers
report a frame once it is done, so `f` shows the full frame when it is
finished, not row by row; a frame already being traced runs to its end
before a cancel is seen.

## Tests

Every test is quick: no GPU, no shelf and no staged scene. They run on
mocks and on temporary folders, under `go test -race ./render/live/`,
about 4 s with `GOMAXPROCS=3` (the slowest waits on a real poll or on
a cancelled frame, under 0.3 s each; a few that expect nothing to
happen wait 100 to 200 ms). Nothing here draws, so there is nothing to
list under "Tests that draw".

Every positive test is answered by at least one anti-test, marked
`// anti-test of TestX` (`go run ./tools/antitests ./render/live/`).

The failure cases that matter are covered by anti-tests: a key that
changes nothing does not cancel a frame; a frame cancelled by a key is
not a trouble at the foot or in the status; a job that does not read at
the start is not staged and is in the status as an error until it does;
a refused job and a panicking scene are told and the loop lands; a
status that cannot be put in place leaves no stray file.

### Tests with no anti-test, and why

None.
