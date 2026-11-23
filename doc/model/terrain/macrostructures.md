# Macrostructures and large terrain features

Five independent manufactory recipes add kilometre-scale scenery without
changing any existing shot. All dimensions, proportions, colours, structural
interpretations and geological profiles are authored choices. These are
mathematical scenery constructors, not engineering or geological simulations.
Their named numeric records are in
[macro-choices.yaml](../../../manufacture/macro-choices.yaml).

| Kind | Default physical envelope | Quick faces |
| --- | --- | --- |
| `macro-terraced-arcology` | 1,800 × 1,500 m footprint, 2,200 m tall | 900 |
| `macro-twisted-spires` | 1,500 × 1,500 m footprint, 4,000 m tall | 4,236 |
| `macro-inhabited-bridge` | 6,000 × 1,200 m footprint, 1,500 m tall | 1,084 |
| `landform-mountain-massif` | 5,000 m radius, 2,500 m summit | 1,152 |
| `landform-caldera-volcano` | 7,000 m radius, 3,500 m rim | 1,344 |

The [example set](../../macrostructures.yaml) has an overview and separate
`arcology`, `spires`, `bridge`, `mountain` and `volcano` cameras. The
[still job](../../examples/jobs/macrostructures.yaml) uses the normal Slayer front door:

```sh
game build
game run slayer -- draw doc/examples/jobs/macrostructures.yaml
game run slayer -- draw doc/examples/jobs/macrostructures.yaml --camera spires
```

Choose `--out` to retain separate views. Set camera and world coordinates are
east, north, up. The Go model uses x east, y up and z south, with its base on
y=0. The example uses one distant sun. No recipe generates attached lamps;
caldera lava is an authored emissive surface, not a source-light simulation.

`hull`, `quick` and `demo` preserve physical envelopes. Loft, arch and terrain
subdivisions are 6, 12 and 24; terrain uses four angular segments per radial
subdivision. The arcology's box/gallery rule needs no curved refinement.
Unsupported `clumps` is refused. Every specimen has a 20,000-face budget.

Architecture batches through the existing `good.Things` path into at most
four material meshes. A terrain feature uses two material meshes: rock and
snow, or rock and lava. No independent acceleration structure is created for
every rib, support, floor, tree-like feature or window.

## Checked dimensional constructors

`DefaultMacrostructure(style)` supplies the authored dimensions. Change them
before calling `Macrostructure(spec)`; no whole-character mesh is stretched
in place of regenerating its architecture. This function-body fragment
assumes an existing `world`:

```go
spec := manufacture.DefaultMacrostructure(manufacture.TwistedSpires)
spec.Width = measure.Metres(120000)
spec.Depth = measure.Metres(120000)
spec.Height = measure.Metres(400000)
spec.Twist = measure.Degrees(150)
recipe, err := manufacture.Macrostructure(spec)
if err != nil { return err }
made, err := manufacture.Make(recipe, manufacture.Sow(31))
if err != nil { return err }
place := maths.NewSubspace(at, orientation, 1e-6)
parts, err := made.Things(place)
if err != nil { return err }
for _, part := range parts { world.Add(part) }
```

Import `manufacture` and `maths` from Slayer, and `measure` from Celestial.
`at` and `orientation` belong to the consumer's surface placement. For an
orbital whose scene unit is one million metres, use exactly `1e-6` as above:
this example is physically 400 km tall and becomes 0.4 scene units tall.

The default 4 km spire becomes 0.004 scene units; visibility does not justify
silently relabelling those metres. Placement and composition remain with the
orbital adapter.

Macrostructure constructors admit finite dimensions from 1 m through 1,000,000,000 m, with
an aspect limit of 1,000. A spire cluster needs width/depth between 0.5 and 2,
and height at least half its larger footprint dimension. Bridge span must be
at least twice its depth and height. Twist is limited to ±180 degrees.
These bounds keep these particular construction rules meaningful; they are
not physical feasibility claims.

`Landform(DefaultLandform(MountainMassif))` and
`Landform(DefaultLandform(CalderaVolcano))` use `LandformSpec`. Its `Radius`,
`Height`, `CraterRadius` and `CraterDepth` are typed lengths. Caldera radius
must be below 70% of the outside radius and depth below 80% of the summit
height; mountain recipes refuse crater dimensions.

`LandformElevation(spec, Sow(seed), east, south)` returns a typed height from
the same analytic profile used at the mesh vertices, and zero beyond its
radius. Coordinates are signed metre offsets. Rendered triangles interpolate
between sampled vertices, so an exact attachment to a coarse representation
may require intersecting that representation rather than using the analytic
height alone.

## Construction formulae

