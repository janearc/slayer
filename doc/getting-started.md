# Getting started with Slayer

Slayer makes pictures and films by tracing light through a scene. Start
with a supplied scene, change its camera and lighting, and add a soundtrack
without writing Go. Go packages let you build new scenes and extend the
renderer. First follow [Building Slayer](building.md), then run these
commands from the checkout root.

## The pieces of a picture

A **set** describes the world: geometry, materials, lights, volumes and
sky. A **job** describes the photograph or film: scene, camera, size,
samples, time, development, soundtrack and outputs. The **renderer**
computes light; the **darkroom** turns linear light into display colours.
A PNG is developed; a PFM retains linear light; an MP4 holds developed
frames and optionally sound.

Slayer supplies numbered **smokes**, small scenes that demonstrate rendering
capabilities, and **named scenes** built in Go. Use them before authoring
your own set.

## Your first picture

The [first-picture job](examples/jobs/first-picture.yaml) draws smoke 1 at
160 by 90 on the CPU. It needs no Metal device or external terrain.

```sh
game run slayer -- assess doc/examples/jobs/first-picture.yaml
game run slayer -- draw doc/examples/jobs/first-picture.yaml --dry-run
game run slayer -- draw doc/examples/jobs/first-picture.yaml
```

Assessment checks the input without drawing. Dry run prints the resolved
job and output paths. Drawing writes `out/first-picture.png` and
`out/first-picture.pfm`. Open the PNG in your image viewer. Output paths
in these jobs are relative to the job file, not your terminal directory.

Try a larger picture and more samples:

```sh
game run slayer -- draw doc/examples/jobs/first-picture.yaml --width 320 --height 180 --rays 2
```

Rays are counted **per axis**: 2 means four rays per pixel, 4 means
sixteen. Increasing 2 to 4 roughly quadruples tracing work. Samples help
edges and sampling noise; they cannot repair a wrong camera.

## Develop the light again

Copy the job, change its `print` settings, and develop the existing PFM:

```sh
game run slayer -- develop doc/examples/jobs/first-picture.yaml out/first-picture.pfm
```

This uses the job's current development settings without tracing again.
See [print and stock](../job/README.md#print-and-stock). Exposure changes
the display of the light; adding a lamp changes the world itself.

## Your first film and soundtrack

The [first-film job](examples/jobs/first-film.yaml) makes three seconds at six
frames per second, with a synthesized score. The scene is stationary:
this example checks frames, sound and encoding before camera animation.
Movie output needs `ffmpeg` on PATH.

```sh
game run slayer -- draw doc/examples/jobs/first-film.yaml --dry-run
game run slayer -- draw doc/examples/jobs/first-film.yaml
```

It writes `out/first-film.mp4` and numbered PNGs in `out/first-film/`.
The encoder pads this tiny 90-row picture to 96 rows in the movie.
Longer films require more frames. Control-C stops between frames; an
unchanged job, inputs and build can resume completed frames at the same
output location. Read [output and resume](../door/README.md) before moving
or reusing a long render's files.

## Make the scene yours

`game run slayer -- list` discovers scenes, smokes, presets, stocks,
tunes and voicings. The [set guide](sets.md) explains world YAML. In a
job, replace `scene: {smoke: 1}` with `scene: {set: PATH}`; PATH is
relative to that job. Unknown keys are refused at their source location.

The camera looks along negative z with y up. Sets use metres; some
named astronomical scenes declare another unit. A camera pointed away
from its subject can correctly draw empty sky. A laser needs scattering
matter to reveal a beam through space; it is not a painted line.

Choose [a rendering mode](modes.md) before a larger film. `renderer: gpu`
requires supported macOS Metal. `vibes: on` requests lighting shortcuts;
it has tradeoffs and feature restrictions.

## Learn the rest

- [Build](building.md): tools, Game, dependencies and checks.
- [Renderer](renderer.md): geometry, backends, films and gravity.
- [Light](light.md): illumination, shadows, mist and development.
- [Modes](modes.md): careful measurement and fast films.
- [Optimizations](optimizations.md): how each saving works.
- [Music](music.md): choosing, writing and synchronizing sound.
- [Testing](testing.md): choose a focused check before a full render suite.
- [Making models](making-models.md): construct through the manufactory.
- [Azazel](azazel.md): direct rendering and the future queue.
- [Examples](../EXAMPLES.md): more scenes and application commands.
