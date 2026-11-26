# door

Slayer's front door: the `slayer` tool, built on Puzzlemouth (`pzm`), and
its verbs. `cmd/slayer` is a thin `main` that builds the tool and runs it.
The verbs that draw take a job document (package `job`); list and
tunes take none. This package joins the pieces that already exist and
holds no picture-making of its own.

## The tool

- `Tool(env)` is the `*pzm.Tool`. An `Env` is made by `NewEnv(Given)`:
  the context the verbs run under, the build stamp, the environment to
  read `SLAYER_RENDERS` and `SLAYER_TEAM` from, the clock, and the working
  directory `out/` is under. The output and error writers are the ones
  handed to `Tool(env).Run`, as pzm takes them; they are not kept twice.
- `--age` and `--who` are pzm's, from the stamp.
- The flags every job-taking verb shares are declared once, by
  `jobFlags`, and read once, by `overridesOf`, into `job.Override`s:
  `--out`, `--width`, `--height`, `--rays`, `--at`, `--camera`,
  `--renderer`, `--quality`, and a flag for each of the job's flags
  and shortcuts, `on` or `off`, made from the schema (`job.Switches`):
  `--vibes`, `--fastlight`, `--lamp-store`, `--sky-looks`.

  Unset is
  the job's. A bad
  number or time is refused in one wording:
  `--width takes a whole number, not "x"`.
- Verbs return errors. Nothing here exits the process.
- Each verb other than draw is in a file of its own, `NAME.go`, which
  declares it with `declareNAME`; `tool.go` calls each once.

## list

`slayer list` says the headings, one a line with what each is.
`slayer list HEADING` lists one: each entry's name, a line of what it
is, and below it what it takes. `--json` writes the same for a program.

Every heading is read from the registry that owns it, never a list kept
here: `scene/named`'s scenes and their parameters, `scene/smoke`'s
smokes, `job`'s presets, `ninjafilm`'s stocks, `cheeptune`'s built-in
scores and voicings and its library, and `manufacture`'s kinds (the
catalogue's). The tests hold each heading to its registry, entry for
entry.

## tunes

`slayer tunes DIR` serves a tune library folder's audition page and
records keep and toss votes in its `votes.json`, through
`cheeptune/audition`'s server. It listens on a loopback address only
(`--address`, `127.0.0.1:8417` unsaid), makes the server with the port
it bound, says the address to open, and stops on control-c.

One switch in place of serving, over every tune or the names given:
`--check` (length, peak, voicing; fails if any will not load),
`--films` (`NAME.mp4`), `--plates` (`NAME.png`) and `--page`
(`index.html`), written beside the tunes; `--plates` in `--out`'s
folder when it is given. `--sounds VOICING...` draws the studio
voicings' instrument sheet to `--out`, or `out/sounds.png`.

