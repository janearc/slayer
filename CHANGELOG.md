## unreleased

- CONTRACT CHANGE, the dependency: Slayer requires celestial v0.0.9 as
  released, and `go.mod` replaces nothing. The Metal geodesic shader is
  Slayer's own, `metal.GeodesicLibrary`, with the two march numbers it
  needs beside it. celestial is not changed.
- With celestial v0.0.9 the extreme ultraviolet ends at 121 nm where it
  ended at 120, so `scene/thing`'s `Range` answers extreme ultraviolet for
  a band at 120 nm. Ultraviolet is 100 to 400 nm as before.
- KNOWN: `TestRangeEdgesAndOverlaps` in `scene/thing` is disabled for this
  release. Its row at 120 nm was written for the earlier limit.
- Tests run in four tiers: hot path, non draw, draw, and all tests.
  `game check` and `go test ./...` run the non draw tier and open no GPU
  device. `game run test-draw` runs the draw tier one package at a time.
  `doc/test-tiers.md` says how each is run and how a test is placed.
- The film compositor is in the tree, under `improv/moonlake/omnipositor`,
  with the packages it is built on (`lights`, `cues`, `ocean`, `mist`,
  `matte`, `develop`) and azazel's budget and process work. `DEBT.md` says
  what it owes, the light show told twice among them.
- One placed Kerr field renders on Metal through `gpu.Sampled`.
  `gpu/GRAVITY.md` and `doc/gravity-rc-integration.md` say what it refuses.
- The Larson's dome is set into its deck, not standing on it.
- CONTRACT CHANGE, names in a scene: three cameras of `improv/cloudsea`
  lost a prefix and are `close`, `overhead` and `south east overlook`.
- The plans and jobs of `improv/searching` and `improv/magnetar` name
  their sound, score and outputs with no machine path. A plan's song and
  score are read from beside it; a job's outputs go under a renders home
  when one is set, and beside the job when not.
- Every function, method and type has a comment above it, and notes of
  who asked for a thing and who decided it are taken out of code and docs.

- Welcoming GitHub landing page, contributing and practical testing guides,
  mathematical-model examples for the Larson and trees, and an Azazel guide.
- Local bootstrap with read-only prerequisite inspection, isolated Game
  configuration, module/source preparation and a Game build.

- New beginner and build guides, including Game's purpose and setup,
  with small runnable picture and soundtrack-film jobs.
- Expanded renderer, light, modes, optimization and music documentation,
  covering retained resources, section reuse, native traversal, field and
  light grids, streaming, compositing, gravity and render admission.

- Omnipositor can composite ordinary GPU set jobs from a sharp scenery
  plate and the scene's 3D scattered light and RGB transmission. The new
  `scene` verb exposes reduced mist resolution and explicit plate reuse,
  with a full-render comparison path. Moving cameras require fresh plates;
  this is an artistic preview mode. See its `SCENE.md` for limitations.
- Editable audio beat grids and a cue clock for supplied music, with a
  source hash, explicit estimated status and optional aligned drum stem.
- Named scenes play cues on playback time during a hyperlapse, instead of
  accelerating their music clock with scene time. Audio trims retain the
  original song's bar numbering.
- The visual metronome labels estimates and takes Pzm flags through
  `game run metronome -- draw`; its positional invocation is replaced.
- Something answers is a small multi-set music shoot: YAML song cues,
  beat-shaped continuous camera motion, optical searches, a shared lock,
  two dish fields and stylized blazar component views. Its preview command
  retains sets between frames and writes PNGs plus a native movie with the
  source soundtrack. Assembly articulation and spectral physics remain
  illustrative; see `improv/searching/README.md`.

## v0.1.2 12 november

A front door for drawing, a document that says how to draw, films
written without ffmpeg, and a synth that now lives in the tree. Also: a
live view and an editor for jobs; blur and depth of field in the kernel;
disk caches; an opt-in fast light for mist; and the shorts that explain
pictures (the cloud sea, a real magnetar, moonlake's light show), with
the tools that make them.

Also in this release, from earlier in the week: the spacecraft metals'
optics, read from the database; the yard built from materials and
finishes; what a material takes from an x-ray beam; the MCRN Larson,
landed.

Review: on the evening of 12 November seven areas were each read by a
session that wrote none of their code: the GPU kernel where four
changes met, the job package, the door with watch and edit, the render
session, the disk caches, the movie writer and reader, and the voting
server.

What they found and what was fixed is under "Review fixes",
below. Nothing else here was read that way; "landed" means merged, not
accepted.

### Contract changes

What changed in something a script, a job file or a habit could have
depended on. Each is one line; the sections after say more.

- Removed: `slayer set FILE`. Use `slayer assess FILE --judge`. The old
  word prints that line and exits 2.
- Removed: `slayer smokes` and the `smokecheck` command (and its Game
  run). Use `slayer assess --smokes [GROUP|N...]`; `checksmall` is
  `--small`, `-fresh` is `--fresh`, `-out` is `--out`. `game run
  smoke-one` now runs that.
- Removed: the `develop` command (and its Game run). Use `slayer develop
  JOB RAW...`. Differences: each raw is its own picture, `--sum` adds
  them; `-exposure`, `-knee`, `-top` and `-bloom` are the job's `print`
  keys `stops`, `knee`, `top` and `bloom`; `--white 0` no longer means
  no balance, so leave the flag out.
- Unchanged: `slayer render FILE`, still read ahead of the door in its
  own flags, and `hillside` with its own.
- New document: job-v1, `https://github.com/janearc/slayer/schemas/
  job-v1.json`. It is closed: a key it does not know is refused with its
  file, line and column. Three of its choices:
  `outputs.note` is words, not a path; `print.stops` runs from -40 to 40;
  a camera path's `unit` is the scene's own.
- Still read: sequence-v1 plans. `slayer draw` lifts one into a job with
  the same frames, and `hillside sequence` reads it as before.
- A film's frames are working files. A film that writes a movie and no
  frames removes its frames once the movie is written; name
  `outputs.frames` or pass `--keep-frames` to keep them.
- A resume key now covers the content of every file a job reads, and the
  build. A job file's text alone no longer decides it.
- Catalogue: `catalogue render` carries on past a failed specimen and a
  second run resumes the book. It used to stop at the first failure and
  refuse an existing book.
- cheeptune score format: an optional loudness mark on a note, chord or
  hit (`c5:1/2@p`); an optional top-level `tempos` list; a slide mark
  `>`; hits struck together `(k h)`. A score with none of them is read
  as before.
- cheeptune voicing format: `machine` may be `studio`, with its own
  channel kinds and a mix section. `gameboy` voicings are unchanged. On
  the studio the drum letters `k s h o t b` mean one thing in every
  voicing and no others are allowed.
- cheeptune: cut sheets, `NAME.cuts.json`, are a new kind of document.
  A library tune is named `@NAME`, and may not take a built-in's name.
- The Metal kernel's header grows from 256 to 336 bytes: five rows
  (`close_origin`, `close_right`, `close_back`, `optics`, `exposure`)
  are appended, for blur and depth of field. With
  the shutter closed and no aperture, pictures are the old kernel's,
  byte for byte: `gpu/kernel_unchanged_test.go` pins their hashes.
- One exception: a thin lens's rays are now made
  in the kernel even with the shutter closed, where the CPU made and
  sent them (it saves about 800 MB and 8 s a frame on the night fire).
  Such a picture agrees with the old within the exact tolerance, not
  byte for byte, and is pinned as it is drawn now. The CPU tracer, which
  the proof path replays, is unchanged.
- New job keys: `film.aperture` and `film.focus` (a job says both or
  neither); `vibes`, `fastlight` and `shortcuts` (`lamp-store`, `sky-
  looks`), each `on` or `off`; a set says `vibes` and `fastlight` under
  `scene`. Nothing said is the exact tracer, as before. Said, they are
  in the job's canonical form, so they change its picture key.

  The door takes a flag for each: `--vibes`, `--fastlight`, `--lamp-
  store`, `--sky-looks`.
- New verbs: `slayer cache`, `slayer watch`, `slayer edit`, `slayer
  schema`. `slayer draw` takes `--fresh`.
- Bare `slayer`, at a terminal, runs `slayer edit` on the newest job file
  in the working directory, or a new `job.yaml`.
- Removed: `cmd/tunefilms`. Its five verbs are `slayer tunes` with `--check`, `--films`, `--plates`,
  `--page` and `--sounds`; `plate DIR NAME OUT` is now `--plates`, which
  writes every still, or the names given, into a folder.
- New key of a mist: `asymmetry` (forward scattering of droplets), in
  the set reader and both schemas. Unsaid or nought is as before.
- `trace` and `scene/thing`: a field may say `LetGo`, the radius in
  masses past which outgoing light is let go straight. Nought keeps the
  old 12. Smoke 36 is byte for byte as it was.
- A set's `weather` block may say `pace` and a turning, so a planet
  alone is drawn as a film. Unsaid, every frame is the still.
- Frames of a film of maps (the weather) are working files too.
- A pinned file (the elevation model) is re-hashed only when its path,
  size, time, inode or device changed since the last run. See "Disk
  caches".
- Set files: `fizzes` is a new key of a world, and `setting: space` is
  now drawn by hillside (a space set refuses ground, a planet, recipes
  and burning things by name). Every other v2 key hillside does not
  draw is refused by its own name, as before.

### One front door

`slayer` is the one command to draw a job. It has nine verbs: `draw`,
`list`, `tunes`, `assess`, `develop`, `cache`, `watch`, `edit` and
`schema`. Bare `slayer` at a terminal is `edit`.

- `slayer draw FILE` draws a job, a sequence plan or a set file here:
  frames, a movie with its sound, titles and a stock. A set file drawn
  directly is a still with the set's own defaults. The flags `--out`,
  `--width`, `--height`, `--rays`, `--at`, `--camera`, `--renderer` and
  `--quality`, and for each flag and shortcut `--vibes`, `--fastlight`,
  `--lamp-store` and `--sky-looks` (`on` or `off`), stand over the job.

  `--replace` draws a still over another job's; `--keep-frames` keeps a
  movie's working frames; `--fresh` draws a still even where the cache
  holds its picture; `--dry-run` prints the job in its canonical form
  and each file it would write, and draws nothing.
- `slayer assess FILE` says whether a file will draw, without drawing:
  the same reading, flags and refusals as `draw`, then each piece `draw`
  uses is opened and let go. `--judge` is what `slayer set` did (a set
  drawn as the terminal shows it and as a picture, the one judged against
  the other) and `--smokes` is what `slayer smokes` and `smokecheck` did.
  A judgement that is not honest fails.
- `slayer list [HEADING]` says what there is: scenes, smokes, presets,
  stocks, tunes, voicings, library and kinds, each read from the
  registry that owns it, never from a list kept in the door. `--json`
  writes the same for a program.
- `slayer develop JOB RAW...` develops light already drawn through a
  job's print or stock, with its titles at each frame's film time.
- `slayer tunes DIR` serves a tune library's voting page, and writes the
  library's films, stills and page with a switch. See "Hearing tunes
  and keeping them", below.
- `slayer cache` says what the disk caches hold; `slayer cache --clear
  [KIND]` empties them. See "Disk caches".
- `slayer watch JOB`, `slayer edit JOB` and `slayer schema [--where]`.
  See "Watch and edit a job".

To make a short: `SLAYER_RENDERS=... SLAYER_TEAM=... slayer draw JOB`.
Under a renders home a still goes into `frames/DATE-NAME/` and a film
into `animations/DATE-NAME/`, the job file is copied beside them, and
one line of the job's `outputs.note` goes into the team's notes file.
Control-c stops between frames, throws the movie away and leaves the
frames to resume. A second run resumes.

- Verbs return errors and say them in words; nothing in the door exits
  the process but `main`.
- `slayer draw` says one plain line for each finished frame, with the
  time split into draw, develop, titles, write and encode, and
  the time left once the meter can say it. No escape bytes.
- Not here: `slayer send`, which `job/README.md` names. `slayer render
  FILE` is not on the door yet; it is still read ahead of it, in its own
  flags.

### A job says how to render a scene

The `job` package reads and checks job-v1: which scene, the film's size,
rays and lens, the renderer, the time or span, the camera, the
development, the sound and titles laid over it, and what comes out.
`jobs/` holds nine examples. The package draws nothing and imports no
renderer.

- A scene is one of `{set: PATH}`, `{smoke: N}`, `{named: NAME, with:
  {...}}` or `{frames: DIR}` (frames already drawn; only development,
  titles, sound and the film are done).
- Precedence is the command line, then the job, then the set, then the
  scene's own default.
- `film` takes a preset (`portrait`, `portrait-preview`, `4k`, `2k`,
  `720`, `320`, `imax`, `imax-2k`, `video-preview`, `video-preview-720`;
  `slayer list presets` says each size and rate). Width and height run
  from 1 to 8192, rays from 1 to 16.
- `stock` develops through a ninjafilm stock: `soot`, `ash`, `smoke`,
  `dusk`, `pollen` or `lantern`, with a seed and a push or pull. This is
  the first time a command reaches a stock: one roll a job, grain fixed
  by seed and frame. A stock print at 1080 by 1920 costs about 0.6 s a
  frame (pollen 3.9 s).
- A job's `Key` is the sha256 of its canonical form, with the sha256 of
  every file it reads. `PictureKey` leaves out the sound, so a changed
  tune remakes the movie from the frames already drawn.
- Camera paths may say the scene's own unit, so a path can be written in
  the orbital's thousand kilometres.
- Not here: files a set itself reads (terrain, assets) are not in the
  key. A set whose terrain changes under it resumes frames drawn from
  the old terrain.

### A scene drawn a frame at a time

`hillside.Open` stages a job's scene once, and its frames are drawn one
at a time at any size, as undeveloped light. The caller develops them.

- A job's width, height, rays, `wide` or `tall`, exposure, renderer and
  camera stand over the set's. A smoke drawn from a job takes the job's
  renderer; a smoke with bands, a gauge or a palette is refused by name.
- A still takes a camera path. A film's shutter draws glimpses across the
  open interval, and `Drawn.Light` is their mean, with `Drawn.Glimpses`
  beside it.
- A film is drawn through a set's lens rig when the set has one: each
  glimpse at its own time, the lens refocused from each eye, the frame
  the lens's exposure. A job's shutter beside a rig is refused by name.
  `jobs/moonlake-film.yaml` draws this way.
- A session draws the named scenes (swarm, orbital, ring, moonrise) from
  a job, equal to their commands.
- On the GPU a film keeps its kernel, its inputs and its hierarchy on the
  device across frames and sends only what changed (`gpu.Frames`). When
  only the camera moved it sends the header alone. It is proved bit for
  bit against the per-call path, which stays as the reference. `hillside
  sequence` prints each GPU frame's stage times.
- A set with `setting: space` is drawn: manufactured craft placed by
  east, north and height with no ground, a fixed camera, the sun as one
  lamp at any altitude and a black sky. `sets/webb-at-l2.yaml` is smoke
  50 as a set.
- Not here: a session cannot yet draw frames already rendered (the door
  does that through `render/finish`). The tracers do not report rows, so
  the progress sink is told once a frame, with the whole frame as one
  band.

### Frames are working files

A film that writes a movie and no frames draws its frames into a working
folder and removes them when the movie is written and closed well: frames are not
kept when there is a video and they are not hard to make again.

- The folder is `cache/TEAM/NAME-DIGEST` under the renders home when
  `SLAYER_RENDERS` and `SLAYER_TEAM` are set, else `slayer-frames/` in
  `TMPDIR`. The digest is the first twelve hex digits of the sha256 of
  the job file's path, so one job has one folder on every day.
- `manifest.json` in the folder holds the frames' key, each frame's
  name and sha256, and the movie made from them. It is written before
  any frame and after every frame. Rewriting it costs 3.6 ms at 6,000
  frames.
- A second run keeps a frame whose file is there with its hash and draws
  the rest. A folder with another key holds frames of an earlier version
  of the job, its files or the build: the door says so, removes the frames
  the manifest lists and only those, and draws again.
- It refuses, by name and removing nothing, a folder with a picture the
  manifest does not list. A manifest that does not read is refused.
- A still drawn under the renders home is refused where its folder
  already holds a picture from another job file of the same name;
  `--replace` draws over it. A resume finds an earlier day's folder, so a
  film resumed after midnight finishes where it began.
- Notes are appended with one write and never rewritten; a folder's last
  line stands.

### Movies without ffmpeg

`interfaces/movie` writes H.264 and AAC mp4s through AVFoundation, with
no ffmpeg and no shell, and reads them back.

- The coded height is a multiple of 16: another height is refused by
  name, or padded at the foot when the request says so (the door says
  `padded 180 rows to 192 for the film`). The width is even, the colour
  is BT.709, a keyframe comes at least every 48 frames, the level is the
  lowest that fits and at most 5.2, and the index is at the front. The
  file is published on close; an abort leaves nothing.
- `Probe` reads a file's facts back, `ReadSound` reads any sound file
  the system reads, and `OpenFilm`, `Film.Next` and `FrameAt` read a
  film's pictures. A 4096 by 3008 HEVC film reads at about 5 ms a picture
  scaled to 1080 across.
- Only on macOS. Elsewhere every call that touches a file returns
  `ErrNeedsMacOS`.

### Faster frames

Each was measured by the slice that made it, on a shared laptop, and was
not run again for this note.

- `maths/noise`: `Gradient` is 9.1 times faster (a table in place of a
  twelve-way switch, the corner hash shared across a cell), every value
  unchanged to the bit.

  The night-fire film's per-frame encode halves at three cores (1.66 s
  to 0.84 s); a 320 by 180 film frame goes from 2.66 s to 1.79 s on the
  retained path; a 1080 by 1920 frame from 4.35 s to 3.54 s; orbital CPU
  frames are 3.5 times faster; the gpu test suite runs in 86 s where it
  took 164 s.
