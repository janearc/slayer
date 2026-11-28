# object constructors

use `game run catalogue-list` to find the name of a kind for a set or a
Go scene. the [catalogue menu](../cmd/catalogue/menu.yaml) documents the
kinds and their supported detail levels.

`game run catalogue-resources -- mcrn-larson` constructs the named kind
at its advertised detail levels with seed 1 and reports geometry as JSON
lines. omit the name to check the catalogue. see
[model resources](model-resources.md) for what those counts measure.

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

a specimen's `Things(at)` places its surfaces. `Lights(at)` transforms
the attached light descriptions. `PlaceLights(descriptions, at)` works
with retained descriptions. [making models](making-models.md) has a
worked example and explains units, materials and provenance.

## Configurable factory recipes

each factory validates its spec. pass the resulting kind to `Make` or
`MakeWithComplexity` with a seed from `Sow`. `Spacecraft` and
`SpacecraftFrom` return a pointer to the kind.

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
| [Cavitation cloud](../manufacture/cavitation_cloud.go) | `NewCavitationCloud(config, pose)` makes seeded analytic gas spheres for a scene. Speed and authored duration set trail extent. `game run cavitation-cloud` emits a YAML fragment. See [the cloud guide](model/boat/alfa-cavitation-cloud.md) for the inputs and limits of this geometry model. |
| [Clear cloud medium](../scene/medium/clear_cloud.go) | `NewClearCloud(field)` opts a finite density field into mixed named-water transport while inheriting the legacy water phase. `NewClearCloudWithHenyeyGreenstein(field, g)` selects an independent authored cloud HG phase, finite `-1 < g < 1`; explicit zero is isotropic, not Rayleigh. Density, bounds and clock remain those of the wrapped field. See [the Alfa cloud guide](model/boat/alfa-cavitation-cloud.md). |
| [Propeller vapor medium](../scene/medium/propeller_cavitation.go) | `NewPropellerCavitation(config)` builds one bounded analytic field with an explicit blade count, onset, phase, advection, density breakup and pressure-recovery controls. The Alfa adapter authors five blades; other propellers reuse the constructor. Set YAML is `scene.world.vapor-cavities`; Puzzlemouth `cavitation field` emits a starter block. See [the cavitation guide](model/boat/alfa-propeller-cavitation.md). |

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

[set recipes](../set/recipes.go) can select checked parameters such as an
instrument's aperture, a plant profile or a city grid. place the result
in `scene.set.things` or `scene.set.stands`. the placement supplies the
seed and can override detail. see [sets](sets.md) for the file format and
how to check a scene before drawing.

use `slayer list scenes` to see the procedural scenes the running binary
offers. the [scene constructor reference](scene-constructor-reference.md)
shows how a job names one and how to build a scene directly in Go.

[Authored anatomy](model/bioform/authored-anatomy.md) describes the opt-in individual and
small population constructors, their fields, shared meshes and limits.

Individual scene constructors can also create shapes, media and lights
directly. They do not automatically pass through the manufactory
catalogue, so catalogue complexity rows alone cannot be presented as a
complete scene inventory. Use a scene-level measurement when one is
available, and retain the distinction in reports.