These were `cmd/tunefilms`'s five verbs. One changed: `plate DIR NAME
OUT` wrote one still to a path, and `--plates` writes every still, or
the names given, into a folder.

## assess

`slayer assess FILE` is whether FILE will draw, without drawing: the
same reading, flags, places and refusals as draw, then each piece draw
draws with opened by draw's own runner (`tools`, then for a film
`prepare`: the frames folder's ledger and the sound) and let go. It
says what draw would write, or `NAME will not draw: why`.

`--census PATH` writes the job's census to PATH as JSON (`census.go`):
the scene opened by the same runner, then the session's `Session.Census`
of the first frame (a still's at `--at`), counted by `thing.Count`, with
nothing drawn.

Each placed thing says its faces; a mesh group its instanced faces, its
stored faces (each distinct mesh once) and its instances; analytic shapes
(spheres, planes, height fields with their grid) and volumes are listed by
kind and never given faces.

The receipt is the job file's sha256 and key, its scene and camera as the
job says them, the set file's sha256, the frame's time, the camera as staged
and the build and commit. `counts` says `staged, not visible`: these are the
faces in the scene, not the faces seen or hit from the camera. Frames already
rendered have nothing to count and are refused.

`--places PATH`, here and on draw, writes the table of where each thing
of the set was stood: `of,kind,east,north,height`, a line a thing, in
metres, as the old command's `-places` wrote it (`Session.Places`, from
the staging's own table). It is not the census. A scene that stands
nothing on a ground, such as a set of a world and no ground, is refused.
Draw draws the picture afresh when it is asked for, not from the cache.

`--judge` draws a set as the terminal shows it and as a real picture
and judges the one against the other (what `slayer set` did), writing
both, the look and the verdict to `--out`, `out/` unsaid. `--smokes`
judges the smokes through `render/smokejudge` (what `slayer smokes` and
`smokecheck` did). A judgement that is not honest fails.

## develop

`slayer develop JOB RAW...` develops light already drawn through the
job's print or stock (`finish.Developer`, as draw) with its titles
(`finish.Titles`) at each frame's film time. RAW is .pfm files or one
folder of plates (`finish.OpenPlates`).

Each raw is its own picture: a change from `cmd/develop`, which added
several raws into one. A raw named `frame-000042.pfm` is frame 42, its
grain and titles frame 42's; others count up from `--frame N`, or from
0, and the line says so.

`--sum` adds the raws into one long exposure, as `cmd/develop` did.
`--white K` balances the light first; `--white 0` no longer means none,
so leave the flag out. `cmd/develop`'s `-exposure`, `-knee`, `-top` and
`-bloom` are the job's `print` keys `stops`, `knee`, `top` and `bloom`.

## prove

`slayer prove JOB RAW` holds a raw drawn on the cpu to the exact tracer.
JOB is read as draw reads it, with the same flags laid over it: give the
flags that drew RAW. `Session.Prove` makes chosen pixels again through
the world, eye, lens, film time, size and rays the frame is drawn with,
and `fermat` holds each to RAW bit for bit.

`--pixel` names the pixels to prove first, X,Y each; a flag is taken
once, so several go in one, between semicolons: `--pixel "3,4;10,2"`.
`--many N` is how many more RAW's own bytes pick, `fermat.Many` unsaid.
`--frame N` is the frame of a film that RAW is, 0 unsaid. `--proof DIR`
is where the sheet goes (`Session.Sheet`), beside RAW and named for it
unsaid.

It says the proof and where the sheet is, and exits 0 where the proof
holds and 1 where it fails or cannot be made. A gpu session, an explicit
stack of media, a frame that is a shutter's mean, a film through a rig
and a raw of another size are refused by name.

## draw

`slayer draw FILE` reads FILE as one of three things, by its `$schema`:
a job (job-v1), a sequence plan (sequence-v1, lifted with
`session.LiftSequence`), or anything else as a set file, drawn as a
still with defaults (`job.StillOf`). The overrides are laid over it.

Then `placesOf` works out where each output goes. `--dry-run` prints the
job's canonical form as YAML and each file it would write, and stops.

Otherwise the runner draws. For each frame: the light, from a
session for a set or a smoke, or from `finish.Plates` fitted to the film
for `scene: {frames}`; developed (`finish.Developer`); the set's own hud
(`Overlay`); the titles at the frame's film time; then its PNG and the
movie.

Where the job says no print and no stock, the session's tone develops
the light, and for plates the clamp. A still is one frame written to
its picture, and its light to `raw` when asked.

The sound is made before the first frame: a tune through `cheeptune`,
looped to the film's end when its score loops and cut there, or a file
read with `movie.ReadSound` from its `from`. One line says which.

`--fetch-terrain` and `--refresh-terrain`, one at a time, tend the
terrain and moon face of the set the file is, or the job or plan names,
then draw: the shelf is `SLAYER_SHELF`, the list the `sources.yaml`
nearest the set (`session.TendTerrain`). Fetch brings what is missing;
refresh fetches all again, replacing a file only with a good one.
`assess FILE --check-terrain` compares, draws nothing, exits 1 if it differs.

### Seams

The runner talks to four small interfaces in `seams.go`: a `source` of
frames, a `developer`, a `titler` and a `reel` (the movie writer), made
by a `workshop`. The real workshop makes them from the session, finish and
movie; the tests use mocks generated by mockgen where a test is about
bookkeeping, and the real workshop where it is about pixels.

### Where things go

- Paths the job says are absolute already: `job` resolves them against
  the job file.
- With no outputs said, a still goes to `out/NAME.png` and a film to
  `out/NAME.mp4`, under the working directory. NAME is the base name of
  the file given.
- Frames are working files unless the job names `outputs.frames`: a film
  that writes a movie and no frames draws them in a working folder, and
  removes them once the movie is written and closed well.
  `--keep-frames` keeps them there.
- The working folder is `frames/TEAM/NAME-` and a digest, under the
  cache home (`SLAYER_CACHE`), with `SLAYER_TEAM` the team; with no
  team it is `frames/NAME-` and a digest. Where `SLAYER_CACHE` is not
  set the draw is refused by name; there is no other place.
  The digest is twelve hex digits of the sha256 of the job file's path,
  so one job has one working folder on every day.
- The working folder is not under the renders home, so the renders page
  makes no card of it.
- With `SLAYER_RENDERS` set, outputs said with no directory (a bare name,
  which `job` resolves into the job file's own directory) and outputs not
  said go under it instead: a still in `frames/DATE-NAME/`, a film in
  `animations/DATE-NAME/`, DATE from the clock.
- A film's frames are written in that folder itself and its movie beside
  them, so the renders page shows one card for it, the movie being the
  newest file. The job file is copied into the folder as `NAME.yaml`.
- A still's raw light, a film's frames and its movie keep their own base
  names inside the folder.
- A file given from inside the renders home is in its folder already, so
  nothing is moved: its outputs go where it says, and its note names the
  output's folder.

### The note

When the job says `outputs.note` and `SLAYER_RENDERS` and `SLAYER_TEAM`
are set, one line goes into the team's `latest-notes.txt` under the
renders home: the output's folder under the renders home, a tab, and the
note. A note with a tab or a line break is refused.

The line is always appended with one write, as every other writer of
that file appends; the file is never rewritten, since that would lose a
line another writer appended meanwhile. A folder's later line stands
over its earlier one: the page takes a folder's last line.

An output outside the renders home, or no renders home, writes no note
and says so on one line. The page's script is never run.

### Resume

A job that writes frames keeps `manifest.json` in its frames folder: the
frames' key, for each frame written its file name and the sha256 of its
bytes, and the movie made from them. It is written when the folder is
made, before any frame, and again after every frame, beside itself and
renamed into place.

The frames' key is the sha256 of the job's `PictureKey`, which holds the
sha256 of every file the job reads (`job/README.md`), and the build: its
stamp, or the sha256 of the program where the linker stamped none. The
sound is not in it, but for a played scene, whose frames move with it. The movie's record holds the whole job's key and the
build, and the sha256 of the movie's bytes.

A second run skips a frame whose file is there with its hash. A folder
whose manifest has another key holds frames of an earlier version of the
job, its files or the build: the door says so, removes the frames the
manifest lists, by name and only those, and its manifest, and draws again.

It refuses, by name and removing nothing, a folder holding a picture its
manifest does not list, or PNGs and no manifest (they may be someone
else's).

Every frame, kept or drawn, is fed to the movie, a kept one read back from
disk. When every frame is kept and the movie is the one recorded, nothing
is drawn or encoded; a changed tune keeps every frame and makes the movie
again.

A frame written and not yet recorded, as a kill between the two leaves
it, is drawn again. A manifest that does not read is refused: empty the
folder or draw to a new one. Rewriting the manifest costs 3.6 ms at
6,000 frames, against seconds a frame, so it is written every frame.

Under a renders home, a bare-named film looks for a folder of its name
from an earlier day whose manifest has its frames' key, or whose movie
its working frames were begun for, and resumes it, saying so, so a film
resumed after midnight finishes where it began. A finished film whose
frames were working files keeps nothing to resume: drawn again, it is
drawn whole.

### Progress

One plain line per finished frame on the verb's output, counted by
`render/progress`'s meter, with the frame's time split into draw,
develop, titles, write and encode, and the time left once the meter
offers it. A final line for each file written, and a line of totals. No
escape bytes and no carriage returns.

### Movie sizes

The movie's settings are checked before any frame is drawn: its rate,
count and, where the job says its width and height or draws plates, its
size. A set or a smoke that leaves its size to the scene is checked at
its first frame. The movie writer is asked to pad; when it pads, one
line says so: `padded 180 rows to 192 for the film`.

### Stills and their folders

A still drawn under the renders home is refused, by its picture's name,
where its folder already holds a picture drawn from another job file
of the same name; `--replace` draws over it. The job file is copied
beside the outputs only once the run is known to be that folder's.

A still a set drew is kept on disk under its picture key, the build,
the scrooge setting and the shelf's sources list (`door/kept.go`), and served in place of drawing
only where `SLAYER_KEPT_STILLS` is `on`: the key does not yet hold every
file a set reads from beside itself, its hud theme for one. `--fresh`
draws even then, and keeps what it drew.

### Long stills

A still drawn on the processor is drawn a row at a time
(`trace.SampledKept`), and each row is kept under the cache home, kind
`rows`, as it is finished (`door/rows.go`). Its key is the job's light
key, the build, the scrooge setting, the sources list and the row's number, so rows of
another job, size, camera, rays or build are never taken up, and an
entry that does not check is a miss.

A still that is stopped, or whose
process dies, is picked up from the rows it finished when the same job
is drawn again, and the picture is the same to the byte. The rows are
removed once the still is written. `--fresh` takes up no rows and keeps
what it draws.

While it draws, a line every fifteen seconds goes to standard error:

    progress rows=412/900 taken=300 elapsed=4m10s remaining~5m2s

rows finished (taken up or drawn) of all, rows taken up, the time since
the first row, and the time left at the pace of the rows drawn in this
run, once the meter's least time and rows have passed. A last line is
said when the draw ends. Standard output is unchanged.

A still through a lens rig, of several shutter glimpses, or told ray by
ray keeps no rows and says no progress line: its rows are not one
picture's (a row's key would need the instant or glimpse in it), and a
walked ray must be told every time.

Before the rows, staging the set says where it is on standard error at
the same interval, the step and, where it places things, the thing of
how many:

    staging at=things 12/400 elapsed=1m2s

A terminate during staging stops it at the next step (between two
things, or two of its steps), and the draw ends saying where and that
nothing staged is kept. A single step is not cut in two.

A terminate or a control-c stops each row being drawn at its next ray;
that row is not kept, and the draw ends with the rows finished kept.
The same file a set reads from beside itself that the pictures key
misses is missed here: change one between a stop and a pick-up and give
`--fresh`.

### Scenes drawn as maps

A named scene of the second kind (`scene/named`'s `Maps`, the weather)
has no world and no eye, so it takes a path of its own in `maps.go`: no
source, developer or titles. Each frame's scene time is handed to the
scene, which makes its maps there (the weather's three, `spun-CLOCK`,
`bands-CLOCK`, `aged-CLOCK`, as `cmd/weather` names them).

- A film's maps go into its frames folder under those names, recorded in
  its ledger; the movie is made of the one the scene shows (`show`).
  With no `outputs.frames` they are working files, as a traced film's
  frames are, removed once the movie is written; `--keep-frames` keeps
  them.
- A still writes the map it shows to its picture.
- The scene keeps its moments in its own cache, spaced a frame apart, so
  a second run reaches each frame from a checkpoint.
- With no renderer said the scene takes its own default: the weather
  runs on the gpu where there is a device, and says so.
- Refused by name: a film size, rays, a lens, an exposure, a camera, a
  shutter, a print, a stock, titles, sound, raw light and a quality.

### Cancel

The context is looked at before each frame. A cancelled one aborts the
movie, so no partial file is left, keeps the manifest as it stands, and
gives the context's error.

### Refusals

By name: whatever `session.Open` refuses of a job (a named scene and
a job's `shutter`, `aperture` and `focus` are drawn by it now, and it
says what it will not draw), a sound with no movie, a movie
with no rate, a frames scene with fewer plates than the film's frames,
a folder of pictures the door did not write, a still over another
job's.

## watch

`slayer watch JOB` draws a job live in this terminal: package
`render/live` is the view (its README is the design), and this verb
wires it to the real pieces.

The file is read as `draw` reads it, by
`jobOf`, with the same flags laid over it by `overridesOf`; a job that
says no quality, and no `--quality`, is watched at `preview`. It is
read again each time its time or size changes, or that of the set or
title theme it names.

The stager (`stage.go`) opens a session for the job and
reads a set's cameras by name with `set.Read`. Frames already rendered
are refused by name: `draw` develops them. The screen is
`terminal.Live`; the ticks a `time.Ticker` twenty-five times a second.

The status folder is `.NAME.watch/` beside the job: `frame.png`,
`screen.png` and `status.json`, each written whole.

Output that is not a terminal is refused, saying to use `draw`; a file
that is not there is refused, saying `edit` starts one.

## edit

`slayer edit JOB` lays out three panes in the current tmux window: the
console across the foot (`carrie chat --workspace DIR`, ten rows, or a
shell in DIR where carrie is not on PATH), the editor on the right
third (`nvim` with `contrib/nvim/slayer.lua` loaded and the schema's
path given, or `vi` where nvim is not on PATH), and `watch` in the pane
`edit` was run in, which it becomes.

The tmux arguments are made by `splits`, a pure function, and run with
`os/exec`, no shell. Focus ends in the editor, the last pane split.

Outside tmux, or with no tmux, it says so, prints the three commands
and runs watch here. Each program missing from PATH is said by name
with what `edit` does without it. Nothing is installed.

A JOB that does not exist is written from the template: the modeline
naming the schema's copy, then a still of smoke 1 in portrait with the
presets listed. `SLAYER-JOB.md` (the schema's keys and two examples)
and `SCORING.md` (`cheeptune.Scoring`) are written beside the job where
absent, and left alone where present.

Bare `slayer` at a terminal is `edit` (`pzm.Bare`) on the newest job
file in the working directory, or a new `job.yaml` there.

The programs on PATH, slayer's own path and running tmux are behind
`programs` in `seams.go`, so the tests give a mock.

## schema

`slayer schema` prints the job schema. `--where` prints the path of
its copy in the cache home (`schema/job-v1.schema.json` under
`SLAYER_CACHE`, read through the door's environment; refused by name
where it is not set), written first when absent or not this build's.
`slayer.lua` is kept beside it for the editor.

## Tests with no anti-test, and why

None. Every positive is named by a marker (`// anti-test of TestX`);
`go run ./tools/antitests ./door/` counts them. Two things to know when
reading the markers:

