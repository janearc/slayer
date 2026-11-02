# The manufactory: objects grown from seeds

A hillside contains trees, ferns, stones, fallen logs, discarded cans,
and perhaps a fire after dark. The manufactory builds those objects from
a kind and a seed rather than asking callers to place every triangle.
The seed supplies variation within the kind’s declared ranges.

This document records the design of 2026-10-06, including its implementation
sequence and original open questions. Later work added kinds and revised
some boundaries; [families.md](families.md) continues the tree design.
Treat historical steps as design history, not a complete current API list.

## Shared parts instead of a catalogue of algorithms

The earlier `manufacture/` exposed separate functions such as `Ottoman`,
`Sofa`, `Fern`, `Lantern`, and `OrganicTree`, each accepting hand-set
parameters and returning triangles. The replacement reuses their useful parts:

- `BranchOf` in `conifer_proper.go` computes a branch from whorl, index,
  and seed without recursion or stored triangles.
- `orientedTube` connects two points with a tapered tube.
- The lump in ninja-gaiden’s `lesson/debris.go.txt` perturbs a sphere
  with seeded noise and cleaves it with planes.
- `unit(seed, a, b, c)` gives a reproducible draw independent of the machine.

A common construction path connects those primitives. A new kind supplies
data unless it needs a genuinely new growth rule.

## Construction boundary

Inputs are validated by constructors; resulting values have private
representation and public methods. Every dimensional quantity is typed
through `measure`.

A deliberate exception is a table row’s range, `span`: its bare endpoints
use the unit named by that row—metres, degrees, kelvin, or watts. Once a
seed selects a value, it becomes a typed quantity.

The proposed interface expresses the idea:

```go
func Sow(number uint64) Seed
func Make(kind Kind, seed Seed) (Good, error)
```

The actual public surface is `manufacture/offered.go`; several types and
constructors shown here remained private. Use those declarations rather
than treating this design sketch as copyable client code.

A seed is only a random seed. An age-based growth model was removed on
2026-10-06. A kind names the ranges for dimensions, counts, habit, and
appearance. The same kind and seed must generate the same object.

## Dependencies

The design uses `measure` for quantities, libtheme bands for spectra,
`forest` for ground and placements, `maths` for transforms, `scene/thing`
for scene objects, and `mesh` for triangle surfaces.

The manufactory does not import `emit`. It describes attached lights;
`emit` reads those descriptions. Keeping that dependency one-way is part
of the contract, tested by inspecting imports.

## Primitive geometry

### Seeded draws

`draw(seed, a, b, c)` retains the original deterministic draw. It reads
neither a clock nor a shared generator, so evaluation order does not
change an object.

### Forms

A form describes a surface before tessellation:

- **Tube:** a straight segment between base and tip rings, with closed
  ends. A zero tip radius makes a cone. Bent objects join several tubes.
- **Blade:** a flat tube with two corners per ring, represented by a
  strip visible from either side. It covers grass, needles, and leaflets.
- **Lump:** a triangulated sphere perturbed by seeded noise, stretched
  along three axes, cleaved, and rested on its flattest side.

```go
func Tube(length, foot, tip measure.Length) (Form, error)
func Blade(length, foot, tip measure.Length) (Form, error)
func Lump(long, middle, short measure.Length, worn float64) (Form, error)
```

`worn` runs from zero for newly broken stone to one for river-worn stone.
A shared rule derives cleaving and noise settings. Its parameters are
identified as tuning, not measurements. Tube roughening for bark was left
open in the original plan.

### Arrangements and composition

An arrangement maps index and seed to position, orientation, and scale:

- **Whorls:** tiers up a stem, rotated between tiers. Spruce uses the
  golden angle; paired fern leaflets use a half-turn.
- **Ring:** attachments around a point, such as fire stones and logs.
- **Scatter:** placements over ground, already provided by `forest.Placements`.

```go
func Carry(stem Form, places Arrangement, part Form) (Form, error)
```

A tree is a trunk carrying branch tubes, which carry foliage sleeves.
A fern has fronds carrying paired blades. A log carries broken stubs.
These compositions avoid a separate geometric representation per species.

## Generated objects

A `good` contains parts in local space, each with a form and material
appearance. A stone may have one part, a spruce hundreds, and a campfire
may combine stones and logs with an attached light.

`Things(place)` supplies scene objects, grouped into meshes by material
and wound outward. `Lights` supplies attached light descriptions.
World coordinates use y up. Tests rotate older z-up Ninja geometry by a
quarter-turn rather than reflecting it, preserving face winding.

`Volume`, `Mass`, and `Called` were removed from the proposed object API on
2026-10-06. The remaining material rows describe rendering appearance.

## Attached lights

A campfire, spotlight, and maser all describe a spectrum, power, and
angular spread. The manufactory stores that description; `emit` converts
it to a renderer lamp.

```go
func Thermal(heat measure.Temperature) (Spectrum, error)
func Line(wavelength measure.Length) (Spectrum, error)
func Everyway() Spread
func Cone(half, soft measure.Angle) (Spread, error)
func Aperture(across measure.Length) (Spread, error)
func NewLight(colour Spectrum, power measure.Power,
    spread Spread) (Light, error)
func (made Light) Flickering(seed Seed, depth float64) (Light, error)
```

These are design names; consult the current exports before calling them.

`Thermal` supplies a black-body spectrum; `Line` supplies one wavelength.
`Everyway` is isotropic. `Cone` describes a beam with a soft edge.
`Aperture` requires a line spectrum and derives the diffraction angle as
`1.22 × wavelength / aperture`; a thermal spectrum is rejected.

