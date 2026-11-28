# live

The view behind `slayer watch JOB`: a job's current frame drawn small at
once and then finer, in a terminal pane, drawn again when the job file
changes or a key asks.

The terminal retains the last good picture when a scene fails to read.
It cancels an obsolete draw when a new edit or key arrives and draws the
latest view when the worker is ready. Job dimensions and the terminal's
pixel shape determine how the picture fits.

For a preview, use `slayer watch JOB` in a terminal. Choose
`--dry-run` to inspect the resolved job and status directory without
rendering. Use the [render queue](../../doc/azazel.md) for offline jobs.

## Status files

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

The session (`session.Open`) stages the scene with the job's camera,
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

## Rendering limits

The status row identifies preview or exact quality. The full-size action
shows a frame when tracing completes. Cancellation of a trace can wait
for the current frame.

The tests use temporary folders and mock stages. Run them
with `game test -- -race ./render/live/...`.
