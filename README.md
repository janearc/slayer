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
- [Scenes and commands](EXAMPLES.md): smoke scenes and larger examples.
- [Package guide](PACKAGES.md): responsibilities and dependency boundaries.
- [ninjafilm](ninjafilm/README.md): photographic stocks, optics, and grain.
- [Independent validation](burning-trousers/README.md): physics checks and
  the rules keeping the harness separate from its renderer.

Run examples from the checkout root:

```sh
go run ./cmd/hillside
go run ./cmd/swarm
go run ./cmd/smokecheck smoke 1
```

Use the Go version specified by `go.mod`. Published datasets are fetched
to the external `SLAYER_SHELF`; the operating guide explains setup and
verification. Individual scenes may need those inputs before rendering.

## Reading the design notes

`doc/` contains both implemented interfaces and historical plans. Each
rewritten design note identifies that distinction; proposed syntax and
archived commands are not promises of current API support. Package
exports and current command help define the live interface.

[DEBT.md](DEBT.md) tracks the remaining style migration. Development uses
Game; harness changes belong in their own commits, as described in
[burning-trousers/INSIDE.md](burning-trousers/INSIDE.md).