An arcology has eight rings. With `t=i/7`, their outside dimensions are
`W_i=W(1−0.55t)` and `D_i=D(1−0.55t)`. The through courtyard stays
`0.18W × 0.18D`. Most occupied bands fill 72% of the `H/8` pitch, leaving
open galleries; the last fills its pitch to attain the requested height.

Four supports bridge every gallery gap within overlapping floor rings.
Planted rings top the lower terraces. Small seeded offsets grow with height,
producing asymmetry while keeping the courtyard and footprint clear.

The spire cluster has three unequal hollow rectangular lofts. Section width
and depth contract by `s(t)=1−0.7t`, and rotate by `θ(t)=φ+αt`. Each local
corner transforms as `x'=s(x cosθ−z sinθ)`, `z'=s(x sinθ+z cosθ)`, with
`y=t h`. The inner rectangular opening is half the outside section. Helical
ribs follow these same corner stations.

The tallest reaches H; lesser heights and phases vary by seed. Braced galleries and sky parks connect the
towers at 0.34H and 0.50H above a grounded foundation.

The bridge has two continuous swept arches, with span `S=0.9W` and
`y(x)=r+A[1−(2x/S)²]`. Here `r=min(0.035D, 0.015H)` and the seed chooses
`A` between 87% and 95% of the 0.65H deck elevation. Ground-bearing ends,
footings and vertical spandrels support the inhabited courtyard galleries.
The swept arch shares rings rather than overlapping capped segments.
A planted deck and an unequal end pavilion break its bilateral outline.

For a mountain, let `u=r/R`, exponent `p=1.35` and roughness `ε=0.2`:

`y = H(1−u)^p [1−εu(1−u)(1+cos(kθ+φ))]`.

The seed selects ridge count k and phase φ. The summit stays H and the base
stays zero. For a caldera of rim radius c and depth d, its inside profile is
`y=H−d+d(r/c)²`; outside the rim, use the mountain formula with
`u=(r−c)/(R−c)`. This retains the exact rim, floor and grounded base across
seeds and tessellation levels. Snow/lava assignment is authored scenery.

## Existing hexes and biome capabilities

`maths/hexes` provides stable IDs, neighbours, refinement, positions and
surface distances for Sphere, Band and Flat. It does not generate mountains,
volcanoes or architecture. `biome` already retains downhill river/lake paths,
peak volcano tiles, water-adjacent cities and classified rock/snow/ash tiles.
Those features previously supplied only paints, glow and aggregate counts.

`(*biome.World).Features()` now returns owned records in stable tiling order:
`Tile`, `At`, `HeightAboveSea`, and `River`, `Lake`, `Volcano`, `City`,
`Highland` flags. It reads those existing maps; generation, Paint and Shine
are unchanged. Highland means the existing rock/snow/ash classification,
not a surveyed mountain peak. River records count path tiles, while the
existing Rivers counter counts source walks.

`At` and `HeightAboveSea` are not metres. Sphere positions are directions;
Band/Flat use their own surface coordinates. Height is the dimensionless
height field minus sea level. A consumer chooses physical relief, aligns a
recipe to the existing surface frame and converts metres to its scene units.
The accessor does not introduce another terrain or planning framework.


## Ancient living variations and orbital geography

`MacrostructureSpec.Wildness` is optional, from zero through one. Zero
preserves the original recipe. A continuous twist/sway passes through every
structural part, with shared-edge refinement before warping. Its bounded
horizontal map retains the requested footprint and height. `LivingCover`
adds native branching growth rooted on the generated surface, then carried
through the same warp. The growth is authored giant fantasy biology.

Seeded mineral and living patina vary over several physical scales. Both
controls retain four native material batches. Original quick geometry has
its 20,000-face budget; warped/refined geometry has a 100,000-face budget,
including growth. It is checked as native closed geometry before drawing.

`LandformSpec.Ruggedness` optionally adds finer radial ridges, crater
terraces and mineral strata. Zero preserves the original profile. Detailed
relief keeps the base, summit/rim and crater endpoints while increasing
radial stations. The new 600 km-wide, 24 km-high shield dimensions are
authored visual choices inspired by planetary relief, not a survey model.

The named orbital accepts `surface-structures` and `surface-landforms`
counts per ring, both zero by default. It consumes existing biome metadata:
structures stand in cities and landforms in highlands, with retained
volcano records selecting calderas. Structures regenerate at approximately
100 to 500 km high. Material mesh vertices map onto the actual polygonal
floor and share its rotation; textures map back into physical metres.

`surface-style` selects `mixed`, `terraced-arcology`, `twisted-spires` or
`inhabited-bridge`. `surface-close` carries the camera with the first site;
otherwise the geographic view is retained.

Thin high-aspect glazing gets four refinement rounds before warping instead
of two: coarse long closed strips could cross their opposite skin under the
nonlinear bend.
