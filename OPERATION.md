# How Slayer works

Slayer renders scenes by following light backwards from the camera. It is
written in Go, with a Metal renderer for supported scenes on macOS.
Physical values should come from a published source or a documented modelling
choice; artistic choices should be identified as such.

This guide follows a render from scene construction to output.
[PACKAGES.md](PACKAGES.md) describes the packages, and
[EXAMPLES.md](EXAMPLES.md) lists scenes and commands.

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
instant, distributes rows across workers, and averages each pixel’s samples.

The sampling argument is the number of rays along each axis: `n` means
`n × n` rays per pixel.

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

Rendering is deterministic for an unchanged world. Row scheduling does not
change a pixel’s arithmetic. Do not mutate shared geometry, materials, or
media during a render.

There are currently two CPU tracers. `trace` contains newer fixes and is the
migration target. The older, independently audited `render/trace` is still
used by `cmd/slayer`, `smokecheck`, `ring`, and `swarm`.

## Rendering on the GPU

On macOS, `gpu.Sampled` renders supported scenes through Apple’s Metal API
and returns the same frame type as the CPU tracer. Kernels compile from
source when opened; building Slayer does not require a separate Metal
compiler toolchain.

The GPU supports a subset of scene features. Unsupported features return an
error identifying them. CPU renders provide the reference for GPU checks;
approximations such as baked haze have explicit test tolerances.
Commands with GPU support expose `-gpu`.

## The darkroom

A `render/frame.Frame` stores RGB light, coverage, and first-hit depth for
each pixel. Light is linear and can exceed one.

`render/colour` applies exposure, bloom, highlight roll-off, and sRGB
encoding. `trace.RenderIn` renders a spectral band; `colour.False` can
combine those images with a visible-light render using assigned colours.

The CPU and GPU band paths have different capabilities; see
[doc/bands.md](doc/bands.md).

`ninjafilm` models photographic stocks, including spectral sensitivity and
grain, and provides lenses with physical focal lengths. Commands currently
use its lenses and frames; the stock simulation is tested separately and
is not yet connected to a command.

## Output formats

- `interfaces/file` writes PNG images.
- `raw` reads and writes linear RGB in PFM files. `develop` can adjust a raw
  frame or sum several frames into a long exposure.
- `interfaces/terminal` represents each terminal cell as six subpixels in
  two colours.
- `interfaces/sheet` arranges labelled images into contact sheets.
- `interfaces/learn` approximates terminal output as a raster image.
- `hud` overlays text, frames, bars, and labels.

## Sources and reproducibility

`sources.yaml` records published datasets, their origins, licences, and
SHA-256 hashes. The `sources` command downloads them to a shelf outside
the repository and checks their bytes:

```sh
SLAYER_SHELF=/path/to/shelf go run ./cmd/sources fetch
SLAYER_SHELF=/path/to/shelf go run ./cmd/sources check
```

Changed source bytes fail verification. `research` exposes named values
from the project’s research data, rather than requiring callers to copy
numbers from memory.

## Running Slayer

Run commands from the checkout root so relative defaults resolve correctly.
Images go to `out/` unless an output flag specifies another location.

```sh
go run ./cmd/swarm
go run ./cmd/orbital -far
go run ./cmd/hillside -night
go run ./cmd/smokecheck smoke 1
```

See [EXAMPLES.md](EXAMPLES.md) for scene descriptions and command options.

## Checking the renderer

Slayer began beside Lamb, an earlier tracer. Its checks were designed to
establish correctness independently of the implementation.

- **Smoke scenes** record successive rendering capabilities. Their numbers
  remain stable, and a change to an existing smoke is a separate commit.
- **burning-trousers** compares exported renderer APIs with closed-form
  physics and independently computed expectations. A commit must not
  change both the harness and the renderer it checks.
- **fermat** selects 64 pixels from a hash of a raw image’s bytes and
  replays their computation. Each must match bit for bit. Run it through
  `hillside -prove` with the original rendering parameters.
- **Validation** reports unsupported or invalid inputs at construction
  boundaries instead of silently rendering an approximation.
- **Mutation testing** changes individual operations and runs the tests.
  A surviving mutation identifies an untested distinction or a case whose
  equivalence needs an explanation.

## Viewing the smokes

`smokecheck` renders each scene at terminal resolution, 179 by 50 cells,
and as a finer raster image. It checks that the terminal view agrees with
the raster image at the coarser resolution.

```sh
go run ./cmd/smokecheck smoke
go run ./cmd/smokecheck smoke 7 -out /tmp/smoke7
```

A cache keyed by the rendering binary avoids repeating unchanged work.
The uncommitted `smokes/` directory contains images, timing records in
`results.jsonl`, and an `index.html` gallery. `lightbox` serves it at
<http://localhost:8765>.

## Further reading

- [Sets](doc/sets.md): what a set file says.
- [Performance](doc/fast.md): rendering optimisations and planned stages.
- [Spectral bands](doc/bands.md): band rendering and CPU/GPU differences.
- [Takes](doc/take.md): camera paths and shutter sampling.
- [The lantern](doc/the-lantern.md) and
  [the orbital scene](doc/the-orbital-scene.md): scene design and history.
- [Harness rules](burning-trousers/INSIDE.md): independent validation.
- [Technical debt](DEBT.md): outstanding style migration.