- `gpu.Frames`: on the night-fire set, a camera-only frame at 320 by 180
  goes from 2.73 s to 0.18 s. A film frame, whose time changes, goes from
  2.82 s to 2.64 s.
- `gpu`: the standings' tree is sorted once an encode instead of three
  times: about 190 ms a night-fire frame at three cores, by profile.
- `maths/noise`: `Fractals` asks one point for several seeds, bit for bit
  `Fractal`; plume's eddies use it: about 70 ms a frame, by profile.
- Orbital gains `-gpu`: one 640 by 360 frame of the near view takes 39 s
  on the CPU and 1 s on the GPU.

### The catalogue carries on, and has a book

- `catalogue render` carries on past a failed specimen: the card is
  marked failed with its reason and the run ends non-zero naming every
  failure. A second run on the same `--out` resumes the book, keeping
  cards drawn under the same plan. `pdf` names the cards that keep a book
  from composing.
- `catalogue pdf` composes a cover, a contents page and one page per
  specimen with its picture, description, notes, references as links,
  provenance and page numbers, with an outline, set in Helvetica. It
  refuses a character outside WinAnsi by name. `--with-failures` gives an
  unrendered card a page saying why.
- The menu holds 106 kinds. A whole book draws in about 20 seconds and
  the PDF is 113 pages, 2.4 MB.

### The synth, in the tree

`cheeptune` is the synth `tune` (its main at 5a0d353) brought in as one
package: score and voicing parsing, rendering to samples and a WAV, a
playhead that says the beat and bar at any moment, the built-in scores
and voicings, and a piano-roll picture. `cmd/tune` was left behind. The
name `cheep` is held for a later library. Writing and choosing music is
in `doc/music.md`.

- Two machines. The Game Boy machine is four channels simulated at the
  register, byte for byte as it was. The studio is new: FM, a filtered
  analogue voice with a four-pole ladder filter, a plucked string,
  struck objects (modal), synthesised drums and struck-metal drums, real
  chords, a reverb and a delay, up to sixteen channels.
- Marks in a score: loudness `@pp` to `@ff`, tempo changes at bars
  (`tempos`), slides (`c3:1/4>`), hits struck together `(k s)`. The
  studio plays them; the Game Boy machine reads loudness and slides and
  plays the score as if they were not there.
- Eight studio voicings are built in (`@acid`, `@acid-clockwork`,
  `@glass-bells`, `@glass-struck`, `@night-pad`, `@analog-drive`,
  `@wire-harp`, `@iron-works`) beside seven for the Game Boy. The acid
  voice has accents that open the filter, fold, overdrive and slides.
- Cut sheets, `NAME.cuts.json`: mood, feeling, intent, parts with bars
  and seconds, and the voicing, read and checked against their score. A
  library of 33 tunes written for videos is compiled in, each named
  `@NAME`; `OpenTune` and `Library` read it. `VIDEO.md` teaches writing
  a tune for a video, `SOUNDS.md` designing an instrument and
  `SCORING.md` the format.
- A job names a library tune as `sound: {tune: "@tune-name", voicing:
  "@voicing-name"}`, the voicing its cut sheet names. A tune that loops is
  played again to the film's end and cut there; one that does not is
  followed by silence.
- Not here: a gradual tempo change (write one change a bar). The
  studio's engines were measured by the sessions that wrote them (peaks,
  spectra, lengths), since a session cannot hear; how they sound is for
  a listener to say.

### Hearing tunes and keeping them

`cheeptune/audition` makes, for a folder of tunes, a film of each tune's
piano roll with its parts marked and named as they begin, and a page to
hear each and keep or toss it. `slayer tunes DIR` serves the page on
this machine only (`127.0.0.1:8417` unless `--address` says another
loopback address) and records votes in `votes.json` in the folder.

- A vote is `keep`, `toss` or `clear`. A vote moves nothing on disk. It
  needs JSON and this machine's host and port, which shuts out a web page
  in a browser posting to it. It does not shut out another program on
  the machine: there is no login.
- A tune added while serving can be voted on, and a vote for a tune gone
  from the folder can be cleared. The page is written from scores and
  cut sheets and renders nothing; instruments lead the page.
- Without serving: `--check` (length, peak, voicing; fails if any will
  not load), `--films` (`NAME.mp4`), `--plates` (`NAME.png`), `--page`
  (`index.html`), and `--sounds` with voicing names for a sheet with one note of
  each instrument.
- The library began as the 23 tunes Ten have been added since (zappy's
  theme, `glass-acid` and `glass-chase`; see "Zappy's theme"), 33 in all.

### Moonlake's explainer

`improv/moonlake/show` reads moonlake's score and cue sheet, moved out
of `cueshow` and `metronome` so three commands share one reading.
`explain` composes the explainer frame, the finished film with a panel
under it that shows the beat, the lamps' cues, the chant and one plain
sentence; `explainer` is its command and writes the shorts.

`lightshow`'s output is unchanged. Not here: the transcribed chant notes
time a line too sparsely, so the song's own lyric track times it.

### Watch and edit a job

`slayer watch JOB` draws a job live in this terminal and draws it again
when the job file, the set it names or its title theme is saved.
`slayer edit JOB` lays out three panes in the current tmux window: the
agent console across the foot, the editor on the right third and watch
where `edit` was run. Bare `slayer` at a terminal is `edit`.

`slayer schema` prints the job schema, or with `--where` the path of its copy
in the user's cache folder.

- `render/live` is the view. It draws the frame small at once and then
  finer: a ladder of sizes and rays at preview quality, a rung whose
  predicted time passes 15 seconds not started. A key or a save while a
  rung draws cancels it. A job that does not read leaves its error on
  the foot row and keeps the last good picture. The status row says
  when a picture is a preview.
- Keys: left and right or `[` and `]` step a film; space plays it; `c`
  changes the camera; `e` and `E` change the exposure a third of a stop;
  `f` draws the frame full size; `s` writes the moment, camera and
  exposure back into the job file, comments kept; `?` is help; `q` or
  control-c quits.
- `.NAME.watch/` beside the job holds `frame.png`, `screen.png` and
  `status.json` (`slayer-watch-status-v1`), each renamed into place, for
  a graphical front end to poll. `render/live/README.md` lists the keys of
  `status.json`.
- `contrib/nvim/slayer.lua`, loaded by `edit`, starts the YAML language
  server with the job schema, rereads the file as watch or the agent
  saves it, and maps leader-d to `slayer draw %`.
- A job that is not there is written from a template, with `SLAYER-JOB.md`
  (the schema's keys) and `SCORING.md` beside it.
- `edit` needs tmux and nvim; hover needs `yaml-language-server`; the
  console is carrie. Neither of the last two is on this machine's PATH,
  and `edit` says so by name and does without. Outside tmux it prints
  the three commands and runs watch.
- Not here: a first picture of the night fire takes about 18 s (the
  scene opens first). The tmux layout was never run in a real tmux. `f` shows the frame when it is done, not row by row. `s`
  cannot save a film's frame position. Frames already rendered are
  refused by name; `draw` develops them.

### Blur and depth of field

A film may say `film: {aperture: A, focus: F}`, both in the scene's own
unit, and a shutter, and the kernel makes both at the primary ray: each
sample of a pixel takes its own time in the shutter and its own place on
the aperture. A film frame used to have no depth of field, and its
motion blur cost a whole render a glimpse. Now N samples a pixel give
both for about the cost of N samples.

- `rays: 4` or more reads as blur; fewer is noise.
- Only the eye moves. A thing that moves while the shutter is open is not
  blurred here; glimpses (several draws averaged) still do that.
- A set whose film has a rig takes its lens from the rig, and an
  aperture over it is refused by name. `jobs/blur-fire-night.yaml` is
  the night fire through a lens and an open shutter.
- With the shutter closed and no aperture the picture is the old one,
  byte for byte. The kernel's header grew by five rows; see "Contract
  changes".
- Design note: the head of `gpu/exposure.go`.

### Disk caches

`diskcache` is the one place Slayer keeps what it worked out before. A
file an entry, written whole and renamed into place, with its key's
record, its length and its sha256; an entry that does not check is a
miss and is never served. Every entry is exact: a hit is what a fresh run
computes, byte for byte, and a change of one byte of what it was made
from is a miss. `doc/caches.md` has the detail.

- Kinds today: `skies` (the moonlit and twilit sky tables), `ground` (a
  set's elevation window, cut and smoothed), `pictures` (a set's finished
  still), `pin-checks` (the sha256 of a pinned file, by path, size,
  time, inode and device, so the 1.8 GB elevation model is not hashed
  whole each run) and `trees` (the GPU's trees, one file each, when
  `SLAYER_TREE_STORE=files`).
- `slayer cache` says each kind's folder, entries, bytes against its cap
  and oldest and newest use; `slayer cache --clear [KIND]` empties them;
  `--dry-run` says what it would remove.
- `slayer draw --fresh` draws a still even where the cache holds its
  picture.
- Home: `SLAYER_CACHE`, else `cache/` under `SLAYER_RENDERS`, else the
  user's cache folder.
- The GPU's tree cache is still one bbolt file, `SLAYER_TREE_CACHE`.
  `SLAYER_TREE_STORE` picks how trees are kept: `bolt` (unset; the one
  file), `files` (a file a tree in `diskcache`) or `pids` (each run
  writes its own file and merges at its end, so no reader waits for a
  writer). The three were put in so they can be measured against each
  other; none has been chosen.
- A plain glow table of the flame is made once a process and shared.
- Not here: files a set reads are in the pin checks but not in the
  frames key (see "Found, not fixed"). The tree cache's bolt file is
  not capped and has never shrunk (647 MB on 12 November).

### Fast light

An opt-in way to draw mist and smoke in less time, for pictures to watch
and not to measure. GPU only. Off unless said. A picture drawn with it
says so: in the job's canonical form, in the copy of the job beside the
frames, and in a `fast light:` line.

- Three keys: `vibes: on`, `fastlight: on` and `shortcuts:` with
  `lamp-store` and `sky-looks`. A shortcut is decided by the first that
  says anything: its own word, then `fastlight`, then `vibes`, each read
  from the command line, then the job, then the set. `vibes` is the flag
  for every flag; `fastlight` is the flag for the light shortcuts.
- `lamp-store`: mist and smoke lit from a store of each lamp's light,
  baked on the device once a frame, in place of a march to every lamp at
  every step. A shadow cast into mist is as soft as a cell of the store
  (12.5 m on the cloud sea).
- `sky-looks`: the sixteen looks at the sky from a surface point are
  shared among a pixel's rays. Needs `lamp-store`. At one ray a pixel
  nothing changes.
- Refused by name: on the CPU; through a rig's lens; for a named scene;
  beside `-prove`; `lamp-store` with more than eight lamps or beside
  `volume-lighting: cached`; `sky-looks` without `lamp-store`.
- Measured on the cloud sea's hero camera, 1920 by 1080, four rays a
  pixel, on a GPU shared with a dozen other renders, two frames drawn
  one after the other: exact 115.8 s of trace, 136 s for the whole
  command; lamp store 46.6 s of trace, 89 s for the whole command. About
  two and a half times on the trace.

  Earlier the same day the same exact frame traced in 41 to 54 s, so
  read the ratio and not the seconds. The two pictures differ by 1.98
  percent of the exact picture's light in sum, all of it in the mist
  between the trees.
- Not here: the sky-looks timing is not in `doc/fast-light.md` yet.
  `disc-shadows` was built and taken out; it saved nothing. Cascades
  were not built: the cloud sea has five lamps and a store for each is
  simpler and nearly exact. The stricter replay path called
  science takes no shortcut; see `doc/flags.md`.
- Not here: `mist-once`, `stop-early` and `dither` were proposed as
  further parts of `vibes`. None is in the tree, and `vibes` holds only
  the two shortcuts above.

### Named scenes: zappy, the magnetar and the weather

- `zappy`: the lantern grown up, a star in nested shells of black-glass
  panels that tumble, with two beams that spin and precess, in a thin
  haze. It has channels (`lantern/zappy-channels.yaml`) that music can
  move: `scene: {named: zappy, play: SHEET}` with a tune as the job's
  sound; `cuesheet` plays them.

  With every channel at rest the picture is zappy unplayed, byte for
  byte. A shell's kept mesh trees are reused between frames (`gpu`: a
  frame of a film that only turns and tumbles its shells used to grow
  every ring's face tree again, 3.5 to 5 s of the encode).
- `magnetar`: SGR 1806-20 at its true size, 10 km across: its field
  drawn turning, hoops where the field falls to named strengths, the 2004
  giant flare (the times are measured, the brightnesses picked) and
  zappy beside it at its own 9,000 km. Every number it draws by is read
  from `research` with its source, or computed from those. It draws at
  eight rays by default.
- `weather`: a planet's air stirred by storms, drawn as three maps
  (spin, bands, age), a second kind of named scene with no world and no
  eye. `slayer draw` draws it from a job; `cmd/weather` is thin over it.
  It runs on the GPU where there is one and no renderer is said. A
  second of scene time is an hour of weather unless `pace` says another.

  `sets/weather-giant.yaml` is Melancholia's giant alone, its weather
  run from still air to 10^7 s over twelve seconds, the planet turning
  once.
- Not here: each of zappy's beams on or off alone (the kernel draws both
  poles of a polar emitter alike, and that is in `gpu/trace.metal`); the
  star's glow does not follow its temperature, only its light does.