Directional strength is power divided by the spread’s solid angle. A
100 W spotlight therefore has greater beam intensity than a 100 W bare
lamp without a separate brightness multiplier.

`Flickering` stores a seed and depth, rejecting unsown seeds and values
outside a fraction’s range. Its time-varying strength is evaluated once
per frame on the CPU, following the wander design in [emitters.md](emitters.md).

The description exposes a libtheme spectrum, intensity, beam half-angle,
soft edge, and time-dependent strength. An all-direction spread uses a
half-turn and zero soft edge. A kind row places its lights relative to
the object, so placing the object also moves its lights.

## Kinds and parameter provenance

All kinds use the same construction path, for example:

```go
spruce, err := manufacture.Make(manufacture.Spruce, manufacture.Sow(11))
fern, err := manufacture.Make(manufacture.Fern, manufacture.Sow(12))
boulder, err := manufacture.Make(manufacture.Boulder, manufacture.Sow(7))
```

The examples express the original design. Name lookup is exposed through
`manufacture.KindNamed("boulder")`; check current exports for typed calls.
Other original kinds are cobble, pebble, log, can, campfire, spotlight,
and maser.

The 2026-10-06 decision was to invent plausible dimensions and colours,
rather than claim a measured botanical or material model. Preserve that
distinction beside the declarations.

- Derived quantities stay derived: the golden angle `π(3 − √5)`, paired
  leaflet turns, aperture diffraction, and intensity from solid angle.
- Chosen values—height ranges, flame temperature, beam width, granite
  colour—are identified as fictional or visual choices.
- Three values were recalled rather than verified: a 211 × 413 can,
  2 11/16 inches across and 4 13/16 inches high; stone class boundaries
  at 4, 64, and 256 mm; and hydrogen’s 1420.4 MHz, approximately 21 cm line.

Rows live in Go tables: `kinds.go` for solids and `lamps.go` for light-bearing
kinds. A runtime table loader was not needed by the original callers.

## CPU and GPU paths

Meshes use the ordinary `thing.Thing` path and GPU validation. Large forests
need a different representation: the recorded cost of 64 flattened conifers
was 1.6 GB. Analytic tubes can describe trunks, limbs, logs, and cans without
storing every triangle. That engine work is separate from the manufactory.

The original plan targeted point lights first, then cone lamps. Steps 11
and 12 added `emit.Of` and cone-limited points, followed by the kernel cone
lamp, checked against the CPU. Earlier point-only limitations in historical
notes refer to the implementation before those steps.

A maser emits nothing in a visible render. Render its own band to see it;
[bands.md](bands.md) explains the band-path differences.

## Independent checks

Tests compare against references not generated by the implementation:

- Tube volume approaches the frustum formula as tessellation increases.
- Tubes match Ninja’s `orientedTube` triangle for triangle; whorls match
  `BranchOf`; a spruce with the original parameters matches `ConiferProper`.
- Lumps are closed, outward-facing, bounded by their axes, rested on the
  ground, reproducible for one seed, and variable across seeds.
- Integrated light intensity agrees with power; aperture follows diffraction.
- Forty seeds per kind produce valid objects within declared ranges.
- Fern leaflets remain in two rows in each frond’s plane.
- Invalid constructor inputs and invalid rows produce the expected errors.
- A hillside smoke contains all solid kinds and renders a nonblank image.

In-package renders use the CPU tracer because `gpu` imports `manufacture`;
a reverse import in tests would cycle. Command tests cover GPU output and
lights. Run mutations over changed Go files before completing a stage.

## Original delivery sequence

Each stage was planned as a separate commit with tests first and gate checks:

1. Seed and deterministic draws.
2. Tube and blade primitives.
3. Whorl and ring arrangements.
4. Composition, original-conifer agreement, and fern.
5. Lump geometry.
6. Generated objects and scene conversion.
7. Kind rows and `Make` for the initial solids.
8. Spectra, spreads, light descriptions, and flicker.
9. Gathered objects carrying lights: campfire, spotlight, and maser.
10. Public client API and `cmd/hillside`, using seeds, counts, and forest
    placements. Flat-mesh budgets prevent uncontrolled forest allocation.
11. `emit` reads descriptions and provides a one-beam light. Implemented
    through cone-limited points and `emit.Of`.
12. GPU cone lighting checked against the CPU. Recorded as implemented.

The first nine stages were marked built in the original notes. Geometry
constructors shown with capital letters in this design remained private
unless an external caller needed them; `offered.go` is authoritative.

## Migration and original open questions

The initial migration kept `Cylinder`, `Cone`, `Conifer`, `ConiferProper`,
and `BranchOf` as reference implementations.

It did not carry over the
registry/encoder path (`kind.go`, `encoder.go`, `params.go`, `scene.go`,
and `manufacturepaths.go`), because `gpu/structure.go` already defined
and packed the GPU representation.

Older clock-seeded fern placement was replaced by a seeded kind. Furniture
and fittings were outside the 2026-10-06 scope and remained on
`slayer-grav-mfg`; the later 2026-10-17 scope includes homes.

The original outstanding items were growth parameters still embedded in
`grow.go`, the name `good`, scene exposure calibration, collision-aware
placements, leaflets along curved fronds, and bark roughening. The tree
parameter work continues in [families.md](families.md).

`cmd/hillside` used an exposure conversion of 2000, chosen by inspecting a
night campfire. It is a scene choice, not a measured conversion from watts
to RGB light. Ground scattering also originally ignored object overlap,
so a boulder could land on a fire.
