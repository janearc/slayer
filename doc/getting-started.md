# Getting started with Slayer

Slayer makes pictures and films by tracing light through a scene. This
page takes you from a fresh checkout to two finished pictures. The two
pictures below need no extra data from the source shelf.

Run every command from the checkout root. First follow [Building Slayer](building.md)
to install Git, Game and the Go version in `go.mod`, and set up the source
shelf used during the build. On macOS the build may print a linker warning
about a duplicate `-lobjc`; it is harmless.

## Set up

```sh
export SROOT="$PWD/out"
export SLAYER_CACHE="${SROOT}/render/cache"
mkdir -p "${SROOT}/render/cache" "${SROOT}/render/frames"
game build slayer
```

The build verifies the research sources first. The named `slayer` target
builds this binary; use the ordinary `game build` when retaining stamped
binaries as render evidence. See [Building](building.md) for provenance.

`SLAYER_CACHE` is one absolute folder where slayer keeps work it can do
again. `slayer assess` and `slayer draw` stop without it:

    diskcache: SLAYER_CACHE is not set

`slayer list` and `slayer draw --dry-run` run without it.

`SLAYER_SHELF` is the folder of published sources behind slayer's physical
values. Neither picture on this page reads it, and both drew with it unset.
Jobs that need data slayer does not ship, such as terrain models and star
catalogues, are the ones that need the shelf. See [Building](building.md)
for how to fetch it before drawing those.

## Your first picture

The first-picture job is a ball on a floor at 160 by 90 pixels, drawn on
the CPU. First ask whether it will draw, then what it would write:

```sh
./bin/slayer assess doc/examples/jobs/first-picture.yaml
./bin/slayer draw doc/examples/jobs/first-picture.yaml --dry-run
```

`assess` prints `first-picture.yaml will draw, as a still; nothing was
drawn`. The dry run prints the job as slayer resolved it and the two files
it would write. Now draw it:

```sh
./bin/slayer draw doc/examples/jobs/first-picture.yaml \
    --out "${SROOT}/render/frames/first-picture.png"
```

It prints the paths to the picture and raw-light file when they are written.

The picture is `${SROOT}/render/frames/first-picture.png`: a peach ball with
a hard shadow on a pale grey floor. The raw light is `out/first-picture.pfm`
in the checkout. A PNG is developed for your screen; a PFM keeps the light
as computed, before development. `--out` moves the picture and leaves the
PFM where the job put it. Git ignores `out/`.

## What a job file is

A job describes one photograph: what to draw, how big, and where to write
it. This is all of `doc/examples/jobs/first-picture.yaml`:

```yaml
$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {smoke: 1}
renderer: cpu
film: {width: 160, height: 90, rays: 1}
at: {value: 0, unit: s}
outputs:
  picture: ../../../out/first-picture.png
  raw: ../../../out/first-picture.pfm
```

- `$schema` names the job format. `slayer schema` prints it.
- `scene` says what to draw. Smoke 1 is a numbered test scene; `slayer
  list smokes` shows them all.
- `renderer` is `cpu` or one of the Metal choices. `cpu` needs no GPU.
- `film` is the frame. `rays` counts rays along each side of a pixel, so 2
  means four rays per pixel.
- `at` is the moment of the scene to draw.
- `outputs` names the files. Paths are relative to the job file, not to
  your terminal, which is why `../../../out/` lands in the checkout's
  `out/`.

Flags such as `--width`, `--height`, `--rays` and `--out` override the
job for one draw. `slayer draw --help` lists them.

## A bigger picture

The city-grid job lays out ten thousand buildings and draws them from
above at 1280 by 720. It reads only files in this repository.

```sh
./bin/slayer draw doc/examples/jobs/city-grid-10000.yaml \
    --out "${SROOT}/render/frames/city-grid-10000.png"
```

It took about two seconds here and wrote one file, the picture. The
picture is `${SROOT}/render/frames/city-grid-10000.png`: a square of street
grid seen at an angle, with grey and rust-red buildings and a few taller
towers, on a dark olive-grey ground.

The buildings are not drawn by hand. Each is made from a recipe by the
manufactory, and the job only names the set and a camera. This job has no
`raw` output, so no PFM is written.

## See what there is

```sh
./bin/slayer list
./bin/slayer list smokes
./bin/slayer list scenes
./bin/slayer cache
```

`list` names its headings: scenes, smokes, presets, stocks, tunes,
voicings, library and kinds. `list HEADING` shows one. `cache` says what
each cache under `SLAYER_CACHE` holds and how much room it may use.

## Where to go next

- [Example jobs](examples/jobs/README.md): one job of each kind to read and
  draw.
- [Job format](../job/README.md): every key a job accepts, including print
  and stock.
- [Output and resume](../door/README.md): read before moving or reusing a
  long render's files.
- [Sets](sets.md): the world files a job's `scene: {set: PATH}` names.
- [Modes](modes.md): careful measurement and fast films.
- [Renderer](renderer.md): geometry, backends, films and gravity.
- [Light](light.md): illumination, shadows, mist and development.
- [Music](music.md): choosing, writing and synchronizing sound.
- [Making models](making-models.md): construct through the manufactory.
- [Testing](testing.md): choose a focused check before a full render suite.
- [Building](building.md): the project runner, checks and the source shelf.
- [Azazel](azazel.md): the shared render queue and worker.
- [Examples](../EXAMPLES.md): more scenes and application commands.