### Shorts that explain a picture

Short vertical films with a picture on top, labels on it pointing at real
places, and a plain panel under it with two or three numbers and a
sentence. Each number says where it comes from.

- `improv/walk`: the library. A script of shots with labels at places of
  a set, or at pixels, or at points of a scene that is not a set; scale
  bars; a facts panel; a rigless film is the pinhole hillside draws.
- `improv/host`: a host laid over any film: clips by name, a seeded drift
  inside a box, jumps and a lift on the kicks, an intro, the tune's own
  sound. A frame drawn twice is the same frame.
- `improv/moonlake/untraced`: the light show in four panes in step with
  the song, the finished film, the explainer, and two drawn views of the
  rig. `-passes` traces the two views' passes and sums them at the cue
  levels. `improv/moonlake/rig` is the light show's geometry and pass sum,
  moved out of `lightshow` so the drawings use the same functions; the
  light show's output is byte for byte as it was.
- `improv/cloudsea`: the cloud sea with its guide stars corrected
  (the dome turned so its slit faces the moon; four launch tubes on the
  slit's rim; one aim; GALACSI's 64 arc seconds of offsets; 30 cm beams
  from ESO's page), a walkthrough script, and seven shorts over glass
  hour, each from its own bar.
- `improv/magnetar`: the magnetar's short, in the same form.
- `improv/pan`, command `improv/pan/cmd/pan write SCRIPT --out FILM.mp4
  [--dry-run]`: a film of stills, each moved under a window the film's
  size (a real crop, never scaled up), cut on a tune's bar lines, with
  optional fades and a small title. For a quick upload when tracing
  motion would cost too much.
- `improv/quasar`: the arithmetic behind the photon ring shorts: `rings`
  (which image of the disc a ray brings back and how thin it is),
  `facts` (every number a sentence says, from `research` or computed; a
  sentence naming a number neither gives is refused), a diagram, the
  pull-back's steps and the commands that compose them.

  The photon ring short itself was dropped on 12 November: the pictures
  were plain. `improv/quasar/NEXT.md` is for whoever builds a better
  looking model.
- `research` gains the values these read: the photon ring, M87, Sgr A*,
  TON 618, Porphyrion, Neptune's axis, a real magnetar, the sun's and the
  earth's nominal radii, the eye's sharp middle.
- Not here: the cloud sea's beams have no glow (that needs multiple
  scattering) and stop at 1 km where a real one reaches about 90 km; the
  1.6 km ground edge shows from high cameras; the air above the deck
  thins over the deck's depth, not a scale height. Moonlake's views stay
  black with the print pushed 20 to 26 stops.

  `untraced`'s other cameras are drawn, not rendered, for the beams,
  haze and water; rendering them from another camera would take a
  general camera in `lightshow`.

### Zappy's theme and other tunes

Ten tunes joined the library.

`zappy-ident` and `zappy-ident-falling`, three beds (`zappy-bed-
curious`, `-ominous`, `-awe`) and three big pieces (`zappy-engine`,
`zappy-big-minor`, `zappy-big-phrygian`) are written around one motif of
five notes, each to a plan in its score's comment, and a test holds that
each states the motif where its plan says.

`glass-acid` and `glass-chase` join `glass-hour`. Eleven candidate
voicings for the studio, measured on one riff, are in
`cheeptune/candidates` with `WANTED.md` files for what the studio could
not do; they are not built in. `cuesheet` gains tempo steps, steps that
count and follows that breathe. Not looked into: how any of it sounds.

### Smaller, later

- `finish`: a stock's screen rolls highlights off instead of clipping
  each colour at white. The night fire through lantern had come out one
  flat yellow at every rating. A picture whose luminance stays under the
  screen's knee prints byte for byte as before through all six stocks.
  `ninjafilm`: a roll says the clearest its print passes.

  Not fixed: pushed three stops the fire is far past the end of
  lantern's curve, so its inside prints pale and nearly flat.
