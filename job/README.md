# job

A job says how to render a scene. A set file says what is in the scene; a
job names a scene and says the film, the time or span, the camera, the
development, the sound and titles laid over it, and what comes out.

The same document serves several workflows. `slayer draw JOB` renders
it here. `slayer watch JOB` draws it live in a terminal as the file is
edited. Queue-oriented fields describe admission, not a public `send`
verb in the current Slayer door. The interactive editor
edits this document and nothing else, so anything done by hand can be done
again from the file.

This package reads and checks the document. It draws nothing and imports no
renderer: only `camera`, `maths` and Puzzlemouth's `yamlfile`.

## The document

The schema is `job.schema.json`, named
`https://github.com/janearc/slayer/schemas/job-v1.json`. It is closed: a
key it does not know is refused with its file, line and column. Every key
carries a description, because the editor shows that description on hover.

Quantities are written `{value: 20, unit: s}`, as in a sequence plan.
Paths are relative to the job file.

| Key | Says |
|---|---|
| `$schema` | the schema's name; required |
| `scene` | which scene; required; exactly one form, below |
| `sources` | source names, identifiers and optional notes for attribution |
| `film` | the picture's size, rays and lens |
| `renderer` | `cpu`, `gpu`, `gpu-native-shadows` or `gpu-native` |
| `quality` | `exact` (the default) or `preview` |
| `vibes`, `fastlight` | flags, each `on` or `off` |
| `shortcuts` | named approximations, each `on` or `off` |
| `camera` | a camera of the scene by name, or a path |
| `at` | a still: the scene's time |
| `fps`, `duration`, `start`, `scene-duration` | a film: its rate and span |
| `shutter` | motion blur: the open angle and glimpses a frame |
| `print` | plain development: knee, top, bloom, stops |
| `stock` | development through a ninjafilm stock |
| `titles` | words drawn over the frames |
| `sound` | the soundtrack |
| `outputs` | what is written, by role |
| `limits` | memory and time ceilings, for the queue only |

### sources

An optional list records what informed the job. Each entry has a `name`,
a `uri`, and an optional `note`. Records are kept in the canonical job
and are returned by `Job.Sources()`. They are attribution metadata and
do not fetch, import or verify the source.

### scene

Exactly one of:

- `{set: PATH}`: a set file.
- `{smoke: N}`: a numbered smoke.
- `{named: NAME, with: {...}}`: a scene built in Go, by name, with its
  parameters. The parameters are checked by the scene, not by this schema.
  This form is a bridge: each named scene is owed a set file.

  `play: SHEET` beside `named` is a cue sheet (package `cuesheet`) that
  plays the scene's channels from the music; its clock is the job's
  `sound`: a tune or supplied audio with a measured/estimated beat grid.
  See [music](../doc/music.md) for playable scenes and clocks.
- `{frames: DIR}`: frames already rendered (`.pfm` or `.png`, in name
  order). Nothing is traced; only development, titles, sound and the film
  are done. A `.png` holds light already clipped at white; a `.pfm` does
  not.

### film

`preset`, `width`, `height`, `rays`, `wide`, `tall`, `exposure`,
`aperture`, `focus`, `mode`. All are optional. A preset gives width and height, and some give a rate; a key
written beside a preset stands over it.

`samples` instead gives the exact total samples per pixel, from 1 to 64;
do not write it beside `rays`, which remains the number along each side.
Exact samples support CPU and software-GPU stills without a physical rig,
aperture, wire mode or region. `--spp` overrides either form; `--rays` restores the
legacy per-side form. Do not give both flags together.

`mode: wire` draws the edges of what the picture's first rays meet, in
place of their light: the same rays, camera, time and things as the
light picture, so the two lie one over the other exactly. On a mesh face
a pixel within half a pixel of the face's nearest edge is white, and the
face between is a faint grey.

A surface with no polygons (a sphere, a plane, a globe, a marched shape)
is a flat dim blue, with no edges made up for it. A height field is that
blue with its sample cells drawn in a pale blue: the cells are real, but
they are not polygons.

A shape that is none of these is a dim magenta. The sky and every volume
are black; nothing is lit or shadowed. The edge's width, its light and the
tints are picked.

The wire is drawn on the cpu only, and `renderer: gpu` or a gpu set is
refused by name. It develops through a plain tone, exposure one; a job's
print or stock still stands over that. `mode: light` is the default, and
a job that says it has the key it would without it.

The presets are data, in `presets.yaml`:

| Preset | Size | Rate |
|---|---|---|
| `portrait` | 1080 by 1920 | |
| `half-portrait` | 1080 by 960 | |
| `portrait-preview` | 360 by 640 | 6 |
| `4k` | 3840 by 2160 | |
| `2k` | 2048 by 1152 | |
| `720` | 1280 by 720 | |
| `320` | 320 by 180 | |
| `imax` | 4096 by 3002 | |
| `imax-2k` | 2048 by 1501 | |
| `video-preview` | 320 by 180 | 6 |
| `video-preview-720` | 1280 by 720 | 6 |

