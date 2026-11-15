# How Slayer works

For a first session, follow [Getting started](doc/getting-started.md)
and [Building Slayer](doc/building.md). The detailed current guides cover
the [renderer](doc/renderer.md), [light](doc/light.md),
[modes](doc/modes.md) and [optimization catalogue](doc/optimizations.md).

Slayer renders scenes by following light backwards from the camera. It is
written in Go, with a Metal renderer for supported scenes on macOS.
Physical values should come from a published source or a documented modelling
choice; artistic choices should be identified as such.

This guide follows a render from scene construction to output.
[PACKAGES.md](PACKAGES.md) describes the packages, and
[EXAMPLES.md](EXAMPLES.md) lists scenes and commands.
The [Game workflow](doc/developing.md) covers setup, builds and checks.

## The rendering pipeline

1. Build a scene: geometry, materials, motion, lights, volumes, and sky.
2. Trace rays from the camera through each pixel.
3. Develop the resulting linear light into display colours.
4. Write a PNG, terminal image, raw frame, or contact sheet.

The frame retains linear light until development. Saving that frame as a raw
file lets you change exposure and white balance without tracing the scene
again. Spectral-band images are separate renders.

## Building a scene

`scene/thing.World` holds surfaces, volumes, gravitational fields, lamps,
sky, and haze. A surface combines a shape, material, and motion. A volume
combines a density field, material, and motion.

Shapes are defined in local coordinates. A motion places those coordinates
in the world at a given time: stationary, spinning, orbiting, growing, or
nested within another motion. The same world can therefore be rendered at
several instants without rebuilding its description.

Coordinates are right-handed: y is up, x is right, and the camera looks
along negative z. Length units belong to the scene. The orbital scene, for
example, uses thousands of kilometres.

Scenes can be built in Go, read from YAML sets, or assembled from body
templates:

- `scene/smoke` contains numbered scenes built in Go.
- `set` reads YAML scene descriptions and delegates construction to the
  relevant packages; `stage` builds a set's world for the tracer, as
  `cmd/house` and `cmd/slayer` draw it.
- `scene/bodies/template` provides bodies such as Jupiter and magnetars,
  each by name.

## Tracing light

`trace.Sampled` takes a world, camera transform, lens, time, image size,
and sampling rate. It validates the request, places the scene at that
instant, distributes rows across workers, and averages each pixel's samples.

The sampling argument is the number of rays along each axis: `n` means
`n x n` rays per pixel.

Requests need a world, finite camera position and scale, positive dimensions,
and at least one ray per axis. The total sample count is limited to
`1 << 28`. Worker count follows `GOMAXPROCS`.

For each ray, the tracer finds the nearest surface. A gravitational field
can bend the ray before it reaches that surface. Surface shading combines
material colour and emission, direct illumination with shadows, sky lighting,
and up to one mirror reflection. Haze attenuates the result toward the camera.

A ray that misses all surfaces returns the sky. Rays crossing volumes
accumulate emission, scattering, and attenuation. The default integration
uses 48 steps through a volume and 10 shadow steps toward each sampled lamp.

Volumes can shadow surfaces and one another; they do not illuminate other
objects through indirect scattering.

CPU row scheduling does not change a pixel's arithmetic for an unchanged
world, inputs and build. Backend equivalence is a separate contract;
native Metal does not promise cross-process byte identity. Do not mutate shared geometry, materials, or
media during a render.

`trace` is the one CPU tracer. `slayer`, `ring`, and `swarm` draw with it.

## Rendering on the GPU

On macOS, `gpu.Sampled` renders supported scenes through Apple's Metal API
and returns the same frame type as the CPU tracer. Kernels compile from
source when opened; building Slayer does not require a separate Metal
compiler toolchain.

The GPU supports a subset of scene features. Unsupported features return an
error identifying them. CPU renders provide the reference for GPU checks;
approximations such as baked haze have explicit test tolerances.

Several commands expose `-gpu`. A hillside set chooses `gpu` or
`gpu-native` with its `renderer` field; unsupported combinations are
refused before drawing. The [ecosystem guide](doc/observatory-ecosystem.md)
describes that path.

## The darkroom