- `scene/thing`, `trace`: a field's `LetGo`, see "Contract changes".
- `trace`, `gpu`, `set`: a prescribed mist's `asymmetry`, a forward lobe
  for droplets (Jendersie and d'Eon 2023). On the GPU it adds no field or
  row. The shipped cloud sea leaves it off: one scatter with the moon
  behind the camera darkens the sea.
- `hillside`: a session says the size it draws at and its exposure; a
  smoke's open shutter is tested.
- `terrain`: a window as bytes and back, the same to the bit.
- `sources`: a shelf may check a file's pin by a hash it is given.
- `terminal`: `Live.Foot` writes lines on the last rows.

### Smaller

- `render/finish`: the darkroom end of a film. Development through a plain
  tone or a stock, the mean of a frame's glimpses, titles with a span,
  frames read from a directory, and the cut that fits them to a film's
  shape.
- `render/progress`: bands of finished rows, a meter with an honest time
  remaining, plain and human lines, a throttle.
- `terminal`: `Live` (size, resize events, decoded keys, a clipped
  present, a status row), a key decoder and `LeaveOnSignal`. Byte 0x1d is
  reported as its own key and never bound.
- `hud`: an element can carry a span of film time to be drawn in;
  `DefaultTheme` is light ink on a dark ground at 11.7 to 1.
- `scene/named`: the swarm, the orbital, the ring and moonrise are built
  through a registry with typed parameters, and their commands draw the
  same pictures byte for byte. `ring` refuses `-around` below 8 instead
  of drawing 8.
- `render/smokejudge`: the six smokes that are not honest are named in
  its test, with why smokes 49 and 50 are not.

### Found, not fixed

- Smoke 36's eye is at 50 masses, outside the field's reach of 40, so
  the tracer draws light straight from the eye to 40 masses before it
  starts bending it. The shadow's edge lands about 6 px low at 1080. A
  reach of 60 costs nothing and removes it and the faint creases inside
  the disc.

  A stair-step on the disc's outer rim against the sky comes from the
  let go of outgoing light past 12 masses or the step length (constants
  in `trace/field.go`). An existing numbered smoke is never changed in
  passing, so smoke 36 is as it was.
- GPU frames differ by 2 to 6 bytes between runs on some frames. At pixel
  (106, 16) of `doc/fire-night.yaml` at 320 by 180, one frame in six
  differs in depth by one unit in the last place; a fresh kernel every
  frame does too. `DEBT.md` holds the report.
- Frames' resume key does not cover files a set itself reads.
- Cloud sea, 1920 by 1080, no mist, four rays a pixel: 0.7 s of trace
  with no trees and 22 to 31 s with them, on the shared GPU. An earlier
  reading blamed closest-hit queries through overlapping instances; it
  was wrong and is withdrawn.

  Measured since: first hit not asked 0.05 s; first hit found and
  nothing shaded 1.5 s; skylight returning nought 3.7 s; whole frame 28
  to 31 s. The cost is the sky's sixteen shadow rays round the normal at
  every surface point of every camera sample (`skylit` in
  `gpu/trace.metal`), about 133 million a frame and 24 to 27 s of it.

  It is not the hierarchy. `sky-looks` shares the sixteen among a
  pixel's rays; the exact picture still pays them all.
- No key covers the files a set reads in turn (terrain, shelf assets,
  its hud theme): a film resumes old frames after one of them changes.
  Mending it needs the set package to say what it read. Until then a
  kept still is not served by default (see "Review fixes").
- A pinned file's re-hash is decided by its size, time, inode and
  device. A file replaced in place with all four kept is taken as
  unchanged.
- Cloud sea (for `DEBT.md`): beams have no glow and stop at 1 km; the
  1.6 km ground edge shows from high cameras (a wider window is a
  setting, but stands and mist follow it; an edge fade is code); air
  above the deck thins over the deck's depth.
- Moonlake's views stay black with the print pushed 20 to 26 stops; the
  crater-moonrise set refuses a 16:9 job because its rig fixes the
  frame's shape.
- Owed: the frame-time table after the two cuts in "Faster frames"
  (`slayer-frames/scratch` holds the slice's harness) is not re-run.
  Lint on `cheeptune` was not run by the slice that brought it in (it
  timed out).
- The orbital's `-cells` still draws on the CPU and refuses `-gpu`.

### Review fixes

- A played scene's cue sheet is an input whose bytes are in the key,
  and its sound stays in its picture key: frames drawn to old music are
  no longer taken for new.
- A job whose canonical form cannot be written as JSON is refused; it
  used to get the key of nothing, shared with every other such job.
- A kept still is served only where `SLAYER_KEPT_STILLS` is `on`.
  Stills are kept either way.
- `slayer watch` gives the terminal back when the view or a frame
  panics.
- A rig's film refuses fast light when it opens; it used to draw exact
  light and say nothing. A cancel stops between a shutter's glimpses.
- The movie writer's finish waits a minute a thousand frames, not one
  minute: a long film's final rewrite could time out and be thrown away.
  Seeking allows a nanosecond, so a moment at a frame's edge shows that
  frame.
- The build's digest is taken as the binary starts.
- The sky's looks are shared only where they are dealt evenly: one, two,
  four, eight or sixteen samples along a side.

Left, with their reasons in the summary:

- Two job files of one name drawn on one day share a dated folder, and a
  film of the second draws over the first's movie. A claim by the job
  file's bytes was tried and taken out again: it refused the same job
  redrawn after an edit. Claiming by the file's path is owed.
- Low findings in the voting server and the movie writer.
- Refusals the session makes at the first frame rather than at open.
- The door's handling of a second control-c.
- Where a job's outputs may point.

### A fizz a set file can say

A set's world takes a new list, `fizzes`: the mist of bubbles too small
to see, as smoke forty-three has it, a column on its foot that thins as
it climbs and widens. Each says its colour, foot, radius, height, thick,
rise, churn and seed.

The set package reads it, both schemas hold it, and stage builds it as
`medium.Fizz`, which the tracer and the GPU already move on with the
clock. `sets/fizz.yaml` is the smoke's column as a set, and `slayer
assess sets/fizz.yaml --judge --at 2s` draws it two seconds on and
judges it.

(`slayer set`, which this was written for, is gone; see "Contract
changes".)

- Only the churn climbs; the column stands. A fizz with a churn of
  nought, or a rise of nought, is one picture at every moment, and may
  be said.
- Not here: the glass, the drink and the bubbles big enough to see. A
  set cannot say a clear thing or a turned shape yet, so smoke
  fifty-two's glass of cola is still a smoke.
- No verb films a set that says only a world: `hillside sequence`
  refuses one with "sequence currently requires ground scenery". The
  film of this fizz was drawn a moment at a time with `slayer set -at`, now `slayer assess --judge
  --at`.
- Drawn at a hundred moments 40 ms apart with `-k 1`, the set verb
  called the terminal picture honest at 92 and not honest at 8, each of
  the 8 for one pixel at the plume's edge, 0.030 off. At `-k 2` the two
  moments tried were honest. Not looked into.

### The GPU draws the studio

The three glass smokes, 49, 51 and 52, could not be drawn on the GPU at
all. Asked to, the kernel refused them twice, and neither refusal was
about glass.

- **A spot.** The studio is lit by a `film.Spot`, which the GPU refused
  by name. A spot is a point emitter kept to a cone, which the kernel
  already drew: the same place, the same inverse square, and the same
  smooth edge written the other way up. The encoder hands the kernel a
  spot as that emitter and the kernel learned nothing.

  On the GPU the two are one picture to the bit; in the tracer they
  agree to rounding. A spot soft all the way to its middle, or aimed no
  way, is refused by name, since the cone cannot say it.
- **A plain mesh shaded smooth.** A `shape.Mesh` with normals and
  `Smooth` was refused. It is now bent by its corners' normals, as
  slayer's own mesh is, with one difference the tracer has too: slayer's
  mesh is also lifted toward the surface its normals draw, and a plain
  mesh is not.

  A smooth face says which it is in a spare number of its first normal.
  A smooth mesh whose normals are no longer one to a vertex is refused
  by a name of its own. In a group a smooth mesh is still refused.
- **The three smokes, drawn both ways.** The glass of water, with its
  turned tumbler that outranks the water, is 3.5e-7 off the tracer on
  average; the ball of water with bubbles 1.3e-6; the cola 2.6e-6, with
  one pixel of 6,144 past the limit, where a ray grazes the rim of a
  fizz ball. Nothing drew a smoke on the GPU before this.
- Not done: nothing draws a smoke on the GPU outside these tests. The
  smoke check still uses the tracer, and a smoke still has no moment,
  so the fizz does not yet move.

### The yaml a picture was drawn from

- The catalogue's book links each specimen to the settings it was drawn
  with, as `settings-yaml.txt` beside its `settings.yaml`: the same
  bytes under a name a browser shows, where it downloads a yaml and
  shows nothing. The link reads "The yaml it was drawn from". A book
  drawn before this still links the yaml.

### Held to the bar

Nothing here changes a picture. It is what the tests found.

- By each package's own tests `scene/sky`, `scene/thing` and
  `materials` run every statement; `trace` runs 98.3 percent, and all
  of `surface.go`; `cmd/catalogue` 94.5. `DEBT.md` names every
  statement left and why.
- The lamps' highlights and the wide sun's soft shadow had no test in
  the tracer, only pictures. A highlight is now held to the lamp's
  light times what the surface mirrors times its lobe, to the seven
  figures a pixel keeps; a soft shadow to the share of the sun's eight
  ways that pass what stands in them.
- The star catalogue's index has no mutant left alive: 151 killed and
  12 let by with a reason each. Its sum test failed about one run in
  forty, by disagreeing with the catalogue in the last bit about a
  star exactly on a disc's edge; it allows either answer within 1e-12
  of the edge now.
- A material is a page of data and one function makes them all, where
  each of thirteen had a function of its own with errors it could not
  have. No reading moved.
- Two comments said what was not so. Silver does not mirror "0.98 or
  more of every colour": its blue is 0.9786. Aluminium is not "about
  0.91 of every colour": its blue is 0.919.
- Every test file this work wrote has an anti-test for every test.

### Webb, lit by the Larson

- Smoke 50: Webb in its own frame with the sun 135 degrees from its
  line of sight, on the shield's side, as it stands at 20:53 UTC on
  2027-05-22 with its mirror on Gaia DR3 4081136684118965760; and the
  MCRN Larson hovering over it, in Webb's shade, its lamps lit. It is
  the first smoke made of the manufactory's things: `stood` makes a
  kind, stands it in a world, and lights whatever lamps it carries.
- Not right yet. Webb's place on its orbit around L2 is not drawn. The
  gold mirror shows black, which is what a mirror facing empty space
  shows, but for three segments that mirror the Larson's rim. The rim is
  in the sun, and has to be: the Larson is 12 m across and the widest
  circle the kite holds is 11.78.

  Tipped 55 degrees to put its belly lamp on the mirror it is 0.93 m out
  of the shade; level it would be 0.11 m out. The Larson's lamps want an
  exposure of 15000 to show at all, and why a 1500 W lamp reads so faint
  is not understood. The check calls the smoke not honest, worst 0.19.
  Its stars are under Real stars, below.
- A set file with `setting: space` is drawn by hillside now, and
  `sets/webb-at-l2.yaml` is this smoke as a set; see "A scene drawn a
  frame at a time", above.

### The Larson's lights keep a programme

- A manufactured light may keep a part in a programme that a set of
  lights keeps together: for the first of each cycle every light is
  lit; then they are lit in turn, so many at once, a lap at a time; and
  a light that is not lit is low. A light with no programme is as it
  was, and no geometry reference row moved.
- The MCRN Larson's lights keep one, six seconds long: all lit for a
  second and a half, then the twelve running lights chase round the
  rim three at a time for three laps while the belly lamp is low. The
  cabin lamp is steady. Each light has its name and its place on the
  rim. A still picture, taken at the start, has every light lit, as
  before.
- `improv/larson/landed-night-from-above.yaml` is the night scene with
  the camera above the rim; `slayer render` it with `-at` for a frame
  at a moment. A film of one cycle is 72 frames at twelve a second.
- Not done: no set file addresses a light by name or gives it a
  programme of its own; the programme is the Larson's and is in its
  numbers. The portholes' own glow does not change with it, only the
  light each throws.

### Real stars

- `sky.Catalogue` is a sky of real stars: each a small disc of its own
  light where it is, and the dark between. It keeps its stars in cells
  of equal area and looks only in the cells round a direction; a
  property test holds it to the sum of every star taken by hand, and
  two more to finding stars across the poles and the line its sectors
  wrap at.
- `sky.BrightStars` reads the Yale Bright Star Catalogue's file,
  `ybsc5`, as celestial-data's bsc5 adapter fetches it: each star's
  place at equinox J2000 and its visual magnitude. The columns were
  checked against the file's own Sirius and Vega. The stars are all one
  colour; the catalogue's colours are not read.
- Smoke 50 has them behind Webb when `SLAYER_STARS` names the folder
  the file is in, turned into Webb's own frame: its mirror on Gaia DR3
  4081136684118965760 and its shield's side toward the sun at 20:53 UTC
  on 2027-05-22. Unset, its sky is black as before.
- `oceantex.SunAmongStars` is the first half of `oceantex.Sun`, split
  off: the sun's right ascension, declination and distance at a moment,
  by the Naval Observatory's approximate formulas. `Sun` calls it and
  gives what it gave.
- **The date is held to the geometry.** A test finds the sun 135 degrees
  from the star at that minute, to half a degree, and not a month
  earlier. Until now the date was worked by hand outside the tree and
  nothing checked it.
- Left out and said: the equator's turn between J2000 and 2027, about
  0.4 degree; Webb's place on its orbit about L2; the stars' colours.
  With the shield a kite the Larson no longer fits wholly in its shade,
  and its rim and its reflection in the mirror burn out in sunlight.

### Stacks: capsules, stages and launchers as data

- A craft that is a body of revolution is now data.
  `manufacture/stacks.yaml` lists each as its sections from the bottom
  up: how wide at its bottom and its top, how long, what it is made of,
  how that was finished, and where its numbers were read.
  `manufacture/stack.go` turns any such list into a kind.

  A craft that cannot be built is refused when the file is read, by
  name, and a file that cannot be read stops the build.
- The first: `apollo-csm`, the Apollo command and service module on its
  engine bell, 10.11 m tall and 3.91 m across, from NASA's Apollo 204
  spacecraft page and NSSDCA, with two figures from Wikipedia said to
  be. In the catalogue under Crewed spacecraft, in both set schemas,
  with its eight reference rows.
- Its numbers are read off pages and are not pinned sources: cmd/sources
  does not hold them to their pages' bytes, and the file says so. Its
  looks are picked. It has no windows, antenna or thrusters.
- Three more, as rows: `soyuz-ms`, 7.48 m, without its solar arrays;
  `mercury-capsule`, 3.3 m, without its escape tower; `gemini`, 5.736 m
  on its adapter. Their sizes are from secondary pages, which disagree
  in places; each craft's rows say which page and what it conflicts
  with, and Soyuz's say unverified.
- Two launchers, as rows: `saturn-v`, its three stages and instrument
  unit, 85.6 m, without the Apollo spacecraft; and `sls-block-1`, its
  core stage, adapter and upper stage with a solid rocket booster on
  each side, 86.7 m, without Orion. A stack may now have smaller stacks
  beside it, so many round its foot: a launcher's boosters.

  The stage and booster sizes are NASA's, as a search of nasa.gov
  relayed them on 2026-11-12; the documents themselves were too large to
  open, and the rows say that. The SLS upper stage's size is a table row
  inferred to be it, and its adapter's height is picked.
- `saturn-ib`, a third launcher: the S-IB, the S-IVB and the instrument
  unit, 43.16 m, with no spacecraft on it. Each stage's length and width
  is as Wikipedia's Saturn IB page writes it, read 2026-11-12 and not
  pinned; they come to the page's own 43.2 m. Its infobox gives the
  second stage the first's width; the text's 6.61 m is used.

  The Redstone, the Atlas and the Titan II are not made: the pages read
  give each only as a whole with its capsule on, and no stage's length.
- `saturn-v-apollo` is the whole vehicle as it stood on the pad: the
  Saturn V, the lunar module's adapter, the service and command modules
  and the escape tower, 110.64 m, NASA's 363 feet. The total is NASA's.
  The adapter's 28 feet is unverified and the tower's length is what is
  left of the 363, and the rows say so; a test holds the sections to
  adding up to it.
- A section may be painted: `finish: painted` and `paint:` naming one
  of the swatches. Saturn V is chalk white with a slate instrument
  unit; the SLS core and adapter are rust, for their foam, and its
  boosters and upper stage chalk. The nearest named colours, not the
  craft's own. A paint that is no swatch, and a paint named on a
  section that is not painted, are refused.
- Neither launcher has engines or fins, and Saturn V's black markings
  are not made: they are stripes down its sides, which a body of
  revolution cannot carry yet.
- `apollo-lunar-module` is the Apollo lunar module on its four footpads,
  7.04 m tall and 9.4 m across them. It is no body of revolution and has
  a maker of its own, `manufacture/lunar_module.go`, with its sizes in
  `lunar-module.yaml` and where each was read: NASA's Apollo 204
  spacecraft page, NSSDCA and Wikipedia.

  Its descent stage wears a blanket of Kapton over aluminium, thing's
  own film over thing's own metal, which is why such a blanket is gold.
  It is plain: each stage an eight-sided prism, each leg one strut.
- Owed, and asked for: the rest of the plan's launchers
  (Redstone, Atlas, Titan II, Saturn IB, the R-7 family, Proton, Delta
  IV, Centaur, Atlas V, Titan IIIE and IVB, Ariane 5, the shuttle),
  none of whose sizes has been read; and pinning every stack's numbers
  as sources.

### A glass of cola

- Smoke 52: the tumbler filled with a dark drink, forty-six bubbles
  rising in it, ninety balls of tan foam in a ring at the top, and
  twenty-eight drops thrown over it. The fizz is placed by the golden
  angle and two square roots, with no dice. A test reads the built
  world and holds every bubble inside the drink, the foam at its top
  and inside the glass, and the drops over it.
- **A look may bend through eight clear surfaces, where it was four.**
  A bubble in a drink in a glass is six. The limit is raised in the CPU
  tracer and in the GPU's own copy of it alike; the tracer's and the
  GPU's tests are as they were, the fizzing glass still the one red.
- Not right: every number of the cola is picked; its brown is at its
  faces and not through its depth; the foam is balls, not films;
  nothing moves; and the inside of the glass is dim, a clear thing
  blocking a lamp whole.

### Water with bubbles in it

- Smoke 51: a ball of water floating over the oak table in smoke 49's
  studio, with seven bubbles of air in it. A bubble is a clear thing
  inside a clear thing, its face water against air, and past the
  critical angle that face mirrors: the bubbles have bright rims. It is
  the picture the tracer's nested clear things were for, and the check
  judges it honest.
- When it was made a look through two bubbles, six bends, went dark at
  the tracer's four; with eight it draws the bubble behind. Not right:
  the ball's shadow is solid, a clear thing blocking a lamp whole.

### Webb's shape, by its papers

- Webb is laid out as McElwain et al. 2023 lays it out. The spacecraft
  bus hangs under the sunshield, on the sun's side, with the solar
  array and the antenna; until now it sat on the cold side, between
  the shield and the mirror. The telescope stands on a tower that comes
  up through the shield. The instrument module is on the primary's back.
- The secondary stands 7.169 m in front of the primary's vertex and is
  0.738 m across, the paper's table 2; both were picked before. The
  paper is registered in `sources.yaml` and the two numbers are in the
  research file with the table's own row.
- Six probes in `testdata/spacecraft.yaml` hold it: structure under the
  shield and none between shield and telescope, the array, the module
  behind the mirror, and the secondary where the paper puts it and not
  where it used to stand.
- The sun side is laid out from NASA's picture of it, by eye: two long
  pallets down the shield's middle, a boom out to each side corner, five
  solar panels in a row running aft from the bus, a radiator on each
  side of the bus, the round interface under it, the antenna, and the
  flap at the aft tip. No size of these is from a paper.
- The sunshield is a kite: a long diamond with its tips fore and aft
  and its corners out at the booms, each edge drawn in toward the
  middle as a tensioned membrane's is. It was a hexagon with flat
  sides. Its length and width are still the published ones; the kite
  and the depth of its scallops are by eye from NASA's picture.
- Authored, because neither paper gives them: the size of the bus, the
  tower, the antenna and the instrument module. The module is one
  closed box; the four instruments in it are not made. Not here yet:
  the frill and bib.
- The sun-facing membrane has its seams: a grid of thin strips where
  its gores are joined, across the shield and along it, each as long as
  the kite is wide or long where it lies. That the membrane is seamed
  in a grid is NASA's picture; the spacing, the width and that the
  seams run straight through are by eye from it.
- The shield's layers fan: "closer together at center and further apart
  at the edges" (NASA). Each layer's middle sits lower than its rim, by
  more the further it is from the sun, and the sun-facing layer is
  flat. By how much is authored. A plate may now have a dish, how far
  its middle sits below its rim; nought, as every other plate has, is
  the flat plate it was, and the producer shows no other kind moved.
- Webb's and the Larsons' geometry reference rows were re-captured
  through the producer with sign-off;
  `testdata/reference-recapture-2026-11-12.txt` says what moved and why.

### Webb in what it is made of

- Webb's mirrors are gold and its sunshield is aluminium, each from the
  metal's published optical constants: gold's are Johnson and Christy's
  measurements, aluminium's are Rakić's model fitted to measurements.
  Until now both were picked colours: a flat gold and a flat grey that
  mirrored nothing.

  `manufacture/webb-surfaces.yaml` names each surface's material and
  finish and says where that was read; a part Webb has not, or a
  material or finish slayer does not know, is refused by name.
- The catalogue's plan gains `studio: true`. Webb's entry is staged in
  space, where a mirror shows the dark; a plan that asks for its studio
  lights a staged entry in the plan's own room, so what it is made of
  can be seen.
- Not right yet. The sun's side of the two hottest shield layers is
  doped silicon over the aluminium, and pink; here it is aluminium. The
  mirror's SiOx overcoat is not drawn. Webb's structure keeps its picked
  dark grey: black Kapton has no optical constants here.

  A mirror seen in a mirror is black, because the tracer follows one
  reflection: the shield's lower layers and a patch of the primary show
  it. And the model's shape is still authored: the bus is on the cold
  side, and there is no tower, instrument module, solar array or
  antenna.

### A studio: a cove, a wooden table and a glass

- `texture.Wood` is sawn timber: a tree's rings, counted out from its
  pith, cut through by whatever faces a board has. A board shows arches
  on its face and rings on its end, and they meet at every edge.
  `swatches.Oak` and `swatches.OakLate` are its two woods.
- Smoke 49, a glass of water on a wooden table in an infinity cove: a
  floor that turns up into a wall with no line, an oak table, and a
  tumbler of clear glass, turned as a potter turns one, most of the way
  full of water.
- **One clear thing may stand inside another.** The tracer knew only the
  air against a thing. A look now carries the clear media it is inside,
  and a surface's two sides are worked out from them: water against
  glass bends by 1.33 over 1.5.

  `thing.Material` gains `Rank`: where two clear things overlap, the
  point is made of the one of the higher rank, so water that runs a
  little way into a glass's wall meets the glass with no gap of air and
  no two faces in one place. A surface that is then no boundary is
  passed straight through and does not cost a bend.

  One clear thing in the air is exactly as it was, and the tracer's own
  tests hold that.
- **The Metal kernel knows it too.** For a morning it did not: the
  tracer's change moved a picture the two are held to agree on,
  `TestTheGPUDrawsAFizzingGlassAsSlayerDoes`, bubbles in a ball of
  water, and the GPU was 0.0028 off the CPU on average and 0.87 at most.

  The kernel now carries what a look is inside, as the tracer does: a
  material's rank goes to it in a spare whole number of the record it
  already had, and the bend loop of `shade` asks what each clear surface
  is to the look, bends at a boundary and goes straight through a
  surface that is none, on its own count.

  That test is 0.00001 off on average. A glass of water with and without
  a rank, and a glass bead under water, are each drawn both ways and
  agree to about a part in ten million. Every test the GPU had passes as
  it did: a clear thing in the air bends and splits by the numbers it
  always did.
- When the water was poured its top surface was a black ellipse: a look
  down onto it wants a fifth bend and the tracer followed four. With
  eight the top is water. Not right in the smoke yet: the glass's shadow
  is too solid, a clear thing blocking a lamp whole.

  And the check judges the smoke not honest, worst 0.023: the grain is
  finer than the terminal's picture, and `Wood` does not yet blur itself
  for a big pixel as other textures here do. Water's index is 1.33 as
  the tracer's tests have it, not yet read from the database.

### The MCRN Larson, landed

- A second kind, `mcrn-larson-landed`: the same saucer with its gear
  down and its stairs out. Three legs, each a strut, a brace and a round
  pad; and one flight of nine steps from the belly disc to the ground,
  built by the stairs constructor from `saucer-landed.yaml`. The ground
  is not booked: it lies where the stairs' foot stands, so the top tread
  meets the belly disc.
- Every family of the gear names its material and its finish, and its
  look is made from the two by `thing.Made`: titanium, weathered, for
  the legs, stringers, rail and posts; aluminium, brushed, for the
  treads.
- The saucer in flight, `mcrn-larson`, keeps its shape: a ship in flight
  has its gear up. Its geometry does not move, but its portholes glow
  now, so its look has.
- The Larson has fourteen lights, each by a name a scene can address:
  `cabin`, the red lamp under the dome; `belly`, a floodlight down from
  the belly disc; and `port-0` to `port-11`, the portholes, which are
  its running lights and whose glass glows. Before this only the cabin
  lamp existed, and no picture had ever shown it.
- Landed, the saucer's origin is the ground under its feet. A scene
  stands a thing's origin on the ground, and the origin was the belly,
  so the legs and stairs were buried and the hull sat on the ground.
- Both Larsons are kinds a set file may place. Neither was on the set
  format's list, so no set file could place one.
- `improv/larson/landed-night.yaml`: the Larson landed at night, lit by
  nothing but its own lights.
- Owed at the merge, and to be agreed first since it re-captures a
  compatibility record: the Larson's rows in `testdata/saucer-reference-
  arm64.txt`, through the reference's producer. The landed kind has none
  yet.

  The flying kind's eight no longer match, because its portholes now
  glow and the record hashes a look's glow: an earlier commit here said
  its records would not move, and that was wrong. Webb's eight rows in
  `testdata/spacecraft-reference-arm64.txt` no longer match either,
  because its looks are now gold and aluminium.

  Until the rows are re-captured
  `TestAuthoredCatalogueMatchesItsGeometryRecords` fails on those
  sixteen, and on nothing else.
- The Larson's own tests were run once when its lights went in, and
  passed. One of them tests less than its name says, found by a reviewer
  on 2026-11-12 and not yet mended: `TestTheTopTreadMeetsTheBellyDisc`
  checks the ground's formula against itself and not where the built
  tread lies.

  The same reviewer found smoke 48's check that lead is black passed for
  anything under two parts in a thousand; it now holds lead to a
  billionth.

### What a material takes from an x-ray beam

- A material can carry its mass attenuation coefficients, mu over rho,
  each at the photon energy its table gives and with the row it was
  read from: `Attenuation`, `Attenuations`, and a material's
  `Attenuating`. A coefficient is answered at an energy a table gives,
  `At`, or for every row in one of celestial's ranges, `Within`;
  nothing is interpolated.
- A material's density can carry its source: `AtDensity` and
  `DensityFrom`.
- `Passes` is the share of a beam that gets through a thickness: e to
  the minus mu over rho, times the density, times the thickness. It is
  derived each time and never stored, and it refuses a material with no
  coefficient at that energy or no density cited.
- Lead and water are materials now, and aluminium has its density: each
  with its coefficient at 60 keV (`FilmEnergy`) from its page of NIST's
  X-Ray Mass Attenuation Coefficients (Hubbell and Seltzer, doi
  10.18434/T4D01F) and its density from that database's tables of
  material constants.

  Only the rows used are quoted; the pages are held by celestial-data's
  nistxray adapter. Through a centimetre at 60 keV, lead passes about 2
  parts in 10^25, aluminium 47% and water 81%.
- Lead's atomic weight, 207.2, is its number over the ratio of number
  to mass in NIST's Table 1.
- `thing.Stopping` is the bridge to the tracer: a material's
  coefficients and density become a `Radiant` that passes, of a band
  whose photon a coefficient was read at, the share one unit of the
  thing passes. A band no row was read at is stopped whole.
- Smoke 48, lead, aluminium and water on an x-ray film: three balls two
  centimetres across in front of an x-ray source, seen at 60 keV. No
  share in it is picked.

### The smoke check draws a smoke's bands

- The check drew only what the eye sees, at a fixed exposure, and never
  read a smoke's `Bands`, `Palette` or `Tone`. A smoke lit only in a
  band, as smoke 48 is, was a black picture both ways, which the check
  compared with itself and called honest.

  It now draws each band and lays it over the visible in false colour,
  in the palette's colour where there is one, through the smoke's tone,
  as the `Smoke` type always said. A smoke with no bands is developed
  exactly as before, pixel for pixel.
- Judged at full size with their bands drawn: smokes 20, 25, 26, 28 and
  48 are honest. **Smokes 21, 31 and 33 are not**: worst 0.031, 0.072
  and 0.418. They were called honest before only because their bands
  were never looked at. Why is not known yet. Smoke 29, which is heavy,
  was not judged.
- `improv/xray`, the stand-in that drew smoke 48's film while the check
  could not, is removed.
- Its three judge tests, the light group's test and the command's
  default test were run once when it went in, and passed.

### The yard built from materials and finishes

- Added `flat`, an eighth finish: gloss nought and satin nought, no
  colour of its own — the material's colour and no sheen at all.
- `painted` is now a pigment coat rather than an eggshell one: gloss
  nought and satin nought, so paint colours and covers whatever it
  meets and does not polish the surface. The colour still travels with
  the paint, in `Painted`.
- The yard's six looks are now made rather than picked: each is
  `Made(materials.ByName(...), thing.Painted(colour))` — steel, ochre,
  white, timber, tyre and window, each a material record with its
  authored formula and nothing cited beyond it, painted in today's
  authored colour.

  The geometry digests did not move: each look keeps its name ("yard
  steel" and so on) and its eight numbers, colour for colour, gloss and
  index still nought.
- Stairs are built by a constructor now — a flight booked in yaml by
  its steps, rise, going and width, its treads, stringers, posts, rail
  and infill of bars or a panel each naming a material and a finish,
  laid out through the same box and rod calls a hand-placed design
  uses — and `access-stairs` is rebuilt by it: the same eighteen parts
  in the same order, and no geometry digest moved.

### The spacecraft metals' optics

- The five spacecraft metals — gold, silver, aluminium, beryllium and
  titanium — now come from the materials package, read out of the
  database rows it commits, and no look moved: every n and k the
  constructors return is what it was, and the geometry records did not
  move.
- Aluminium's blue-band n is corrected to the published row, by
  decision: 0.64439, the row `4.5833e-01 6.4439e-01 5.3994e+00` of
  main/Al/nk/Rakic-BB.yml in the refractiveindex.info database release
  v2026-05-24, in place of the 0.6444 the metals had carried. Every
  other metal value, every band, is unchanged.

### The candidate of 10 november: the music, the trees and a more muscular GPU

This part was written for a v0.1.2 of 10 November that was never
tagged; it ships in this v0.1.2.

Reviewed so far: muscle-weather on 2026-11-09, and the release candidate
at 9d7ba53 by three reviews on 2026-11-10. Nothing else here had a review
on record before those, and "landed" below means merged, not accepted.

### Lights played to music (moonlake)

- Added `cueshow`, the cue compiler: a score (the track's measured beat
  grid and transcribed notes) and a cue sheet written in bars compile
  into a moment's light state as key=value lines. `lightshow` run with
  `CUES` obeys them; without it, it free-runs as before. Editing the
  cue sheet is how the show changes; nothing recompiles. Ten tests,
  one end to end.
- Added `metronome`, the score checked by eye and ear: raw frames to
  ffmpeg with the song underneath, a flash on each measured beat and a
  band for each cue as it fires.
- Added the moonlake improv tools, each its own small program:
  `seawater`, `seamatte`, `mistlake`, `mirrorlake`, `lightshow`,
  `passes`, `pfmstat` and `scalepfm`; `sets/moonlake.yaml` is the set.
  OPERATION.md gains the music heading; `improv/moonlake/README.md`
  is the worked example, with every flag and variable the tools read.
- Added hillside improv flags (`-lake`, `-lakeglow`, `-moon`,
  `-moonglow`, `-giants`, `-close`, `-closeglow`). These are scene
  content; the set says them under a `fantasy` block (see "One render
  library", below), and the flags still work.
- `make_score.py`, the score's generator, remains outside the
  repository; the README records where it is and how the score was
  pulled from the song. Bringing or porting it is still owed.
- moonlake stages with no flag: its default of 3,117,848 kept faces is
  inside the memory budget that `-faces` defaults to. Its improv flags
  such as `-giants` and `-close` can still need `-faces` raised.

### The trees

- Added `manufacture.Taller`: a whorled tree grown taller rather than
  magnified - the trunk taller and thicker with more whorls, branches
  as long as the tree is tall, every branch and needle its own size.
- Added `manufacture.Branched`: a whorled tree whose branches carry
  side branches of their own; a grown tree's limbs flow thicker.
  Today's trees are unchanged, face for face.

### The GPU, more muscular

Stage 0 of doc/gpu-muscle-design.md, stage 1, stage 4 and part of
stage 2 landed; the design doc records what each stage is and what
remains open.

- Stage 0, nothing paid twice while its inputs are unchanged: one
  compile per distinct kernel source through a shared cache; Grid on
  the streamed band path, one copy per input, drawing a band ahead of
  the one being read; a grid stream kept between pictures updating
  what changed; crossing arrays sized to the scene's own volume count
  under a 256 ceiling, replacing the fixed 32-volume budget.
- Stage 1, refraction: a clear surface bends the ray by Snell's law in
  trace and the kernel alike, held to what the kernel can unroll.
  Mirrors do not refract, bent legs do not mirror again, and past the
  critical angle the bend is the reflection.
- Stage 2, in part: `metal.WaterLibrary` (Quan-Fry, Fresnel, Cox-Munk,
  Smith, every sourced number passed in so sources.yaml stays the one
  ledger); `shape.HeightField`; and the kernel marching height fields,
  water and ground alike, against the tracer at a stated tolerance.
  The cascade packing and resolved-variance shading remain open.
- Stage 3 groundwork: titan's made ground hangs on grissom's
  whole-number corner names instead of float64 bit patterns, so a GPU
  refining the same ground derives the same moves the CPU does. This
  changed what ground a seed makes. The celestial pin is read by an
  absolute path in `go.mod`, so the build needs a machine that has the
  directory named in `go.mod` until the pin is replaced through the proxy.
- Stage 4: `medium.Fizz`, the haze of microbubbles over something
  dissolving, scattering light the way a whitecap's foam does.

### Weather, lighting and performance (a checkpoint)

- Bounded current-frame incident-sky lighting prepared on the GPU and
  shared among media; normalized sky illumination for painted media on
  CPU and Metal; projected specimen extent choosing island foliage
  detail. A matched populated 720p scene traces in about one second
  against six seconds of direct tracing; whole-frame gains are smaller
  and recorded, not generalized.

  The figures are as recorded on 2026-11-10 in
  doc/volume-environment-lighting.md; no command in the repository
  reproduces them and they were not rerun.
- The same 4K scene still reaches its 150,000-instance allowance
  before GPU rendering begins. Vegetation preparation, grouping and
  residency remain active work toward the 4K film.

### Doors, names and typed quantities

- `scene/bodies` is behind constructors: `NewBody` checks a body's
  core and hangs its parts, and every part - ground, gas, air, clouds,
  cities, aurora, rings, disc, jets, flare, coma, tail, plume, horizon,
  moon - is made by its own constructor. Every reader documents its
  unit.

  Of the eighteen constructors in `parts.go`, fifteen make parts;
  `NewShell`, `NewAuroraColours` and `NewFineness` make values that
  parts hold. Each refuses an argument that is not a finite number.
  `NewAuroraColours` only holds three colours, and `NewAurora` checks
  them.

  `NewBody` and `With` refuse a nil part and a part that did not come
  from its constructor, so `bodies.Gas{}` cannot be hung on a body.
  `With` returns an error, `Aim` is the value verb `Aimed`, `Tilting`
  sets a tilt without the day, and a copy of a body owns its moons,
  tails and plumes. `Binary` makes the checks `Place` makes.
- Shots adjust by verbs (`With`, `WithoutMoons`, `Resized` and kin)
  instead of reaching into fields. The templates propagate errors;
  `template.Named` and `stage/build.go` stop swallowing them.
- Constructor parameters are typed quantities, so a call site reads as
  a sentence and a length cannot stand where an angle or a share goes,
  inside the part constructors: measure's lengths, angles and kelvin;
  `maths.Portion`, where `Percent(50)` is half and an amount that reads
  as a fraction is refused with the fix in the complaint; `Loss` for
  extinctions per kilometre.

  `Omega`, `Binary`'s distance and the template helpers take typed
  quantities too. `Body.Spinning` takes a plain number, because a
  spin is signed, and the function `Stand` takes world units.

  Two arguments of one kind can still be swapped: a disc's inner and
  outer edges are both lengths.
- `bodies.Span` holds kilometres exactly as written: a trip through
  measure's natural units costs a last digit, which the worked halves
  and quarters cannot pay.
- `maths/hexes` speaks its own vocabulary - along and across, column
  and row, side and steps - and its value types carry the value vow
  maths already uses, rather than doors that would re-say their
  documented zero readings.

### The catalogue

- Added the stage lights to the manufactory's book: seven lasers and a
  strobe, with their menu rows and sources, so the catalogue again
  holds every kind the manufactory makes - the two-way check is a
  test, and `game run catalogue-render` draws all 87.
- Added `catalogue pdf`: a finished book composed into one PDF to
  send, a cover and four specimens a page, refused while the book
  holds pending or failed cards. `game run catalogue-pdf` is the short
  form; doc/browsable-catalogue.md says the rest.

### One render library, gas in the kernel, Melancholia (10 and 11 november)

Merged on `night/integration` and `11-11-melancholia`; reviewed by the
session that wrote it and by agents it briefed, not yet by anyone else.

- `hillside` is a library: `hillside.Run` stages and draws a set file.
  `cmd/hillside`, `slayer render` and `azazel/engine` are thin callers of
  it. A set's `world` half and its terrain half stage into one world.
- Added the set's `fantasy` block: `lake`, `lakeglow`, `moon`, `moonglow`,
  `giants`, `close` and `closeglow`, the hillside scene flags of the same
  names, each with its flag's type. The block states what a scene does
  that the physics does not say, and the render prints one line for each
  key. A flag given on the command line stands over its key, and the line
  says so. The flags still work; celestial is unchanged.
- `render/trace` is deleted; `trace` is the only CPU tracer. Scenes
  written for the old tracer have their lamps scaled by pi. Six smokes
  with one sun on both a matte surface and a volume (10, 11, 16, 19, 24,
  38) do not match their old pictures; the brief lists them for review.
- The Metal kernel draws gas: `gas.Optical` volumes, ring slabs asked in
  closed form, and orbital's wake. Orbital and Melancholia draw on Metal.
  The tolerances against the CPU are stated beside each test.
- The face budget is in bytes: 5 GB at a measured 1,300 bytes a kept
  face. Melancholia stages with no flag.
- `planet.Face.Within` is 7 to 8 times cheaper with every bit kept, and
  makes nothing on the heap for each point.
- The tree cache (`SLAYER_TREE_CACHE`) and striped lens rays are in.
- Melancholia's fifteen fires are recipes; the legacy `alight` form is
  gone from the set. A kind is refused in a sequence when it gives lamps
  sampled at construction, not for its name. The recipe lamps number 402
  against 422; the bonfire's brightness is within its flame seed's spread.
- Added `planet.Chart`: a face painted once onto a grid, a tile at a time
  as rays land, and read between its points. On Metal a planet with a
  flow is drawn from a chart with two points to a pixel unless the set's
  planet says `chart:`. `chart: 0` works the face out for every ray, and
  the CPU always does.
- The chart changes Melancholia's Metal picture. At 1024 wide and 16
  rays it is 0.45% of mean light from the uncharted one, and the render
  took 4m03s against 5m51s on a loaded machine.
- Added `planet.Face.Laid` and `planet.GridFlow`: a face with a grid's
  weather laid out as tables a renderer can paint from. Nothing reads
  them yet but their test.
- A sequence draws in sections. A frame wider than 320 pixels is split
  into sections 64 pixels square; the next frame keeps each section that
  cannot have changed and draws the rest (`trace.SampledEach`).
- A section is kept when nothing in the world changed, or when all its
  rays can meet is the same sky and bodies that show only their own
  light, have not moved and are not painted by a texture that moves. A
  moved camera, a lens rig or a world whose rays bend, and every section
  is drawn again.
- On the CPU the frame made this way is the frame drawn whole in every
  bit, by test. On Metal a frame is kept whole or drawn whole: Metal
  draws no part of a frame yet.
- A set's film says `sections: {over, size, keep}`; `keep: false` draws
  every section of every frame, to check the kept ones against. Both
  schemas and `doc/sets.md` have it.
- Added the `section` package: it splits a frame, says from a body's
  bounds which sections the body can show in with no ray traced, and
  says which sections a change between two frames keeps.
- A film's `sections` may say `drift`. With a drift, a section with a
  steady volume in sight is kept when the lamps that changed bring the
  volume no more than that share of what the steady lamps bring, and
  nothing that changed could shadow it. Tested against whole frames: no
  kept section is further off than the drift times its brightest pixel.
- Things and volumes are matched between frames by identity, so sparks
  added each frame do not stop the rest being kept.
- A tester's tests of sections found four defects, now fixed. A
  section's sight is found from every pixel's ray, not nine: it missed
  bodies through a wide aperture and near a half turn. A changed lamp's
  light on a volume is bounded by what the lamp says it sends at most
  (`Stands`, `AtMost` on the point, ball and spot lamps), not by asking
  it at ten points.
- A changed lamp that does not say where it stands, or that stands in
  the volume's box, keeps nothing under any drift. Worlds that differ in
  their pixel's width or how volumes are lit keep nothing.
- On a set with terrain no section is kept yet: the terrain is one lit
  body whose bounds cover the whole frame. The bounds test is too coarse
  for it; a first-hit pass is needed.
- Not done for sections: a single frame is not drawn in sections; no
  section is kept on disk between runs; a lit surface in a section's
  sight (the ground by a fire) always redraws it; Metal draws no part of
  a frame.
- A hole built with a retrograde spin is drawn: celestial's hole of that
  spin's size, turned over, as celestial's own comment says to. A field
  celestial refuses is now refused by `trace` by name; it used to vanish.
  The Metal path has not been read for the same fault.
- A world-only set that gives no size draws at 320 by 200 with 2 rays
  (it was 1280 by 800).
- `slayer render` takes `--width`, `--height` and `--rays`, in
  puzzlemouth's form, and refuses `-w`, `-h` and `-n`. `cmd/hillside`
  keeps its short flags.
- A set's terrain and its moon face are read from the shelf and never
  downloaded unasked. A staging that finds one absent fails with one
  error naming the dataset, the path looked at and `--fetch-terrain`.
  Every shelf read in `hillside` goes through `pinnedAssetAt`.
- `slayer render` and `hillside` take `--fetch-terrain` (fetch what the
  shelf lacks, check it against `sources.yaml`, then draw),
  `--refresh-terrain` (fetch again, replace a file only with a good one,
  say what changed) and `--check-terrain` (download nothing, compare the
  shelf with `sources.yaml` and the server's size, draw nothing).
- The Kilauea dem's one member, `kil_2019dem/kil_2019dem.tif`, is pinned
  in `sources.yaml` under `unpack`, and unpacked to `unpacked/` by
  `sources fetch` and `--fetch-terrain`. Staging holds it to that pin.
- celestial-data has no terrain or Moon adapter yet, so these datasets
  still come from `sources.yaml`. `sources.Origin` is the seam an adapter
  fits. 
- Smokes are judged by group and can be judged small: `slayer smokes
  check` and `slayer smokes checksmall`, each taking group names or
  smoke numbers, with `smokecheck` taking the same. The groups are light,
  worlds, lens, energy, gas, holes and earth. The judging is in
  `render/smokejudge`; the old `smoke` verb is gone.
- `-k` defaults to each smoke's own fineness. Small draws the terminal at
  180 by 75 with at most 3 rays a side, and skips smoke 6 by name. The
  test's guard on how much was compared is a quarter at small size; the
  test says why.
- The tests judge the four heavy smokes (29, 35, 36, 37) small. At full
  size they pushed the package past its ten minutes in the full sweep,
  so full size is judged when `SLAYER_SMOKES_HEAVY` is set or by
  `slayer smokes check 29 35 36 37`. OPERATION.md says to do so before a
  release.
- hillside refuses an argument that is no flag, by name. A set's path
  given without `-set` used to be ignored, with every flag after it, and
  the default hillside drawn.
- Smokes 40 to 47 draw capabilities nothing drew: four atmospheres (40),
  a laser in haze (41), a hill beside a closed solid (42), a glass fizzing
  (43), the earth by night from a picture of its lights (44), three planets
  on a track of the caller's (45), a baked puff beside the exact one (46)
  and a stage lit by ninjafilm's lamps (47). None is heavy.
- Melancholia's fire is from the fire support cookbook. The bonfire is
  the night wood fire of `doc/fire-night.yaml` sized to the pyre: 24
  flames with hot tongues at 1900 K, sodium's lines, sparks and smoke.
  The burning trees are the tree fire. The fire's heart shows white at
  the giant's exposure. The first recipes, made on 11 november from the
  old kind's numbers, drew one soft orange flame and are gone.
- `sets/melancholia.yaml` says `renderer: gpu-native`. As written it
  draws at 2048 by 1501 with 16 rays in 81 s (build 13 s, render 68 s),
  7.4 GB at its peak, on a machine at load 2.6; on 10 november it took
  about 112 s on the kernel's own traversal.
- `sets/kilauea-iki-forest.yaml` and `sets/moonlake.yaml` say
  `renderer: gpu-native`, each within 0.08% of mean light of `-gpu`. Not
  so for `kilauea-iki` (0.3% off, brighter in its fire),
  `kilauea-iki-moonrise` (0.2% off) and `planetary-turbidity` (no
  meshes). A test draws each set that says it, small.
- `cmd/lightbox` is removed.
- `hexes.Tile.I` and `J` are `Column` and `Row`; `hexes.Sphere.N` is
  `Splits`.
- Fixed a hang in `renderer: gpu-native`: one shadow ray from Melancholia's
  far planet, 2.4e8 m off, never came back from the hardware query. The
  kernel now clips each native query to the sphere round the standings,
  a header field. At 1024 and 2048 wide with 4 rays a side it draws in
  44 s and 82 s, in 5 to 7 GB; the 1024 picture is within 0.01% of mean
  light of the plain `-gpu` one.
- A wait on the GPU gives up after 60 s (`metal.Kernel.SetLimit`; the
  longest command measured was 1.2 s) with an error naming the kernel and
  rows. The device may stay busy, so that kernel is not used again. A
  band drawn ahead that fails is now an error. An empty input is bound as
  zeros as long as the kernel declares.
- Known and open: the full suite needs more than the default ten minutes
  in `cmd/smokecheck`; ring brightness depends on `-n` on both renderers;
  a set path given without `-set` is ignored.
- The set reader takes schema version 2: one reader, which checks a file
  against the schema it names. Version 2 now keeps every kind version 1
  has, and a test compares the two schemas' word lists as well as keys.
- Read from version 2: `setting`, `date`, `frame`, `anchor`, `globe`,
  `sky`, `film.fps`; the kinds `point`, `star`, `lantern`,
  `culture-orbital` and `nebula` with their keys; a thing's `place`,
  `around`, `orbit`, `attitude` and `eye`; a camera's `moves`, `duration`,
  instrument `eye` and `location.relative`; an atmosphere over the world.
- The four sets `corona`, `orbital-and-lantern`, `webb-looking-out` and
  `keyhole-to-ground` read. `hillside` refuses each by name before it
  stages, with the file, the key, its place and that it is read and not
  yet drawn. Still to be drawn: all of the above; a space scene, camera
  moves and the star's disc, limb and corona come first.

## v0.1.1 09 november

- Separated grounded specimen screen extent from projected displacement error
  when choosing island foliage detail. Visibility keeps conservative source
  bounds; thresholds and instance limits remain unchanged. Independent pinhole
  fixtures check the new extent query. Larger-scene and temporal acceptance
  remain separate from those geometry controls.

- Added optional bounded native preparation of incident sky light. Current-frame
  grids share the source among media, interpolate inside their fixed domain and
  blend to direct integration at its boundary. Density, opacity and explicit
  lamps remain direct. Native controls cover source updates, fallback, float
  coordinate refusals and a small interpolation study. Full-scene accuracy and
  preparation-plus-trace performance remain under review.

- Added opt-in normalized isotropic sky scattering for painted media on CPU
  and Metal, including geometry shadows and medium transmission. Island YAML
  selects the model and bounded angular sample count. The lamp normalization
  changes only in the selected mode; emission and legacy inputs retain their
  earlier response. Independent small native controls pass. Full-scene quality
  and the cost of direct per-step sky queries remain under investigation.

This is the consolidated change inventory since `2a4d86d`, on the working
branch `dist/2026-11-09`. It includes work also summarized
in the dated release notes below. Drafts and missing integration are called
out where they belong; this inventory is not a fresh test or release verdict.

### Camera travel, sequences and demos

- Added optional bounded terrain transitions for the island sequence. A common
  refinement blends source-grounded endpoint meshes over scene time; unchanged
  selections reuse geometry without repeating normal samples. Native moving
  and held-view controls support the local improvement; general temporal,
  normal/shadow-lift and high-resolution acceptance remain open.
- Added optional continuous coordinate interpolation to island tracks, with
  shared waypoint velocities and a fixed-target 24 fps example. Legacy segment
  easing remains the default. A native 120-frame comparison crosses the old
  stop waypoint; terrain LOD flicker remains a separate unresolved defect.
- Added an island-sequence draft with geographic camera/target keyframes,
  angular zoom and separate scene/playback clocks. Source rasters, native
  height indices and bounded prototype geometry remain open across frames;
  view meshes and GPU resources are rebuilt. Each frame records its exact
  effective scene, hashes and costs. This is authored geographic travel,
  not an orbital solution or certified obstacle-avoidance path.
- Verified a first native 60-frame, 320 by 180 ascent through a one-hour
  scene clock to 300 km viewing height, encoded as ten seconds at six fps.
  Frame/source hashes and repeated tree identities were checked. Visible
  terrain faceting, hard authored colour bands, wave water and instrument
  lookdown remain open; this is a choreography study, not the final shot.
- Added the `camera` package: checked constant-speed travel, timed moves,
  linear or smooth easing, holds, continuous joins and frame-time sampling.
  Paths refuse invalid coordinates, timing and crossings of the look target.
- Added a look target per move through `Move.Looking`, including moving
  targets and pans during holds. Position and look joins are both checked;
  these targets are scene coordinates, not object-relative flight planning.
- Added optional angular zoom, independent of travel, with linear or smooth
  easing and exact held endpoints. Existing plans keep their fixed lens.
- Added hyperlapse `scene-duration`, separate from playback `duration`.
  Camera speeds, fire and prescribed weather use scene time; frame records
  retain their actual scene timestamps. This does not animate fixed Sun or
  Moon directions from an ephemeris. See [camera motion](doc/camera-motion.md).
- Added Hillside's Pzm `sequence` command and bundled JSON Schema. It retains
  scene construction between frames, accepts terrain scenes or numbered
  smokes, and supports camera plans, dry runs and resumable numbered PNGs.
- Sequence manifests identify scene, plan, executable and referenced input
  bytes, including terrain and smoke assets. Resume verifies real files,
  frame hashes and timestamps; changed or missing inputs cannot silently
  reuse an earlier frame. Failed preparation does not publish completion.
- Added fast lossless `SequencePNG` and `SequencePNGHash` output. Frames are
  encoded before atomic replacement; the digest describes the bytes written.
  This trades larger files for cheaper compression without changing pixels.
- Added `game run demos` and `game run demo N`, with stable smoke selectors
  and orbital, swarm, ring, hillside, moonrise, house and camera previews.
  Defaults and sample budgets live in a checked menu. Optional terminal
  output keeps the PNG and uses the existing sextant renderer.
- Demo commands build their actual scene executable and report outputs under
  `out/demos`. Missing data is a refusal, not a successful blank picture.
  Argument forwarding needs a suitable Game build; private review dependency
  pins still limit the fresh-checkout experience. A broader preview/data/cost
  guide remains open. See [demos](doc/demos.md).

### The manufactory and its specimen book

- Added `hull`, `clumps`, `quick` and `demo` complexity controls, named YAML
  construction recipes and shared geometry. Detail changes representation;
  instrument apertures and declared physical envelopes stay fixed.
- Added a native-Metal specimen catalogue with local search, family/detail
  filters, comparison views, observatory cutaways and source notes. Each
  view records its binary identity, inputs, bounds, stored faces, GPU timings
  and native query counts. Pending and failed views stay visible in a partial
  book. See [the specimen book](doc/browsable-catalogue.md).
- Catalogue runs refuse unavailable GPUs, unsupported details, excessive
  sample/geometry budgets and existing output destinations. A native image
  records that view's execution; it does not certify every kind or establish
  CPU agreement. The book covers named kinds, not all scene recipes.
- Corrected loose-shoot catalogue orientation and improved specimen sampling
  and the uniform backdrop. Added viewing headings/elevations for useful
  instrument views without changing the manufactured object.
- Embedded the supplied light and dark Miami artwork. Grissom selects its
  native application icon by system appearance; the catalogue includes both
  favicons. Miami's scenery, typesetting and HUD remain design references,
  with their unimplemented work tracked separately.
- Added opaque `manufacture.Kind` and `AttachedLight` aliases, owned local
  light descriptions and `PlaceLights`. Placement caches can retain lights
  without keeping a specimen's construction geometry alive.
- **Go API change:** specimen `Lights(placement)` now returns a light list
  and an error. Callers must handle invalid placements. `Things` and assembly
  paths also propagate placement/construction failures without partial goods.
- Hillside's old count flags retain their original kinds. New kinds enter
  through recipes and YAML; this does not expand the legacy flag catalogue.

### Trees, brush and ecosystems

- Added optional spherical cloud layers with direct CPU/Metal density
  evaluation and separate camera/shadow intervals for both shell legs. They
  continue offshore without the local mist bank's oval boundary. Authored
  weather is not a forecast or a planetary multiple-scattering atmosphere.
- Regional decks can use broader footprints and separate weather patch size
  from vertical depth. Existing omitted-field appearance remains available;
  the broad planar studies exposed an oval boundary and remain diagnostic.
- Added an optional bounded cache for immutable mist floor grids, keyed by
  source projection parameters and the fixed world pose. Weather density still
  advances and rebuilds per frame. Records expose builds, reuse, evictions,
  payload bytes and source queries; zero keeps uncached construction.
- Added optional per-material overhead-area fitting for distant island crowns,
  using the original clump geometry and an explicit sampling grid. Records
  retain the source/before/after areas, scales and unresolved cases; matching
  resolution enters the prototype cache key. This display approximation keeps
  height and visibility bounds and does not certify directional transport.
- Added an optional distant crown representation derived from each seeded clump
  specimen's material and occupied height/quadrant bounds. The woodland example
  keeps geometry farther away. Optional overhead-area quadrature and a
  documented overlap approximation replace the assumption of full canopy cover;
  both the sample work and estimated coverage enter render records. Candidate
  bounds reuse region height envelopes before individual height reads.

  Moving appearance and directional transport remain under review.
- Added optional hierarchical region assessment before root enumeration in
  island forests. Distant supports use authored canopy appearance; measured
  prototype envelopes, source-height bounds and shared work limits govern
  nearer cells. Added stable density/colour thinning at region edges and a
  wider woodland example. Moving-view and resource inspection remain open.
- Added an island-site draft with source-grounded equipment and seeded trees,
  projected-size detail selection, authored aggregate landcover and regional
  prescribed mist. A bounded native height index tightens refinement priority.
  Camera-relative placement shares the island frame; terrain datum conversion,
  temporal residency and inspected moving transitions remain open. See
  [the sourced site study](doc/island-scene.md).
- Indexed selected terrain edges by lattice line instead of scanning every
  leaf at every vertex. Explicit island command ceilings allow denser studies;
  their cell-span targets still report when quality is limited by the budget.
- Added optional authored elliptical mist-bank edge fades after an overview
  exposed rectangular clipping. Zero fade retains the original deck support.
- Added an optional checked secondary-ray reference length. Its zero value
  preserves the geometry-derived default; explicit planetary scenes can pick
  a local base without changing the shared CPU/GPU offset multiplier or the
  point-magnitude contribution. This is a heuristic, not an error certificate.
- Added coast redwoods, giant sequoias and their separate shoot specimens.
  Recursive foliage uses physical needles or scales and bounded shared shoot
  prototypes on CPU and GPU. Construction notes retain botanical dimensions,
  algorithm references and authored choices. See [conifer foliage](doc/conifer-foliage.md).
- Drafted physical recursive shoots for the named spruce and Norway spruce
  entries while retaining their legacy scaffold/reference constructors.
  Foliage carriers are identified by role, including spruce without hanging
  branchlets; physical leaf size is independent of tree scale.
- Drafted `valley-oak`, `alder`, `poplar`, `manzanita`, `chaparral-brush`
  and `joshua-tree`, with distinct forking/leader/multistem/rosette habits
  and closed leaf meshes. Saguaro has separate pleated columns, curved arms,
  rounded caps and shared spine geometry. These are authored family recipes.
- Added named summer, autumn and winter recipes through `WithSeason` and
  YAML. Deciduous winter removes blades while retaining wood and the leaf-on
  reserved envelope; evergreen foliage remains. These are discrete authored
  states, not simulated phenology. See [vegetation families](doc/vegetation-families.md).
- Added bounded spatial populations in `scene/biome`, separate from the
  older top-level `biome`. Region/seed/cell/ordinal identities and independent
  draws keep candidate positions, recipe, geometry seed and heading stable
  across overlapping queries. Query boundaries use half-open ownership.
- Added local `scene.ecosystems` mixtures, density and elevation filters,
  exclusions, prototype variety, detail and optional shared atmosphere.
  Regions share one preparation work allowance; exhaustion returns an error
  rather than dropping later regions. See [regional ecosystems](doc/regional-ecosystems.md).
- Hillside mixed populations still use the cropped planar terrain stage. Final ground
  placement and filtering can change with the crop. Whole-world residency,
  aggregate distant canopy, projected-size vegetation detail and inspected
  mixed-biome scenes remain open; the new plant drafts are not all GPU-accepted.

### Observatory instruments and working sites

- Added a configurable optical observatory: slit dome, mount, truss and
  segmented or monolithic primary. Added Palomar, Keck I/II and Subaru
  scenery families with separate source dimensions and authored supports.
- Added Rubin's survey optical arrangement, shared primary/tertiary backing,
  annular secondary and camera housing. Its facility now has a faceted
  enclosure, base and tiered service building, doors and a separation joint.
- Added ELT scenery with a nominal 39 m primary, 798 installed panels and
  five mirror roles. The thin-support CPU/Metal cutaway discrepancy was
  resolved by the ordinary triangle-intersection repair below. Added draft
  OWL open-frame scenery and an independently placed roll-away shelter, with
  shared component instances and connected structural attachments.
- Added Griffith's rotunda, wings, rear planetarium and east refractor with
  a source-sized 12-inch objective. The west solar instrument is a closed
  roof proxy; none of these buildings is an engineering replica.
- Added Spitzer, Hubble and Webb spacecraft templates and checked custom
  apertures. Webb has 18 primary panels and five shield layers. These are
  ordinary scenery meshes, not compound optical prescriptions or orbits.
  See [space observatories](doc/space-observatories.md).
- Added Maunakea families `cfht`, `gemini-north`, `irtf`, `uh-88`, `ukirt`,
  `jcmt`, `sma-antenna`, `sma-array` and `vlba-maunakea`, alongside Keck and
  Subaru. Shared dish/array components preserve prototypes without expansion.
- Added Haleakala families `panstarrs-1`, `panstarrs-2`, `faulkes-north`,
  `dkist`, `atlas-haleakala`, `tlrs-4`, `solarc`, `mees-solar`, `aeos`,
  `mssc-1-6m`, `mssc-twin-1-2m`, `geodss` and `geodss-site`. Source notes
  distinguish current, ended and removed facilities, nominal apertures and
  picked instrument proxies. See [Hawaiian observatories](doc/hawaiian-observatories.md).
- Added the personal telescopes `dobsonian-36`, `apo-refractor-5` and `sct-24`, with their
  mounts, bases and characteristic structure at the requested apertures.
  Their opaque objectives remain display proxies.
- Added fifteen reusable yard kinds: mobile crane, mirror gantry, scaffold
  tower, service truck, utility pickup, equipment trailer, shipping crate,
  equipment cabinet, generator, cooling unit, access stairs, work platform,
  handrail, cable tray and weather mast. Their dimensions and parked poses
  are authored. See [yard equipment](doc/yard-equipment.md).
- Added optional height above terrain and explicit placement headings,
  retained by relative camera and named launch placements. Added golden-hour
  working-site YAML and summit/yard/Moon study inputs. These include access
  lanes, equipment, personal telescopes, stones and vegetation; a study input
  alone is not a completed render.
- Broader optical/OWL acceptance, general ALMA/SKA recipes, radomes,
  CORONA reconnaissance scenery, gravitational-wave instruments and the flyby
  remain on the [feature backlog](doc/feature-backlog.md).

### Guide-star light, mist and the night sky

- Added finite Gaussian outgoing beams to CPU and Metal: launch radius,
  geometric divergence, axial reach, wavelength and display strength.
  Spectral coordinates survive observer conversion. Beams appear through
  scattering, without an emissive skin in vacuum.
- Added named common boresights and gnomonic sky-tangent offsets in arcseconds.
  Existing explicit pointing still works. The four-beam examples replace
  degree-scale splay with a cited GALACSI pattern; launcher positions remain
  distinct while their axes share the same pointing frame.
- Added visible guide-star launch tubes aligned with resolved beam axes and
  exits. Sodium-layer focus, finite-distance parallax, fluorescence and an
  adaptive-optics loop are not supplied by these outgoing beams.
- Added bounded beam-aware volume sampling on CPU and GPU. Samples refine
  near conservative beam intervals while Gaussian tails retain ordinary
  integration; the bounds do not cut the light off. See [guide stars](doc/guide-stars.md).
- Added prescribed grove mist over canopy footprints and regional mist over
  clearings. Overlapping atmosphere layers can combine terrain-following
  weather, level cloud decks and thinner upper air, clipped by the ground.
  Extinction, scattering, wind, depth and coverage remain explicit controls.
- Retained redwood transpiration and fog research beside the atmosphere
  recipes, with model limits. Tree count does not produce a measured water
  flux; these fields do not simulate condensation or transported moisture.
- Fixed the visible Moon blocking its own light. Distant nonreflecting
  emitters no longer enlarge local secondary-ray offsets, and pure GPU
  emission skips diffuse lighting queries.
- Corrected three Moon-only night inputs that also supplied warm solar
  twilight. Added independent surface and occluded-mist direction references
  plus scene-intent guards. **Light-1 remains open:** surfaces sample sky
  illumination, while painted mist scatters explicit lamps. See
  [lighting regressions](doc/lighting-regression.md).

### Fire, smoke, sparks and fuel

- Added composite `flame.Field` sources with independent burn windows,
  fades and additive transport. A checked `manufacture.Burn` recipe creates
  single sources, seeded clusters or lines independently of fuel geometry.
- Added unit-bearing YAML fire controls for layout, temperature, composition,
  smoke, sparks and finite light cells. Spectral emission combines continuum
  and lines; sodium references and authored powers remain distinct. Hot
  tongues and soot detail are authored appearance controls.
- Added repeatable `embers` firebrands with seeded birth IDs, prescribed
  wind/loft, linear drag, gravity and cooling. Living-particle and aggregate
  spark budgets bound work, including distant-time release queries.
- Added `firefront` stand recipes for prescribed ignition ages in bounded
  spatial groups, smooth burnout and shared smoke. These schedule effects;
  they do not predict spread or consume tree geometry.
- Added the CPU/Metal `fuelbed` surface heat/fuel model after Mandel et al.,
  with explicit timestep/stability checks, retained state, rewind and owned
  snapshots. Woodland/Moon and overhead study scenes distinguish this model
  from the earlier prescribed-front film.
- Added optional `NewAdvected`/`NewMetalAdvected` heat transport and matching
  `heat-transport` YAML. Joint diffusion/cooling/advection bounds and encoded
  weight checks refuse invalid steps. Ambient inflow and fuel staying put
  are explicit; the supplied velocity is not coupled atmospheric wind.
  Reacting CPU/Metal acceptance and spread studies remain open. See
  [heat transport](doc/fire-heat-transport.md).
- Added finite isotropic volume lights in `emit`, with ball-averaged falloff
  held to independent ray quadrature. Composite flame fields and their
  finite lights prepare through Metal; shared thermal palettes avoid
  rebuilding identical source tables.
- Added explicit cached GPU smoke lighting, recomputed for changing frame
  lights. Custom paint/emission retain direct evaluation; unsupported cached
  rig use is refused. This is an approximation with recorded image budgets,
  not exact transmission reuse or a persistent scene cache.
- Crown ignition, moisture, slope feedback, firebrand spotting, transported
  burn history and coupled smoke remain open. See [fire recipes](doc/fire-recipes.md)
  and the separate [surface-fuel model](doc/fire-fuel-design.md).

### GPU traversal, preparation and readback

- Track the originating analytic sphere through outgoing shadow, skylight
  and mirror queries. A convex sphere cannot intersect its own outward ray;
  excluding that primitive avoids false planetary self-hits without raising
  every object's offset. Inward rays and other occluders retain their tests.
  Matched native ocean and negative-sun-altitude controls support the repair;
  general planet-scale intersection accuracy remains unproved.
- Checked pinhole pointers now use the GPU-generated camera-ray path, like
  pinhole values. The `film.PinholeOf` constructor no longer accidentally
  selects full CPU ray generation and upload through its pointer result.
- Added explicit `gpu-native-shadows` and `gpu-native` modes alongside
  software traversal. Nested Metal acceleration structures retain shared
  meshes/groups, instance/material identities and clear/opaque masks.
  Native ray tracing requires supported macOS/Metal hardware; unavailable
  backends and unsupported scenes are reported rather than relabelled.
- Native primary candidates are refined with Slayer's predicate; refused
  candidates use software traversal and counted fallbacks. Native shadow
  candidates now receive the ordinary-mesh check too, with a full software
  shadow query after rejection rather than assuming the ray is clear.
- Stabilized thin-triangle intersections by translating/shearing into the
  ray frame and using compensated edge determinants. This fixes the ELT
  cutaway and scaffold self-shadow witnesses without a larger ray bias or
  looser image tolerance. It does not claim the paper's full double-fallback
  watertight guarantee. See [triangle precision](doc/triangle-precision.md).
- Added CPU/GPU shared mesh-group hierarchies and direct hit normals, avoiding
  expanded foliage and discarded normal searches. Scaled/group intersections
  retain world ray windows. Degenerate triangles and coincident dome-pole
  corners are rejected before their edges or rounded area can create faces.
- Added `gpu.Profile` stage timings, logical sample/input bytes, peak host
  readback-band bytes and native query/fallback counts. Failed bake/query
  paths preserve spent work and distinguish failure from a successful miss.
- Added `Kernel.GridBands` and incremental pixel finishing to the tracer.
  One reusable host sample band replaces the full raw-sample result array;
  global row order, per-pixel summation, coverage, nearest depth and counters
  survive band boundaries. Finishing respects the Go CPU allowance. The
  full-array `Grid` API remains available.
- Streamed flame density/emission, wispy density and smoke-light bands now
  write into reserved packed scene ranges. Writers check lengths, ordering
  and reservations; incomplete output is not a completed field.
- Replaced per-record named-field packing with ABI-aware row plans and final
  capacity reservations. Dropped duplicate construction arrays after their
  consumers finish and avoided whole-input temporary C copies at synchronous
  Metal copying calls. Kernel function switches retain existing resources.
- Added host-only `render/emptyspace` occupancy masks, fingerprints, packet
  bounds and certified sample prefixes. Interpolation halos and independent
  emission count; exterior queries and exhausted proof budgets are unknown.
  **No renderer uses these skips yet.** A pilot ray's miss proves nothing
  about its neighbours. See [the packet contract](doc/mist-packet-contract.md).
- Partial stream cleanup, native size narrowing and preallocation limits are
  explicit. Host finishing has focused evidence; matched device fields,
  lifetimes, moving views and whole-process memory/timing still need checks.
  These preparation changes do not establish lower RSS or an animation speedup.

### Terrain sources, projected detail and world coordinates

- Added optional source-height shading-normal stencils with explicit angular
  steps, bounded normal records, source-halo refusals and query counts. Added
  authored smooth transitions between landcover height colours. Defaults keep
  flat shading and stepped colours; these are appearance approximations, not
  terrain geometry or ecological accuracy guarantees.
- Added a geographic bilinear sampler, stripped float32 TIFF reading and
  separate angular row/column scales for the delivered NOAA coastal raster.
  Registration and source no-data remain explicit; legacy projected rasters
  still require square sampling. Source errors cannot become flat ocean.
- Added `scene/island` view-dependent spherical terrain preparation, bounded
  source queries, world-space matching of unequal-detail patch edges and
  camera-relative meshes. Added a Pzm/Game `island render` study command with a
  checked YAML schema, source hashes and render provenance. Its cell-span metric
  is not a certified relief-error bound.

  The first scene uses NOAA topobathymetry, an explicit radial-height
  approximation and smooth ocean proxy. Regional trees/mist and a first ascent
  sequence are now connected; wave water, richer appearance and CORONA/Keyhole
  instrument views remain open. See [the island scene](doc/island-scene.md).
- Added explicit raster budgets for decoded tile bytes/count, individual
  tiles, TIFF tags and returned grids. Bounded decoding avoids retaining the
  compressed payload and a second decoded array. LRU eviction, cache discard
  and close behavior are explicit; oversized dimensions fail before allocation.
- Added source coverage, availability-returning elevation/grid reads and
  bounded legacy windows. No-data and out-of-coverage samples remain absent;
  corrupt data and exhausted decode budgets remain errors, not sea-level ground.
- Added native source-patch IDs, deterministic subdivision, bounded sampled
  grids and meshes with an explicit source-to-world mapper. Missing sampled
  corners suppress triangles. Sampling fewer points does not bound unseen peaks.
- Added byte/entry-bounded sampled-patch LRU caching, independent of camera
  or render crop. Equivalent resolved grids share entries; stricter caller
  budgets still apply on hits. Consumer-held grids outlive cache eviction
  and count separately from retained cache payloads.
- Added budgeted `SelectPatches` refinement and `SelectProjectedPatches` for
  the actual pinhole pose, lens and raster. Caller-supplied complete bounds
  and representation displacement determine projected pixel error. Budget
  exhaustion keeps coarse coverage and reports `Limited`.
- Added bounded `PatchGrid.StitchTo` source-space edge matching for affine
  mappings. Fine heights follow coarse edge segments, missing samples remain
  missing, source caches stay immutable and added displacement is reported.
  Curved-world seams and resident patch-set management remain open.
- Added tiled float32 TIFF Predictor 3 support, with scanline scratch counted
  in the decode allowance. Unknown predictors and unsupported compressed bit
  order are refused. Fixtures include independent libtiff little/big-endian
  files and hand-derived byte planes.
- Added `SourceReference` GeoTIFF codes and strict GeoKey validation, plus
  `SampleLocation` and native-patch registration for PixelIsArea/PixelIsPoint.
  Point postings avoid an incorrect half-cell shift. Legacy metre APIs refuse
  geographic/geocentric and declared non-metre sources; untagged legacy inputs
  retain their prior assumed-metre contract. No EPSG/datum conversion is added.
- Added float64 spherical anchors, radial/tangent coordinates and render-origin
  helpers in `scene/frame`, including antimeridian normalization. Added
  conservative off-axis pinhole/frustum assessments in `scene/footprint` and
  checked view/lens construction, plus `maths.Local` and `NewSubspace`
  construction helpers. See [world frames](doc/world-frame.md) and
  [terrain patches](doc/terrain-patches.md).
- These are source and geometry foundations. Hillside still stages its static
  terrain crop; global source/datum adapters, camera-relative staging of every
  scene component, fallback coverage, overviews and moving residency are open.
  Ground-to-orbit rendering is not delivered by this merge.

### Phone terrain and ocean references

- Checked the phone JONSWAP/TMA references against retained original pages.
  Labelled the rounded peak fit as model glue while preserving numeric values.
  Donelan/Banner and the complete TMA piecewise-cap attribution remain pending;
  see [the source check](doc/phone-source-checks.md).
- Imported `titan` and `oceantex` as CPU references, preserving original file
  hashes in `doc/phone-import-manifest.json`. Titan provides a geographic
  height-provider boundary, seeded finer detail and measured/generated
  reporting; it is not a resident terrain renderer.
- Oceantex provides spectra, Fourier wave fields, sea states, optics, colour,
  shore/tide sampling and a standalone CPU camera/render reference. It reads
  its source ledger through Pzm, clones exposed value slices, bounds wave,
  image and shore allocation products and respects GOMAXPROCS.
- Standalone phone commands were omitted because they bypassed Pzm. Neither
  package is connected to Hillside or Metal. Whole-cascade wave filtering,
  unresolved slope variance, wind-height conventions, ray-march exhaustion,
  planetary curvature and source verification need work before runtime use.
  Passing package tests does not establish physical or GPU acceptance. See
  [the integration status](doc/phone-ocean-integration.md).

### Scene schemas, Grissom and development

- Added bundled opt-in ecosystem v1 validation through Pzm's YAML reader,
  recipes, stands, exclusions, fire/fuel/weather, lasers, spacecraft and
  renderer choices. Expanded schema key parity to the existing film, rig,
  camera eye, bounds, assets, text, HUD, world and planet reader vocabulary.
  Legacy documents keep their independent reader checks.
- Added ecosystem v2 as a **specification without a reader**: date, anchors,
  globe/space scenes, orbit/attitude, instrument eyes, relative cameras,
  extended sky/weather and edited shots/music timelines. Example YAMLs are
  written ahead of implementation, including Keyhole, solar corona, Webb,
  orbital and lantern scenes. The backlog records gaps in regional ecosystems,
  factory enums and hyperlapse/zoom parity; v2 is not enabled rendering.
- Added Grissom's Cocoa/Metal viewer under `cmd/grissom`: survey mosaics on
  sky triangles, Earth hexes and a standing view. It is a separate graphics
  viewer, not a Slayer ray-traced scene. Added data/setup and compatibility
  layout instructions using Celestial-data's Grissom fetch set.
- Moved build/run documentation to Game, added named command/demo/catalogue
  runs and small check scopes, and documented source-shelf/review dependency
  setup. Added feature, model, source, validation and dated render records;
  prose revisions retain source and model limitations.
- Updated Puzzlemouth to v0.1.4 and Go to 1.27.3; pinned Celestial and
  Celestial-data review candidates and added JSON Schema support. YAML uses
  `gopkg.in/yaml.v3` consistently with Pzm. Dependency releases remain needed
  before a standalone Slayer release; no phone import module upgrade was needed.

### Azazel foundations

- Added the design and private primitives for a future frame/film/check queue.
  **There is no runnable queue, shared admission transaction or supervisor.**
  See [Azazel](azazel/README.md).
- Added overflow-safe memory assessment, identified reservation proposals,
  supplied host-condition checks and their ordinary admission composition.
  Low disk takes precedence, unavailable observations hold work and a
  favorable sample cannot clear a latched stop.
- Added generation-checked stop transitions with explicit eligible resume,
  stale-resume refusal and generation exhaustion that cannot wrap clear.
  These are value proposals, not persistence or permission to launch a child.
- Added ordered snapshot replacement, owned temporary cleanup, immutable
  exclusive result installation and prepared intent/state/completion steps.
  Publication failures preserve original causes and uncertain effects for
  reconciliation; cleanup touches only its owned name.
- Added Darwin temporary-file and nonblocking lock acquisition/release
  primitives, filesystem exclusion/storage capability witnesses, and native
  process start/exit timestamp observations through the public SDK. These
  do not prove child-tree ownership, crash durability or safe recovery.
- Retained paired examples, Rapid properties/counterexamples, native witnesses
  and scoped fault evidence beside the designs. Final independent acceptance,
  root protection, recovery, supervision and engine integration remain open.

### Repairs and checks across the range

- Merged `fix/unit-tests-2026-11-09`: grouped-foliage contracts and geometry
  fixtures now follow the manufactured instances, ecosystem v2 retains the
  v1 keys, and Hillside accepts `-alight` on the GPU. Legacy window infinity
  refusal is attributed to mesh construction; geographic reads and height
  indices retain their own finite-value checks. Game configuration declares
  v0.5.0 and the branch's updated prose/lint conventions.
- Source-export cache tests now accept unavailable Git history and compare
  available provenance with the fixture's actual HEAD. Downloaded source
  metadata is nonexecutable; source commands report shelf and output failures.
- Weather refuses invalid/non-finite durations and intervals; wind summaries
  advance on grids below 32 rows. Ring hull faces wind outward, and planetary
  cooking refuses an empty atmospheric column instead of indexing it.
- Stabilized disc-span fused arithmetic under coverage instrumentation.
  Refused zero-length spherical medium spans, unrepresentable grove bucket
  coordinates and non-finite geometry before native packing.
- Added early host/device byte, smooth-index, group-member, fire/front,
  particle and prototype-budget checks. Replaced a multi-gigabyte GPU test
  allocation with bounded boundary witnesses. Actual fallback lamp reporting
  now counts the scene lamps it submits.
- Expanded real-command coverage using child binaries with matching coverage
  mode/package scope; tests decode PNG/raw output and capture terminal/viewer
  paths without opening a window. Coverage notes distinguish ordinary package
  measurement, per-file gates, generated helpers and unmeasured shader branches.
- Expanded independent mesh/group harness comparisons, committed 4K producer
  identity/input checks, source and numeric fixtures, refusal/cleanup tests,
  native ABI/layout checks and function-scoped mutation evidence. Authored
  appearance records retain their producer and provenance; non-finite output
  cannot pass a closeness check. History-rule changes remain separately listed.
- Dated results include the 2,880-frame, two-minute 4K prescribed-fire film
  and a 180-frame, thirty-second 120x camera hyperlapse. They have different
  workloads and quality settings. Neither proves the dense sixteen-sample
  forest's eight-hour target, a whole-world shot or the current tip's gates.
- Full current-branch tests, coverage, lint/sweep/Shuriken, wider mutations
  and visual/performance acceptance remain separate work. The focused
  [verification record](doc/verification-2026-11-09.md) names earlier scopes
  and limits; this changelog update runs no new unit tests.

## v0.1.0 - v0.1.1 06 november

added:

- YAML camera paths with speeds, timed easing and holds for repeatable
  sequences; numbered smoke scenes can use the same frame and resume writer
- a checked demo catalogue for numbered smokes and named scenes, with PNG
  output and optional terminal previews; selectors need Game forwarding

- fire: a flame field of many seeded patches in one medium, each with
  its own burn window and fade (flame.Field); emission from a sourced
  continuum and lines (flame.Composition); hot tongues as an authored
  temperature profile
- fire recipes in a set's yaml, every quantity with its unit: flame,
  layout (a cluster or a line), smoke, sparks and light cells, checked
  by manufacture.Burn before a scene is built
- embers: firebrands born by a seeded id, carried by wind under linear
  drag and gravity, cooling over their life; budgets bound the living
- firefront: prescribed ignition fronts over a stand's trees, in bounded
  spatial groups, their smoke following what burns. it schedules; it
  does not predict spread
- fuelbed: a surface fire from heat transfer and remaining fuel, after
  Mandel et al. (2007), explicit, on the cpu or on Metal, its stability
  bound refused when broken
- emit: finite isotropic volume lights, their falloff the mean over a
  ball, held to independent ray quadrature
- manufacture: coast redwoods and giant sequoias, their foliage grown
  recursively from shared shoot prototypes, the numbers in
  needled-shoot.yml and the research ledger with their sources
- manufacture: an optical observatory, a dome with a real slit, a mount,
  a truss and a segmented or monolithic mirror; and complexity levels
  for every kind: hull, clumps, quick and demo
- scene/shape: mesh groups, one prototype placed many times with its
  own hierarchy, drawn by the cpu tracer and the gpu without expanding
- scene/medium: grove mist over a stand's canopies, prescribed in yaml,
  drifting with the wind
- gpu: native Metal ray tracing, for shadows (renderer:
  gpu-native-shadows) or for mesh queries too (gpu-native); a cached
  smoke lighting grid (volume-lighting: cached); each stage's cost
  (gpu.Profile)
- metal: a kernel switches entry points and keeps its resources;
  acceleration structures, built from a checked plan, macOS 14 and on
- hillside sequence: timed png frames from a checked yaml plan, the
  scenery kept between frames, each frame written whole, and a manifest
  that refuses a resume when scene, plan, renderer or terrain changed
- set: an opt-in schema for ecosystem documents (ecosystem-v1); recipes,
  stands that exclude around an object and share an atmosphere, fire
  fronts, fuel beds, wind, seed and renderer
- cmd/grissom: survey mosaics laid on the sky's triangles, the earth as
  hexes, and a place to stand under the sky; a work in progress
- the harness compares mesh groups through independent reference
  meshes; committed 4k comparisons are recorded in it

changed:

- hillside's count flags are frozen at the kinds they had; new kinds
  come through recipes and the yaml
- the visible moon no longer blocks its own light; distant emitters
  stay out of local shadow offsets; pure gpu emission skips diffuse
  light queries
- sequence frames are written at fast png compression
- yaml is read through gopkg.in/yaml.v3, as puzzlemouth's yamlfile reads
  it
- the harness rule keeps a second list, read whole, for commits that
  changed the harness and what it judges and are not mechanical; every
  refusal of the rule says why

fixed:

- render/smokecache's provenance tests accept unavailable checkout history
  in source exports and check available revisions against Git's exact HEAD
- a buffer or grid larger than the device or host holds is refused
  before anything is allocated
- degenerate triangles are checked before their edges are assigned
- cmd/sources reports its output and shelf failures
- cmd/weather refuses a time that is not a number, and no longer loops
  on a grid under 32 rows (in 51931c8, with tests)
- cmd/ring's faces wind with the strip's outward direction, and planet's
  Cooked refuses an empty column where it panicked (both in 51931c8)

## v0.1.0 01 november

- the first release