`imax` and `imax-2k` are the sizes the moonlake recipes use. The rest are
picked.

`wide` is the angle across the picture and `tall` the angle down it, in
degrees. A job says at most one. In portrait, `wide: 55` sees about 85
degrees down; `tall` is the way to say what a tall frame should see.

`aperture` is how wide the lens is open, across, and `focus` how far off
it is focused, both in the scene's own unit: metres for a set or a smoke.
A job says both or neither. With them, things at the focus distance are
sharp and things nearer or further are blurred, more the wider the
aperture. Without them the lens is a pinhole and everything is sharp.

A real camera's aperture is millimetres. A wider one, a few centimetres
for a scene in metres, is what makes the blur plain in a small picture.

The blur is made of the pixel's rays, each through another point of the
aperture, so it wants `rays: 4` or more; fewer is noise. On the gpu the
kernel makes those rays and the frame costs what a pinhole's does. A set
whose film has a rig takes its lens from the rig, and an aperture over
it is refused.

Bounds: width and height from 1 to 8192. Rays along a pixel's side are
from 1 to 16 for a whole picture and 1 to 64 for a region.

`region` names a rectangle in the full picture, counting from the top left:
`{left: 20, top: 30, width: 80, height: 60}`. The rectangle must be
nonempty and fit the resolved picture. Its pixels keep the full picture's
camera and scale. A region is a still; unsupported renderers and effects
are refused. `slayer draw --region 20,30,80,60` stands over the job.

### flags and shortcuts

A flag says the picture departs from the exact tracer's, on purpose. It
is not a quality: `exact` and `preview` say how much of a frame is
drawn; a flag says how its light is found. A picture drawn under one
says so in its record.

    vibes: on
    fastlight: on
    shortcuts:
      lamp-store: off

A shortcut is one named approximation, `on` or `off`:

- `lamp-store`: mist and smoke are lit from a store of each lamp's
  light, made once a frame on the gpu, in place of a march toward every
  lamp at every step. A shadow cast into mist is as soft as a cell of
  the store.
- `sky-looks`: the tracer takes sixteen looks at the sky from every
  surface point a ray meets, a shadow ray along each, at every ray of a
  pixel. Shared, the sixteen are dealt round the pixel's rays: at four
  rays a pixel each takes four, at sixteen each takes one. At one ray a
  pixel nothing changes. Sky-lit shade is a little rougher within a
  pixel.

`fastlight` is the flag for the light shortcuts, those two today.
`vibes` is the flag for films to watch and not to measure: every flag
and every shortcut.

The first of these that says anything decides a shortcut: its own word
under `shortcuts`; then `fastlight`, for a light shortcut; then `vibes`.
Each of those is read from the command line, then the job, then the
set. With nothing said a shortcut is off.

An individual shortcut can stand over an umbrella flag, including an umbrella explicitly set off.
To disable both light shortcuts unconditionally, set `lamp-store` and
`sky-looks` off by name. Turning off only the lamp store while shared
sky looks remain enabled is refused.

A set says the two flags under `scene`, the same words. It says no
shortcuts.

Refused, by name: any shortcut on the cpu, through a rig's lens, for a
named scene, or beside a proof (`-prove`), which replays pixels on the
exact tracer; `lamp-store` with more than eight lamps or beside
`volume-lighting: cached`; `sky-looks` without `lamp-store`, since they
are drawn by one mode of the kernel.

`doc/fast-light.md` has the measurements and what each shortcut gets
wrong. Surfaces do not read the lamp store, and should not be made to:
a surface point lies between cells that are inside its own trunk or
under its own ground, which hold no light, so it darkens in patches.

A second light shortcut, `disc-shadows`, was built and taken out again
on 12 November 2026 (commit 13adcc0e holds it): one shadow ray a camera
ray toward a point on the moon's or sun's disc in place of eight. It
drew correctly and saved nothing: a shadow ray through a forest of five
thousand trees costs about 35 nanoseconds, and removing every surface
shadow ray from the cloud sea changed no timing that could be read.

The schema's line for each flag and shortcut is its documentation: the
editor shows it on hover and the door's help prints it (`Switches`).
The canonical form holds a flag as said, `off` too, since a set may say
`on`; it leaves out a shortcut the job's own `fastlight` already says.
A job that says none has the key it had before there were flags.

### camera

