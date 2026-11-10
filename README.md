# Slayer

## pre-released

this is my third attempt at a ray tracing/rendering kit. it is my first with
metal support, and it has the most ambitious efforts at math and physics. i
use this software constantly, but i do not consider it "released" yet and i
plan to change a lot about it.

if you'd like help with using it or you have questions or patches, let me
know. please expect that this will change substantially for a little while.

## reign of bl--i mean, "what is this?"

A Go ray tracer for scenes ranging from a hillside at moonrise to a star
inside a Dyson swarm. Slayer combines procedural geometry, volumetric
light, astronomical scenes, and a photographic darkroom. Supported scenes
can also render through Metal on macOS.

The renderer keeps linear light separate from development. Save a raw
frame to revisit its exposure and white balance without tracing it again.
Measured inputs carry sources; fictional dimensions and visual tuning
should be identified as choices.

## Start here

- [How Slayer works](OPERATION.md): scenes, tracing, development, and output.
- [Game workflow](doc/developing.md): setup, builds, runs and checks.
- [Scenes and commands](EXAMPLES.md): smoke scenes and larger examples.
- [Feature backlog](doc/feature-backlog.md): current plans, partial work,
  and the next camera, demo and scenery features.
- [One-command demos](doc/demos.md): a numbered smoke, an orbital,
  or a moving-camera preview, with PNG and terminal output.
- [Camera motion](doc/camera-motion.md): YAML moves, speeds and holds
  for repeatable frame sequences.
- [Package guide](PACKAGES.md): responsibilities and dependency boundaries.
- [Manufactory catalogue](doc/catalogue.md): the kinds we can make,
  their variations, and how they reach the GPU.
- [Fire recipes](doc/fire-recipes.md): standalone flames, source layouts,
  optional smoke, and the assumptions behind their inputs.
- [Surface fuel study](doc/fire-fuel-design.md): a GPU heat/fuel prototype
  and its 720p overhead scene, with no timed ignition front.
- [Wooded slope and moon](doc/fire-woodland-result.md): actual redwoods,
  surface fire, a measured hillside and short native GPU clips.
- [Running Grissom](doc/grissom-setup.md): the graphics viewer's workspace,
  survey files, Earth data, colourways, and first render.
- [ninjafilm](ninjafilm/README.md): photographic stocks, optics, and grain.
- [Independent validation](burning-trousers/README.md): physics checks and
  the rules keeping the harness separate from its renderer.

Configure the source shelf as described in the [Game workflow](doc/developing.md),
then run these from the checkout root:

```sh
game run sources-fetch
game build
game run hillside
game run smoke-one
```

Use the Go version specified by `go.mod`. The named runs are declared in
`.game`. The [examples](EXAMPLES.md) also show application flags, which
need Game's argument-forwarding fix. Individual scenes may need additional
data before rendering.

## Reading the design notes

`doc/` contains both implemented interfaces and historical plans. Each
rewritten design note identifies that distinction; proposed syntax and
archived commands are not promises of current API support. Package
exports and current command help define the live interface.

[DEBT.md](DEBT.md) tracks the remaining style migration. Development uses
Game; harness changes belong in their own commits, as described in
[burning-trousers/INSIDE.md](burning-trousers/INSIDE.md).

The [night-fire study](doc/fire-night-design.md) builds a moonlit grove with
firebrands and a prescribed ignition front. It includes a checked YAML scene
and a resumable Pzm sequence plan, with the model choices and sources beside
them.