- A pair that are mirrors of each other (inside tmux and outside it;
  no renders home and a renders home) name each other, since each is the
  other's near miss.
- `TestDrawFilmMovieNeedsMacOS` is compiled only away from macOS, and
  its marker names `TestDrawFilmWritesAMovieWithSound`, which is compiled
  only on macOS. On one machine only one of the two exists.

The anti-tests added in `anti_test.go` each answer one positive the
suite had left alone: a failure to write, overwrite or remove is told
from a success there.

- A copy whose source is gone leaves its target.
- A replace that cannot write leaves the old file.
- A stale frame that cannot be removed is named and the others still go.
- A working folder with a stranger's file keeps it and says so.
- A clear of one cache kind leaves another.
- A dry run of a clear of every kind removes nothing.

## Tests that draw

None needs a GPU, the shelf or a staged scene, and none is moved. These
draw on the CPU, at the tests' own tiny size, through the real runner
(the `drew` helper, `assess --judge`, the weather, the tune films);
their times, from `go test -count=1 -v ./door/` with `GOMAXPROCS=3`:

- `TestWatchUnderAPty` (1.77 s): runs `watch` under a pty, the one test
  that does; it also draws.
- `TestTunesStopsCleanlyWithARequestOpen` (2.00 s): waits out the
  server's shutdown with a request open; it draws nothing.
- `TestTunedLoopsAndCuts` (1.07 s), `TestTunedLeavesAOneShotAlone`
  (0.80 s): play a tune through the synth.
- `TestWhatCannotBeJudgedIsRefused` (0.98 s), `TestAssessJudgesASet`
  (0.77 s), `TestAssessJudgesASmokeAndServesItAgain` (0.71 s),
  `TestAssessRefusesWhatAFilmRefusesBeforeItsFirstFrame` (0.37 s).
- `TestDrawFramesAndMovieTwice` (0.75 s), `TestSoundFromAFile` (0.53 s),
  `TestDrawFilmWritesAMovieWithSound` (0.38 s): write mp4s, on macOS only.
- `TestTunesWritesStillsThePageAndFilms` (0.45 s),
  `TestTunesWritesStillsAndSoundsWhereOutSays` (0.23 s).
- `TestRunnerFilmBookkeeping` (0.35 s): over mocks; the time is the
  synth making its sound.
- Every other test that draws takes under 0.2 s (the `Draw`, `AStill`
  and `DrawWeather` tests); the rest do not draw.

The whole suite is about 13.6 s (`go test -count=1 ./door/`,
`GOMAXPROCS=3`).