Either `{name: NAME}`, a camera the scene defines, or a path in the form a
sequence plan uses: `unit`, `look-at`, `up`, `moves`, `zoom`. `zoom` may
stand beside `name`. `unit` must be the scene's own unit, which for sets
and smokes is `m`, and for a named scene the unit it names, such as `a
thousand kilometres`. This package reads any unit; the renderer, which
knows the scene, refuses one that is not the scene's.

### time

A still says `at`. A film says `fps` and `duration`, and may say `start`
and `scene-duration`. A job says one or the other. A film preset's rate is
used when the job says no `fps`. Frame `i` of a film is at
`start + i / fps`; with `scene-duration`, the scene's time is stretched
over the film's span, as in a sequence plan.

### shutter

`{angle: {value: 180, unit: deg}, glimpses: 4}`. Each frame is drawn
`glimpses` times across the part of the frame's interval the angle leaves
open, and the light is averaged before development. A frame costs
`glimpses` renders. Absent, a frame is one instant.

On the gpu a camera path's own move is also blurred inside each glimpse:
every ray of a pixel looks from where the camera is at its own moment of
the glimpse's share of the open interval. So `glimpses: 1` blurs a camera
move at the cost of one render, and wants `rays: 4` or more.

Things that move in the scene are not blurred by this, only by glimpses:
each glimpse is the scene at one instant. On the cpu a camera move is
blurred by glimpses alone, as before.

### print and stock

`print: {knee, top, bloom, stops}` is the plain tone curve. `stock:
{name, seed, rated}` develops through a ninjafilm stock: `name` is one of
the shelf (`soot`, `ash`, `smoke`, `dusk`, `pollen`, `lantern`), `seed`
fixes the grain, `rated` is stops of push or pull. A job says one or
neither; with neither, the scene's own print is used.

### titles

`{theme: PATH, rows: N, elements: {NAME: {...}}}`. An element is a hud
element as a set writes one (`kind` of `text`, `frame` or `bar`, with its
place and roles), and may add `from` and `until` in seconds of film time.
Outside that span it is not drawn. `theme` is a colourway file; absent, the
hud's own default is used. `rows` is how many rows of cells the picture is
laid out in, so fewer rows is larger type.

### sound

One of:

- `{tune: SCORE, voicing: VOICING}`: music rendered here from a score and
  a voicing, each a path or `@name` for a built-in. The beat at any moment
  is known from the piece.
- `{file: PATH, from: {value, unit}, score: PATH}`: an audio file, read
  from `from` onward. `score` names a score file giving its beats; without
  one the beats are estimated and said to be.

Sound longer than the film is cut at the film's end; shorter sound is
followed by silence.

### outputs

By role, each a path: `picture` (a still), `raw` (a still's light, as a
`.pfm`), `frames` (a film's directory of numbered pictures), `movie` (the
film as an `.mp4`, with its sound), `note` (one line for the renders page).
A still writes `picture`; a film writes `frames`, `movie` or both. A job
with no outputs is drawn to the default place its verb names.

## Precedence

For every value more than one level can say, the first that says it wins:
the command line, then the job, then the set, then the scene's own default,
then the built-in default. The job records which values it said, so the
levels below know which to leave alone.

## Sequence plans

A sequence-v1 document is still read. `Lift` turns one into a job with the
same frames: `scene` or `smoke`, `film`, `fps`, `start`, `duration`,
`scene-duration`, `print`, `camera`, and `output` as `outputs.frames`.
Nothing in a shipped plan is edited, because a plan's hash is what lets a
film already rendered resume.

## Identity

`Key` is the sha256 of the job's canonical form: every value that changes
the picture or the sound, with defaults filled in, in a fixed order, and
under `inputs` the sha256 of every file the job reads. Output paths,
limits and the note are not in it.

The files are the set or the frames folder (every file directly in it,
by name and bytes), a played scene's cue sheet, the titles' theme, and
the tune, voicing, sound file and beats. `Inputs` names them. A built-in (`@name`) is not a file: the
build that draws the job holds it, and the drawer adds the build to its
own record. Files the set itself reads (terrain, assets) are not covered.

`PictureKey` is the same without the sound and its files: two jobs with
one picture key draw the same frames whatever their sound, so a changed
tune remakes the movie from the frames already drawn. A played scene
(`scene.play`) is the exception: it moves with its tune, so its sound
stays in its picture key, and a changed tune or voicing draws its frames
again.