A `render/frame.Frame` stores RGB light, coverage, and first-hit depth for
each pixel. Light is linear and can exceed one.

`render/colour` applies exposure, bloom, highlight roll-off, and sRGB
encoding. `trace.RenderIn` renders a spectral band; `colour.False` can
combine those images with a visible-light render using assigned colours.

The CPU and GPU band paths have different capabilities; see
[doc/bands.md](doc/bands.md).

`ninjafilm` models photographic stocks, including spectral sensitivity and
grain, and provides lenses with physical focal lengths. A job's `stock`
develops through one (`slayer list stocks`): one roll a job, grain fixed
by seed and frame. A stock's screen rolls highlights off instead of
clipping each colour at white.

## Output formats

- `interfaces/file` writes PNG images.
- `raw` reads and writes linear RGB in PFM files. `slayer develop` can
  adjust raw frames or sum several into a long exposure.
- `interfaces/terminal` represents each terminal cell as six subpixels in
  two colours.
- `interfaces/sheet` arranges labelled images into contact sheets.
- `interfaces/learn` approximates terminal output as a raster image.
- `hud` overlays text, frames, bars, and labels.

## Sources and reproducibility

`sources.yaml` records published datasets, their origins, licences, and
SHA-256 hashes. The `sources` command downloads them to a shelf outside
the repository and checks their bytes:

Configure `SLAYER_SHELF` in Game's Slayer project block, then run:

```sh
game run sources-fetch
game run sources-check
game run sources-verify
```

Changed source bytes fail verification. `research` exposes named values
from the project's research data, rather than requiring callers to copy
numbers from memory.

## Running Slayer

`slayer` is the front door. Build it with `game build` from the checkout
root, then run `game run slayer -- VERB ARGS`, or run `bin/slayer`. Images
go to `out/` unless a flag or the job says another place. `slayer --help`
lists the verbs; `slayer --agent-help` gives the same verb map for an agent. A verb's
`--help` lists its own arguments and flags.

| Verb | Does |
|---|---|
| `draw JOB` | draws a job, a sequence plan or a set file: frames, a movie with its sound, titles, a stock |
| `list [HEADING]` | says what there is: scenes, smokes, presets, stocks, tunes, voicings, library, kinds |
| `assess FILE` | says whether a file will draw, drawing nothing; `--judge` judges a set, `--smokes` the smokes |
| `develop JOB RAW...` | develops light already drawn through a job's print or stock |
| `tunes DIR` | serves a tune library's voting page, or checks, films or pages its tunes |
| `cache` | says what the disk caches hold; `--clear [KIND]` empties them |
| `watch JOB` | draws a job live in this terminal as it is saved |
| `edit [JOB]` | lays out watch, the editor and the agent console in tmux |
| `schema` | prints the job schema, or with `--where` the path of its copy |

Bare `slayer` at a terminal is `edit`. `slayer render FILE`, which draws a
whole set file as `hillside` does, is still read ahead of the door in its
own flags. The older `set`, `smokes` and `develop` commands are gone;
`slayer set` and `slayer smokes` print where they went.

A job is a YAML document that says which scene, at what size and rays,
with which lens, print, sound and titles, and what comes out
([job/README.md](job/README.md); the schema is `slayer schema`).

The flags `--out`, `--width`, `--height`, `--rays`, `--at`, `--camera`,
`--renderer`, `--quality` and the four switches below stand over the
job. `--dry-run` prints the job in its canonical form and each file it
would write, and draws nothing.

To make a short on a team's renders home:

```sh
SLAYER_RENDERS=~/renders SLAYER_TEAM=me slayer draw jobs/short-fire-night.yaml
```

A still goes into `frames/DATE-NAME/` and a film into `animations/DATE-
NAME/` under the renders home, the job file is copied beside it, and the
job's `outputs.note` goes into the team's notes file. Control-c stops
between frames. A second run resumes: frames drawn under the same job,
the files it reads and the build are kept, and the rest are drawn.

A film that writes a movie and no frames removes its frames once the
movie is written; `--keep-frames` keeps them. The door's
[README](door/README.md) has where things go and what a resume checks.

The older runs still work for the scene commands:

```sh
game run swarm
game run orbital -- -far
game run hillside -- -night
game run smoke-one
```

