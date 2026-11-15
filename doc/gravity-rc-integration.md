# Gravity integration with the release candidate

This merge brings rc-v0.1.2 at b3d592f8 into the gravity and sampler
prototype at 2c34ff9. It keeps the release candidate's native traversal,
height fields, retained frames, gas optics, ring and wake encoders, and
test tiers, alongside the prototype's software gravity path.

`gpu.NewRenderer` retains its volume-grid cap and immutable observer-seed
cache. `gpu.Sampled` and the renderer's `Sampled` method support one
placed Kerr field. Native traversal and retained-frame rendering refuse
fields; refraction, gas volumes and cached volume-sky lighting are also
refused by the gravity path before device setup.

The gravity kernel takes all seventeen ordinary scene buffers, followed
by its parameters at binding 17, output at 18 and band origin at 19.
Its result records follow the release candidate's expanded layout.

The field's own LetGo setting overrides the default stopping radius.
Every volume encoder initializes the additional gravity metadata.
Curved-segment intersections use the release candidate's closest-hit
refinements and discard hits beyond the segment's end.

Celestial v0.0.9 has no Metal library, and Slayer carries it. The shader
is `metal/geodesic.metal`, embedded as `metal.GeodesicLibrary`. The two
march numbers it needs are beside it, `metal.GeodesicTimeShare` and
`metal.GeodesicHoverRatio`. `go.mod` requires Celestial v0.0.9 as
released and replaces nothing.

Until 2026-11-14 this merge built against a local copy of a Celestial
review candidate that added the shader to Celestial's `hole` package.
[The dependency patch](../dependencies/patches/celestial-metal.patch)
holds what that copy added. It is kept as a record and is applied to
nothing now.

Prototype GPU tests carry the `draw` tag. The flame image comparison
also belongs in the draw tier; its CPU-only density tests stay ordinary.
The old noise comparison tests use the release candidate's independent
reference helpers, preserving their assertions without duplicate helpers.

Validation uses a focused GPU sample, as asked; the full draw
run was stopped because the complete suites had run that morning.
All 28 selected functions pass, including moving gravity, seed caches,
planes and caps, native and retained rendering, refraction, rings, wakes,
sun discs and the ordinary hall. Seed-cache race checks also pass.

With `-coverpkg=./gpu,./trace`, the sample covers all Go statements in
`gpu/field.go` (117/117), `gpu/field_cache.go` (22/22), and
`gpu/renderer.go` (16/16). This is a statement-coverage measurement,
not coverage of the Metal shader or the whole project.

The optics refusals and the per-field stopping assertion failed before
their fixes and pass afterward. Nine compiled counterexamples are

rejected: three merge guards, four seed-cache cases and two noise cases.
The added Celestial pacing test and focused vet pass. The inventory has
6,269 names, preserving all 6,231 existing names; its five changed
packages have no tier problems. The history gate is checked after the
merge commit, when the incoming RC ancestors are visible from HEAD.
