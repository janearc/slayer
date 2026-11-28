# Clustered city districts

`city-clustered-10000` places
10,000 buildings on selected lots in a 40 by 40 block grid, with two large
continuous parks and a diagonal elevated freeway.

[The example set](../../city-clustered-10000.yaml) has `overview`, `park` and
`interchange` cameras. Its
[still job](../../examples/jobs/city-clustered-10000.yaml) runs
through shared Slayer after the normal Game build:

```sh
game run slayer -- draw doc/examples/jobs/city-clustered-10000.yaml
game run slayer -- draw doc/examples/jobs/city-clustered-10000.yaml --camera interchange --out out/city-interchange.png
```

The recipe supplies district reservations, connected highway geometry
and park circulation. The green and night variants add native trees, woodland
masses, open meadows, curved paths and discrete facade panes. All dimensions and organization are
authored scenery; planning runs on the CPU and the example renders through
`gpu-native`. No road-engineering or traffic simulation is claimed.

`SampleNeymanScott(spec, Sow(seed))` returns genuine finite-window Poisson
parents and independent Poisson offspring per parent. Parent intensity is
`1 / ParentSpacing²`. `ThomasGaussian` uses independent normal axis offsets
with standard deviation `Spread`; `NeymanScottDisk` uses uniform disk-area
offsets with radius `Spread`. Offspring are not clipped to the parent window.
This is a finite-parent-window process, not an infinite stationary plane.

`DefaultClusteredCity`, `ClusteredCity(spec)` and
`PlanClusteredCity(spec, Sow(seed))` compose those parent centres with road
lots. A Gaussian mixture intensity supplies weighted selection without
replacement. The exact population, disjoint lots, parks, highway/ramp
clearance and a small uniform tail change the raw process statistics. This
city realization is not passed off as unmodified Thomas offspring.

`NeymanScottSpec` exposes typed lengths for parent-window dimensions,
spacing and dispersion, plus dimensionless mean offspring. `ClusteredCitySpec`
contains `Grid`, exact `Count`, `Process`, `Parks` and `Highways`. Park
width/depth and highway width, median, clearance and elevation are typed
lengths; positions use the city's metre x/y/z frame. Plan accessors return
owned building, parent, park, highway and actual ramp records.

`ClusteredCitySpec.Grid.BatchPlacement` uses the same independent Go API
setting as `CityGrid`: its zero value or `CityBatchFused` selects the default
fused append; `CityBatchNested` selects historical nested placement.

Both
preserve exact output and mesh-failure behavior. Set and job YAML do not
expose it on direct `kind` placements. To choose it in YAML, define a named
city recipe with `batch-placement: fused` or `batch-placement: nested`, then
reference that recipe from an individual thing.

The four registered grid and
clustered-city kinds support this key; non-city recipes and stand/ecosystem
uses refuse it. This setting remains independent of quality and lighting.
Jobs keep their existing set reference, whose contents contribute to job
input provenance.

`ClusteredCitySpec.VacantLotIndex` is a separate control for tree-bearing
clustered-city recipes. Its empty value and `CityVacantLotIndexOn` cache the
vacant lot records during planning; `CityVacantLotIndexOff` reproduces the
historical full-grid plan-and-rescan when the vacant landscape is made.

The default remains indexed. In set YAML, a named recipe for
`city-clustered-green-10000` or `city-clustered-night-10000` may set
`vacant-lot-index: on` or `vacant-lot-index: off`, then be referenced by an
individual thing. Other kinds, recipes without trees, stands and ecosystems
refuse the setting.

It is independent of `batch-placement`, tree settings,
quality and lighting; it does not alter selected buildings, reserved margins,
vacant lot order or geometry.

The recipe remains part of the ordinary set
source provenance; job YAML keys do not change.

The default freeway has two distinct carriageways separated by a median
gap. Four cubic descending ramps join two actual grid streets. Piers are
placed away from local carriageways and cross alleys.

Park surfaces cover
former road/lot patches into continuous landscapes; a connected walking
cross uses split surfaces rather than overlapping coplanar path triangles.
These are native geometry primitives and material meshes, without another
renderer or planning framework.

Seed 31 has 19 Poisson district centres. Its 10,000-building quick specimen
has 970,316 faces in nine material groups and 21 native mesh batches.

`city-clustered-green-10000` and `city-clustered-night-10000` are opt-in
variants; the earlier clustered and approved grid defaults remain stable.
Their sets and jobs use the same names under `doc/` and `doc/examples/jobs/`. Cameras
include `overview`, `park`, `interchange`, `park canopy` and `tree row`.

Set `spec.Trees` to `DefaultCityTrees()` and `spec.Night` to true to construct
a custom night specimen. `CityTreeSpec` exposes existing plant kind, physical
height, street/park spacing and prototype budget.

`Trees()` returns owned
placements, planting seeds and shared `PrototypeSeed` identities. Valley oaks
use existing native quick leaf geometry by default. Planting still uses the
original physical hull bounds, preserving coordinates, scale and seeds. Four
geometry prototypes retain shared foliage-cell meshes across every placement.

Seeded groves leave explicit meadow clearings. Park circulation has a curved
perimeter loop and winding crossing routes, with trunks kept off paths.
Vacant lots receive soil/meadow surfaces. Night panes occur on individual
floors and bays across four elevations; occupancy varies by building and
pane, including unlit windows. Wall mass itself does not emit.

Six physical downward fixtures describe actual light through the existing
manufactory lamp API. Their visible posts serve ramp streets and park paths.
The night set adds a cool distant moon and sky context, for seven lights in
total. Emissive windows do not claim to illuminate surrounding roads. Named
dimensions, occupancy and fixture choices are in district-choices.yaml;
all are authored scenery choices.


`CityTreeSpec.Complexity` selects `hull`, `clumps`, `quick` or `demo`; an empty
value selects `quick`. The green/night examples now use actual native leaf
geometry, including close park and tree-row views. An overview caller may
explicitly choose `hull`; changing foliage detail never relocates a tree.

Existing tree groups are lifted into the city's material hierarchy by composing
cell and planting transforms. Their original flat mesh pointers stay shared;
no unsupported nested groups or multi-million-face per-tree flattening is
introduced. Seed 31's day plan has 4,852 unchanged trees and 1,598,864 cell
instances, but only eight unique foliage meshes and 81,296 stored faces.
