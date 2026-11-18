# Authored anatomy and small populations

Heard scenes can opt into composable geometry with `stage.anatomy` on an
`individual` stage, or a small authored population with
`stage.population` on an `authored-multitude` stage. Omitting `anatomy` from
the existing `individual` stage keeps its seeded historical constructor.

The historical `kind: multitude` scene and the roughly 3,000-being v9 finale
remain unchanged; this small authored path does not replace that finale or
remove the project's overall complexity limits.

## Anatomy fields

An anatomy contains any non-empty combination of `cores`, `bands`, and
`sails`. Geometry lengths use the scene's metres. Every element can have a
local `at` offset, `rotation-degrees` Euler rotation, `spin-radians-per-second`,
linear RGB `colour`, and RGB `glow` (emission).

- A core has `form` (`sphere` or `octahedron`) and `radius`.
- A band has `form` (`annular` or `undulating`), `count`, `segments`,
  centreline `radius`, cross-section `width` and `thickness`, and radial
  `spacing` between repeated bands. Optional `eccentricity` and `warp-share`
  alter the ring shape. `undulating` adds a vertical wave; `waves` and
  `phase-degrees` control its cycle and offset.
- A sail has `count`, `segments`, `span-segments`, `length`, and `width`.
  `sweep` is a share of length, `camber` a share of width, and
  `twist-degrees` twists the sheet along its length. Sails are open sheets;
  bands are closed surfaces.

Each geometry component is an independent choice, so an anatomy can mix
different solid bodies, closed ring forms, and open sails. Repeated identical
mesh geometry may be shared while placement, spin, colour, and glow remain
independent.

## Population fields and limits

An authored `population` has 2 to 32 members. Each member supplies an
`identity` (stable seed for its motion), world `at` offset, `scale` from 0.01
to 20, and its own `anatomy`. The stage uses its authored `eye`, `end-eye`,
`look`, `wide`, and `exposure` camera fields.

The authoring path caps the whole population at **32 beings, 64 geometry
components, and 65,536 estimated faces**. A single `individual` anatomy has
the same component and face ceilings. These are deliberate bounds on
allocation and construction work for this opt-in authored route.

Mesh segments, extents, light values, and motion rates are also validated
before geometry is built. Sharing repeated immutable meshes reduces stored
geometry for matching forms, but does not bypass the instance, component, or
face ceilings.