See [EXAMPLES.md](EXAMPLES.md) for scene descriptions and command options.

## How a picture is drawn

By default a picture uses `quality: exact`, the configured tracer's own
answer. This does not establish bit identity between backends or every
native process, nor remove the model's finite integration limits. Anything else is asked for by name and
says so in the picture's record. Detail is in
[doc/flags.md](doc/flags.md).

- `quality` is how much of a frame is drawn: `exact` (the default) or
  `preview`. It is not about how light is found.
- `vibes: on` is for films to watch and not to measure: it turns on
  every shortcut. `fastlight: on` turns on the light shortcuts. A
  shortcut is one named approximation, `on` or `off`: `lamp-store` (mist
  and smoke lit from a store of each lamp's light) and `sky-looks` (the
  sky's sixteen looks shared among a pixel's rays).

  Each is a flag of the job, of the set under `scene` (the first two
  only), and of the door (`--vibes`, `--fastlight`, `--lamp-store`,
  `--sky-looks`). The first that says anything decides: the shortcut's
  own word, then `fastlight`, then `vibes`; each read from the command
  line, then the job, then the set. GPU only.

  Measurements are in [doc/fast-light.md](doc/fast-light.md).
- `fantasy` is a block in a set that states what the scene says and the
  physics does not (its keys are `lake`, `lakeglow`, `moon`, `moonglow`,
  `giants`, `close` and `closeglow`; see `hillside/fantasy.go`). Hillside prints a line for each key, saying
  it departs from the measured or derived value. It is a statement about
  the scene, where `vibes` is about how it is drawn.
- Caches are meant to be exact: a hit is what a fresh run computes, byte
  for byte. Two known gaps: no key covers the files a set reads in turn
  (so a kept still is served only with `SLAYER_KEPT_STILLS=on`), and a
  pinned file's re-hash is decided by its size, time, inode and device.
  `slayer cache` says what they hold; see [doc/caches.md](doc/caches.md).
- The strict setting, science, is not one switch today. It
  is the replay of selected pixels by `fermat` (`hillside -prove`), the
  harness's checks and a signed binary. No shortcut may be combined with
  it: a proof beside any shortcut is refused by name.

## Music

Slayer has its own synth, `cheeptune`, with two machines: the Game Boy
machine (four channels simulated at the register) and the studio (FM, a
filtered analogue voice, plucked string, struck objects, synthesised
drums, real chords, a reverb and a delay).

A tune is a score, which says what is played, with a voicing, which says
on what; a library tune also has a cut sheet, `NAME.cuts.json`, with its
mood, its parts by bar and the voicing it was written for.

- A job names a tune as `sound: {tune: "@NAME", voicing: "@VOICING"}`, or
  an audio file as `sound: {file: PATH, from: {value, unit}}`. `slayer list tunes`, `voicings` and
  `library` say what is built in.
- To write one, read `cheeptune/SCORING.md` (the format), `VIDEO.md` (a
  tune for a video) and `SOUNDS.md` (an instrument).
- To choose, `slayer tunes DIR` serves a page on this machine only
  (`127.0.0.1:8417`) with each tune's piano roll as a film, to keep or
  toss each; votes are written to `votes.json` in the folder and move
  nothing. `--check`, `--films`, `--plates`, `--page` and `--sounds`
  write instead of serving.

Detail is in [doc/music.md](doc/music.md).

## Checking the renderer

Slayer began beside Lamb, an earlier tracer. Its checks were designed to
establish correctness independently of the implementation.

- **Smoke scenes** record successive rendering capabilities. Their numbers
  remain stable, and a change to an existing smoke is a separate commit.
- **burning-trousers** compares exported renderer APIs with closed-form
  physics and independently computed expectations. A commit must not
  change both the harness and the renderer it checks.
- **fermat** selects 64 pixels from a hash of a raw image's bytes and
  replays their computation. Each must match bit for bit. Run it through
  `hillside -prove` with the original rendering parameters.
- **Validation** reports unsupported or invalid inputs at construction
  boundaries instead of silently rendering an approximation.
- **Mutation testing** changes individual operations and runs the tests.
  A surviving mutation identifies an untested distinction or a case whose
  equivalence needs an explanation.