A job whose canonical form cannot be written as JSON (a number
that is not finite, in a named scene's `with`) is refused. The digests are
worked out once a job, when a key is first asked for.

## What the package gives

- `Read(path)`: a checked job, its paths resolved, its preset applied.
- `Lift(plan fields)`: a job from a sequence plan's values.
- `With(overrides)`: a job with the command line's values laid over it.
- Accessors for each part, each saying whether the job said it.
- `Key()`, `PictureKey()`, `Canonical()` and `Inputs()`.
- `Presets()`: the table, for the editor and for `slayer list`.
- `Schema()`: the schema's bytes, for the editor.

## What it refuses

Each by name, with the file and place:

- a key it does not know;
- two forms of `scene`, or none;
- `with` without `named`;
- `play` without `named`, or without sound to keep its time;
- `at` beside `fps` or `duration`;
- a film with no duration;
- a film with no rate from the job or its preset;
- `wide` beside `tall`;
- `print` beside `stock`;
- a stock not on the shelf;
- a camera that is a name and a path, or neither, or a path without its
  `unit`, `look-at` and `moves`;
- a camera path or zoom the camera package refuses;
- sound that is a tune and a file, or neither, a tune without its
  voicing, or `from` or `score` without a file;
- a title whose `until` is not after its `from`;
- a still with `movie` or `frames`;
- a film with `picture` or `raw`;
- `shutter` or `sound` on a still;
- a `camera` or `shutter` beside `frames`, where nothing is traced;
- an `aperture` without a `focus` or a `focus` without an `aperture`, or
  an aperture as wide as its focus is far;
- a preset it does not have;
- a bound exceeded.

A refusal wraps `ErrRefused`, so a caller can tell a wrong job from a
file that could not be read.

## Choices made in building

The note above left these open; each was chosen here.

- The schema checks shapes, bounds, names and unknown keys. The relations
  in the list above are checked in Go after it, so each refusal says in
  plain words what is wrong and where. The editor's schema does not flag
  them; reading the job does.
- A job is a film when it says `fps`, `duration`, `start` or
  `scene-duration`. A preset's rate alone does not make a film; it is the
  rate of a film that says no `fps`.
- A still that says no `at` is drawn at scene time zero; `At` says it was
  not said.
- The types are exported with private fields, as `set`'s are, so a caller
  can name what it holds; each part has accessors and nothing else.
- `720` and `320` may be written bare as presets, as well as quoted.
- `outputs.note` is the note's words: one line, with no tab and no line
  break in it. It is not a path.
- `print` says `knee` and `top`; `bloom` and `stops` default to none.
  `stock` says `name`; `seed` and `rated` default to zero.
- A title is a hud element of kind `text`, `frame` or `bar`. A set's
  `label` is left out: it pins to a thing on the set, which a job cannot
  name.
- Bounds the note did not give: a shutter opens above 0 to 360 degrees
  with 1 to 64 glimpses; `stops` -40 to 40, since a rig's raw prints at
  moonlake's 23; `rated` -6 to 6; `seed` 0 to 4294967295; `rows` 2 to
  500, as a set's hud; `limits.memory` in GiB up to 1024, `limits.time`
  in seconds up to a week. The time keys keep a sequence plan's bounds.
- The canonical form holds the paths absolute, so two jobs naming
  different files have different keys, and so does the same job in a
  tree laid out elsewhere. It leaves out the preset's name, since its
  size and rate are what it gave, and a renderer the job does not say.

  It fills in quality `exact`, start 0, easing `linear`, up as the
  scene's, zero bloom, stops, seed and rated, a file's sound from 0, and
  a text's alignment `left`. Numbers are written as `encoding/json`
  writes a float64. The titles are a list in it, in their order.
- `With` lays each override over the job and checks the whole again, as a
  file is checked. `Output` is a still's picture, or a film's movie when
  it ends `.mp4` and its frames otherwise; the job's other outputs stay.
  `CameraName` takes the place of the job's camera and keeps its zoom. An
  empty override is refused.
- `Lift` takes a `Plan`, a struct of the plain values a sequence plan
  holds, which the session fills; the camera goes as the plan's YAML. It was
  chosen over exported options because it is the smaller door, and the
  job it makes is checked through the same schema as a file.

## Tests

Every test that says something works is answered by an anti-test, marked
`// anti-test of TestX` above it (`go run ./tools/antitests ./job/`).
The keys have the most of them: a key that misses an input reuses a stale
picture.

They hold that each knob of the job moves the whole key, the
picture's key or the light's as it should, that each file the job reads
(set, frames, theme, tune, voicing, sound file, beats, cue sheet) moves
the keys when its bytes change, that a file that is absent, present or
unreadable is three keys, and that what is not a picture (outputs, limits,
the note, key order) is not in them.

### Tests with no anti-test, and why

None. Every test in the package has an anti-test.

### Tests that draw

None. No test draws, reads a shelf or stages a scene. The suite is about
0.4 s (`GOWORK=off GOMAXPROCS=3 go test -count=1 ./job/`), all quick.

### A frame that is a link

A frames folder's digest follows a frame that is a symbolic link and
takes the bytes it points at, so a linked frame changed where it is kept
changes every key (`TestAFrameThatIsALinkIsKeyedByWhatItPointsAt`). It
took only regular files until the test pass of 12 November found it.
