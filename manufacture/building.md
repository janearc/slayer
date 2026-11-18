# Shared building and spacecraft components

`manufacture.Building` and `manufacture.SpacecraftFrom` interpret the same
ordered component vocabulary and return ordinary opaque manufacture kinds.
Building remains source compatible through its `BuildingForm`, `BuildingPart`,
`BuildingSurface`, and `BuildingMaterial` aliases. Explicit recipes make no
random choices, so their seed does not change the layout. Both constructors
support every detail returned by `manufacture.Complexities()`:

- `BuildingBox`: `Position` is the minimum corner; set `Width`, `Height`, and
  `Depth`.
- `BuildingDrum`: `Position` is the bottom centre; set `Radius`, `InnerRadius`
  (zero makes a solid cylinder), and `Height`.
- `BuildingDome`: `Position` is the base centre; set `Radius`, `Height`, and
  `DomeCutsPerLevel`, which controls profile refinement at each detail.
- `BuildingSlitDome`: `Position` is the base centre; set `Radius`, `Thickness`,
  and the observing `Opening` angle.
- `ComponentRod`: set local `From`, `To`, and `Radius`; `Frame` transforms the
  endpoints.
- `ComponentDish`: set `Position`, `Radius`, and `Depth`; `Smooth` selects
  round normals.

The `Component*` form names are shared by both APIs. `Cuts` explicitly fixes a
round component's radial segmentation, `Smooth` requests smooth round normals,
`Segmented` selects a segmented dish, and a drum `Profile` supplies an explicit
radius/height contour. These choices let a recipe preserve authored geometry
such as an eight-sided bus and a smooth optical tube and primary. Unsupported
combinations and invalid dimensions are rejected by the constructor.

Lengths and the coordinates in `Position` are metres. The envelope is centered
on X/Z, starts at ground level, and must contain every part. Each component
selects one declared `BuildingSurface` material role (`BuildingWalls`,
`BuildingRoof`, or `BuildingAccent`).

The bounded API currently allows at most 512 parts and 10,000 metres per
dimension. Building recipes use the three architectural roles
`BuildingWalls`, `BuildingRoof`, and `BuildingAccent`.

Spacecraft recipes may declare any nonempty, unique `ComponentSurface` roles,
such as housing, optics, solar, or foil. Extra or missing form dimensions,
non-finite values, unknown roles/forms, invalid material response, and parts
outside the envelope are rejected.

Griffith is a real building consumer: its terrace, wings, drums, domes, portico,
columns, and paired tower roofs use the shared interpreter. Its authored
refractor remains a specialized optical assembly.

Hubble and Spitzer use the same interpreter for their shared bus, baffle, and
primary; their named constructors, solar details, authored surface files, and
material provenance remain in place.

Webb keeps its specialized construction where the generic components do not
represent its exact geometry. The tests also construct an unrelated pavilion
and a custom courier spacecraft from the public APIs.

Constructors copy the caller's part/material slices and each drum profile,
plus the top-level `thing.Material` value supplied in `ComponentMaterial.Look`.
Pointers nested inside that material (including texture or optics data) remain
shared; callers must treat those referenced values as immutable while the
recipe uses them.

Component material names, roles, colours, and finish values are retained by
the recipe. This is a Go construction API; it does not add arbitrary component
syntax to the set or scene YAML formats.