### Running the tests

Tests are in tiers (doc/test-tiers.md). Plain `go test ./...` and
`game check` run the non draw tier: no GPU device is opened and nothing is
drawn, and the scope is substantially smaller than the draw tier.

- `game run test-hot` runs the non draw tests of the hot path packages
  listed in `tools/tiers/hot.txt`; use it on every edit.
- `go test ./...` or `game check` runs every test that does not draw.
- `game run test-draw` runs only the tests that draw: in each package, the
  tests the plain build lacks, with `-tags draw -p 1 -timeout 60m`, one
  package at a time, going on past a failure and naming each package that
  failed at the end. Run it before a merge, with nothing else drawing on
  the machine.
- `game run test-all` runs every test with `-tags draw -p 1 -timeout
  60m`; run it before a release.
- `game run check-tiers` confirms the tiers hold every test name once.

A test that draws belongs in a `NAME_draw_test.go` file whose first line
is `//go:build draw`.

## Viewing the smokes

`slayer assess --smokes` renders each scene at terminal resolution, 179 by
50 cells, and as a finer raster image. It checks that the terminal view
agrees with the raster image at the coarser resolution.

```sh
slayer assess --smokes
slayer assess --smokes 7 --out /tmp/smoke7
slayer assess --smokes gas holes
slayer assess --smokes --small
```

Every smoke is in one group: light, worlds, lens, energy, gas, holes or
earth. `--smokes` takes groups and smoke numbers, and with none does every
smoke. `--small` draws at half the size and with fewer rays, and names
each smoke it cannot judge that way. With no `--k`, each smoke is judged
at the least its world allows. `game run smoke-one` judges smoke 1.

The tests judge the four heavy smokes (29, 35, 36, 37) small. Before a
release, judge them at full size, which takes about ten minutes:

```sh
slayer assess --smokes 29 35 36 37
SLAYER_SMOKES_HEAVY=1 go test -run TestHeavy -timeout 30m ./render/smokejudge/
```

A cache keyed by the rendering binary avoids repeating unchanged work;
`--fresh` draws again what it holds. The uncommitted `smokes/` directory
contains images, timing records in `results.jsonl`, and an `index.html`
gallery.

## Playing lights to music

Slayer's lights can play in time with a song. A show needs two files:
a score, the track's measured beat grid and transcribed notes, and a
cue sheet, what the lights do, written in bars.

`cueshow` compiles the two into the moment's state as key=value
lines; `lightshow` run with `CUES=cue.env` obeys them - rig level and
blackout, beam colour per beat, bar-locked mirror sweeps, strobes
gated to the grid. Editing the cue sheet is how the show changes;
nothing recompiles.

```sh
go run ./improv/moonlake/cueshow score.yaml cues.yaml 41.5 > cue.env
```

The number is the moment in seconds; the optional last argument is the
frame rate, which bounds the window a note must start in to strike a
strobe.

`metronome` is the check: it plays raw frames to ffmpeg with the song
underneath, flashing a square on each measured beat and a band for
each cue as it fires, so the grid is checked by eye and ear before a
render spends hours.

The first show is moonlake, and
[its README](improv/moonlake/README.md) is the worked example: the
pieces, the flags, how the score was pulled from the audio, and the
frame-by-frame pass structure. The set is `sets/moonlake.yaml`.

## Further reading

- [Sets](doc/sets.md): what a set file says.
- [Flags](doc/flags.md): exact, vibes, fantasy and the strict setting.
- [Music](doc/music.md): scores, machines, voicings, the library.
- [Caches](doc/caches.md): what is kept, where, and how to clear it.
- [Fast light](doc/fast-light.md): the measurements.
- [Performance](doc/fast.md): rendering optimisations and planned stages.
- [Spectral bands](doc/bands.md): band rendering and CPU/GPU differences.
- [Takes](doc/take.md): camera paths and shutter sampling.
- [The lantern](doc/the-lantern.md) and
  [the orbital scene](doc/the-orbital-scene.md): scene design and history.
- [Harness rules](burning-trousers/INSIDE.md): independent validation.
- [Technical debt](DEBT.md): outstanding style migration.
