# Object constructors and scene entry points

This is the map into Slayer's current constructors. For every registered
manufactory kind, [the catalogue menu](../cmd/catalogue/menu.yaml) records
its name, group, description, limitations and advertised detail levels.

Read the menu directly or run `game run catalogue-list` to see the current
names and descriptions.

`game run model-complexity` constructs each catalogue kind at each
advertised detail, seed 1, and emits a JSON line per representation.
Pass one or more kind names after `--` to measure a subset. A failing
representation still gets a row and the command exits nonzero. These are
construction and owned-geometry measurements, not a render, GPU memory
measurement or peak working set. See [model resources](model-resources.md).

`game run model-complexity-v9` is an opt-in, one-iteration construction
benchmark for the historical 3,067-being finale. It reports construction time
and Go allocations separately from owned/stored/reference geometry counts.
It does not render, but constructing the full scene can use substantial
memory; schedule it with the shared machine's current workload in mind.

## Shared construction and placement

The [public manufacture API](../manufacture/offered.go) is the door to
registered kinds:

| Constructor or operation | What it makes or changes |
| --- | --- |
| `KindNames()`, `KindNamed(name)` | List or obtain a registered recipe. The catalogue above documents each name. |
| `Sow(number)` | A deterministic construction seed, not an age. |
| `Make(kind, seed)` | A specimen at the kind's default detail. |
| `MakeWithComplexity(kind, seed, detail)` | The same seeded recipe at a supported representation; invalid or unsupported detail is refused. `Complexities()` lists the available names. |
| `Branched(kind)` | A whorled-tree recipe whose limbs carry branches; other kinds are refused. |
| `Taller(kind, factor)` | A whorled tree grown taller with proportionate branching, rather than uniformly scaling every needle. |
| `WithSeason(kind, season)` | A supported vegetation recipe with a seasonal appearance; `SeasonNames()` lists the choices. |
| `specimen.Prepare()` | Constructs and retains immutable local geometry for repeated placements; inspect `Resources()` and place with `Things(at)`. |

A specimen's `Things(at)` and `Lights(at)` place its surfaces and
attached light descriptions. `PlaceLights(descriptions, at)` places
retained lights independently. These are placements, not different
geometry recipes. [Making models](making-models.md) has a complete use
example and explains units, materials and provenance.

## Configurable factory recipes

Each constructor validates its spec before returning a `Kind`. Then use
`Sow` and `Make` or `MakeWithComplexity` as above.

| Constructor | Inputs and result |
| --- | --- |
| [Observatory](../manufacture/observatory.go) | `Optical` selects enclosure diameter, aperture, slit opening, pointing and a segmented mirror. It makes authored scenery, not a calibrated optical model. |
| [Spacecraft](../manufacture/spacecraft.go) | Selects a listed `SpacecraftDesigns()` template and an aperture; proportions scale with that aperture. |
| [`SpacecraftFrom`](../manufacture/building.md) | `SpacecraftSpec` composes a bounded craft from shared `ComponentSpec` parts and named material roles. Use this for custom recipes; `Spacecraft` above remains the named-template constructor. |
| [`Building`](../manufacture/building.md) | `BuildingSpec` composes checked components with explicit dimensions, materials and a bounding envelope. Griffith's shell uses the same interpreter as configurable spacecraft. |
| [City](../manufacture/city.go) | `CityGrid` specifies block and road dimensions, lots, setbacks and junction spacing. `DefaultCityGrid()` supplies a starting recipe. |
| [ClusteredCity](../manufacture/city_districts.go) | `ClusteredCitySpec` combines a city grid, weighted districts, parks, elevated highways, optional trees and night lights. `DefaultClusteredCity()` and `DefaultCityTrees()` provide starting specs. |
| [Macrostructure](../manufacture/macrostructure.go) | `MacrostructureSpec` chooses terraced arcology, twisted spires or inhabited bridge, with footprint, height, twist, wildness and optional growth. `DefaultMacrostructure(style)` supplies a starting spec. |
| [Landform](../manufacture/landform.go) | `LandformSpec` chooses an authored mountain or caldera and its dimensions and ruggedness. `DefaultLandform(style)` supplies a starting spec; `LandformElevation` samples the same checked seeded profile. |
| [Burn](../manufacture/burner.go) | `BurnSpec` makes a checked volumetric flame, optional smoke and sparks, separate from fuel geometry. `Place` creates the time-aware volumes and light sources. |
| [Cavitation cloud](../manufacture/cavitation_cloud.go) | `NewCavitationCloud(config, pose)` returns seeded, bounded map-space analytic gas spheres. Speed and authored duration set trail extent; rise, spread, count and radius are explicit controls. This is a geometry proxy, not solved cavitation. See [the Alfa cloud guide](alfa-cavitation-cloud.md); `game run cavitation-cloud` emits a proof-scene YAML fragment and `game run check-cavitation-cloud` checks the constructor and generator. |
| [Clear cloud medium](../scene/medium/clear_cloud.go) | `NewClearCloud(field)` opts a finite density field into mixed named-water transport while inheriting the legacy water phase. `NewClearCloudWithHenyeyGreenstein(field, g)` selects an independent authored cloud HG phase, finite `-1 < g < 1`; explicit zero is isotropic, not Rayleigh. Density, bounds and clock remain those of the wrapped field. See [the Alfa cloud guide](alfa-cavitation-cloud.md). |
| [Propeller vapor medium](../scene/medium/propeller_cavitation.go) | `NewPropellerCavitation(config)` builds one bounded analytic field with an explicit blade count, onset, phase, advection, density breakup and pressure-recovery controls. The Alfa adapter authors five blades; other propellers reuse the constructor. Set YAML is `scene.world.vapor-cavities`; Puzzlemouth `cavitation field` emits a starter block. See [the cavitation guide](alfa-propeller-cavitation.md). |

`BuildingPart` and `BuildingMaterial` remain source-compatible names for the
shared `ComponentSpec` and `ComponentMaterial` types.

The common vocabulary includes boxes, drums, domes, slit domes, rods and
dishes, with explicit segmentation, smooth shading, frames and drum profiles
where supported. The building/spacecraft guide documents accepted
combinations, bounds, material roles, input-copy guarantees and
shared-reference limitations. These are Go construction APIs; they do not add
arbitrary component syntax to set YAML.

`PlanCity` and `PlanClusteredCity` return inspectable lot plans without
constructing building meshes. `SampleNeymanScott` samples the cluster
field used by clustered cities. These planning operations are documented
beside their types in [city.go](../manufacture/city.go),
[city_districts.go](../manufacture/city_districts.go), and
[neyman_scott.go](../manufacture/neyman_scott.go).

## Ground rigs and historical primitive constructors

[MakeGroundRig](../manufacture/ground_rig.go) makes a named ground
instrument at an explicit detail; `MakeGroundRigWithLaunchers` includes
its launcher geometry and light descriptions. [MakeRadioRig](../manufacture/radio_rig.go)
makes the named radio rig. Their `GroundRig` methods expose the made
geometry and parts to scene adapters.

[Cylinder and Cone](../manufacture/cylinder.go) make triangle arrays
from a base, height, radii, side count and material index.

[Conifer](../manufacture/conifer.go) and
[ConiferProper](../manufacture/conifer_proper.go) are older triangle
constructors used as geometry references, with their own parameter types
and defaults; [BranchOf and BranchCount](../manufacture/conifer_proper.go)
describe the latter's branches. These APIs return triangles directly,
rather than a prepared, shared manufactory specimen.

## Shared scene geometry, materials and media

Scene authors can also build directly from these primitives. Their inputs
and validation are documented at each linked constructor.

| Constructor | What it makes |
| --- | --- |
| [mesh.New](../mesh/mesh.go) | Validated immutable indexed triangles and their local acceleration tree; reuse the returned mesh for identical geometry. |
| [shape.NewMesh](../scene/shape/mesh.go) | A validated reference triangle mesh with exported fields. Treat its geometry as unchanged after creation; its internal tree is built at construction. |
| [shape.NewMeshGroup](../scene/shape/group.go) | A local hierarchy of flat mesh placements. Members may refer to the same immutable prototype; material belongs to the enclosing thing. |
| [shape.NewHeightField](../scene/shape/heightfield.go) | A finite square grid of terrain or water heights. It retains the supplied heights, so callers must preserve them. |
| [thing.NewWorld](../scene/thing/thing.go) | An empty scene world to which things, volumes and lamps can be added. |
| [thing.Still and thing.Sun](../scene/thing/thing.go) | A fixed placement motion and a directional light description. |
| [thing.Diffuse, Polished and NewVolume](../scene/thing/construction.go) | Plain and polished surface materials, or a stationary volume from a medium and material. |
| [thing.Made](../scene/thing/made.go), [Painted and FinishOf](../scene/thing/finishes.go) | A material from a material family and finish, or a named/authored finish. |
| [thing.NewVolumeIllumination](../scene/thing/volume_illumination.go) | Checked volume-light sampling settings for a world. |
| [medium.NewWeather](../scene/medium/mist_weather.go) | The shared prescribed weather parameters used by mist constructions. |
| [medium.NewGrove](../scene/medium/grove.go) | Canopy-bound mist with a ground-height function and an index over the supplied trees. |
| [medium.NewLayer and NewDeck](../scene/medium/layer.go) | A regional mist layer following ground, or an elevated cloud deck clipped by ground. |
| [medium.NewShellWeather](../scene/medium/shell_weather.go) | Prescribed cloud weather in a spherical shell around a planet. |
| [medium.NewPath, NewGrain and NewWake](../scene/medium/wake.go) | Circular sweep paths, seeded noise and a bounded gas wake around those paths. |

Analytic shapes such as `shape.Sphere` and `shape.Plane` are values,
rather than constructor functions; see [scene/shape](../scene/shape/shape.go).
A scene's materials, medium fields and native acceleration structures have
costs beyond the mesh storage shown by the model report.

## Sets and procedural scenes

[Set recipes](../set/recipes.go) refer to registered kinds and can select
a season or checked instrument parameters. The set and its placements supply
seed, detail and placement separately. [Sets](sets.md) documents the file
format, world, camera, terrain and placement vocabulary;
[space coordinates](../set/SPACE-COORDINATES.md) defines positions and
orientation. `set.Read` validates a set before construction.

Named procedural scenes and their parameters come from their owning
registry. `slayer list scenes` reads that registry at runtime, including
scene descriptions and parameter help; [scene/named](../scene/named/named.go)
defines the construction and registration contracts. Application-owned
Heard scenes register through [slayer_scene.go](../improv/searching/slayer_scene.go);
their scene-specific plans are documented in [the Heard README](../improv/searching/README.md).

[Authored anatomy](authored-anatomy.md) describes the opt-in individual and
small population constructors, their fields, shared meshes and limits.

Individual scene constructors can also create shapes, media and lights
directly. They do not automatically pass through the manufactory
catalogue, so catalogue complexity rows alone cannot be presented as a
complete scene inventory. Use a scene-level measurement when one is
available, and retain the distinction in reports.
