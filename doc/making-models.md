# Making mathematical models with Slayer

The manufactory makes objects from construction rules, dimensions and
seeds. Geometry is produced by mathematics at the requested representation.
Slayer's model contribution path does not start with an imported ordinary
asset mesh. A mesh is one possible output of the recipe, ready for the
same renderer as other scene geometry.

## A recipe becomes a specimen

`manufacture.KindNamed` finds a checked recipe. `Sow` supplies reproducible
variation. `Make` constructs its default specimen; `MakeWithComplexity`
requests a representation. The result has local bounds, scene things and
attached light descriptions. Scene adapters place it in the world.

```go
row, err := manufacture.KindNamed("mcrn-larson-landed")
if err != nil {
    return err
}
made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), "quick")
if err != nil {
    return err
}
parts, err := made.Things(maths.Local())
if err != nil {
    return err
}
for _, part := range parts {
    world.Add(part)
}
```

This is a function-body fragment with an existing `world`; import
`manufacture` and `maths` from the Slayer module. The specimen type is
private but its methods are public: infer it with `:=`. `manufacture.Kind`
and `AttachedLight` are exported opaque names. Do not depend on private
implementation fields in client code.

The current declarations are in
[offered.go](../manufacture/offered.go) and
[complexity.go](../manufacture/complexity.go).

`Things` handles geometry and material grouping; it does not install lamps.
Call `made.Lights(place)` with the same placement and adapt descriptions
through `emit.Of(description.Light(), description.At(), description.Toward(),
exposure)`. Respect each light's source exclusion (`Beyond`) in the scene
adapter. Exposure is a scene conversion choice, not an extra physical watt.
Keeping manufacture free of an `emit` import preserves the dependency direction.

## Add a new kind

1. Define local axes, origin, dimensions, parameter bounds and the intended
   physical or authored claim. Manufactured ground objects normally stand
   on local y=0; y is up.
2. Find an existing family and composition helper. A row can be enough for
   another species or instrument. Put a new growth rule beside shared rules
   when the existing architecture cannot describe the object.
3. Add independently computed construction tests and nearby invalid cases.
   Start with a bounded specimen, not a forest or a full astronomical set.
4. Construct from checked primitives, transforms, profiles and material slots.
   Retain the first failure and refuse an incomplete result.
5. Register the maker in the kind table or its family row map. Add checked
   parameters and schema/recipe adapters only where callers need them.
6. Expose a catalogue specimen or small set, then test CPU and supported
   Metal rendering, bounds, materials and representation costs.

`maker`, `good`, `instrumentAssembly`, `newTube` and `turnedProfile` are
package implementation names, not exported client APIs. Adding a factory
kind means working inside `manufacture`; a client composes existing public
recipes instead. Historical capitalized constructors in
[the original design](manufactory.md) are not a current extension API.

## Build from geometry rather than a stored surface

A tube joins endpoint rings with taper. A turned profile revolves a list
of radius/height points. A placement gives a component its own coordinate
frame. Repeated members share a generated mesh where their geometry agrees.
Assemblies retain separate material slots rather than baking their colours
into a texture image.

Choose counts as well as lengths deliberately. More angular segments refine
a surface; they should not move an instrument's pivot or change its aperture.
Validate winding, degeneracy, normals, bounds and clearances. Transparent
shells need real inner and outer boundaries; coincident interfaces can
produce ambiguous ray hits.

Put authored proportions and budgets in the family's checked numeric data
with units and provenance. Published numbers belong in sourced research
data and the verified shelf. Equations remain derivations. A plausible
shape is useful without pretending its dimensions were measured.

## The MCRN Larson

The [Larson maker](../manufacture/saucer.go) is a flying saucer built from
profiles and arrangements. Its [proportions](../manufacture/saucer-choices.yaml)
are picked for the plain saucers in Gary Larson's cartoons. The titanium
hull, glass dome and material optics do not turn that fictional design
into spacecraft engineering.

The hull is a radius/height profile revolved around its axis: flat belly,
bowl, rim band and upper slope. The glass dome has an outside hemisphere,
wall thickness and inner hemisphere. Its foot sits into the deck to avoid
coincident glass/deck hits. Another revolved profile makes the belly disc.
Portholes are arranged by angle around the sloped hull, oriented to its
surface normal.

The [landed version](../manufacture/saucer_landed.go) reuses the flying
body, adds its tripod and a flight from the shared stairs constructor,
then raises the body so the feet define the ground origin. Its checked
[gear recipe](../manufacture/saucer-landed.yaml) keeps structural dimensions
and material choices visible.

Separate material looks give the hull titanium optics and an oxide film,
the dome transmission/refraction, and portholes a visible glow. Attached
lamps provide the actual illumination: cabin, belly and named running
lights with a time programme. A glowing surface alone is not a substitute
for a lamp illuminating the ground.

The four detail levels use 12, 24, 48 and 96 turns around the hull. They
refine one design; the Larson intentionally has no seed-dependent shape
variation. Deterministic does not require every model to change with seed.
The [night set](../improv/larson/landed-night.yaml) shows the landed ship
under its own lights; it also needs its pinned terrain data and a compatible
native GPU. It is a worked scene, not the dependency-free first run.

## How trees work

A tree combines branching architecture, foliage rules and a species row.
A seed selects dimensions and arrangements reproducibly. Trunk and limbs
form a scaffold; whorls and branchlets place supporting shoots. Species
with compatible rules can share that code while changing their data.
This is procedural construction, not physiological growth or a simulation
of a tree getting older.

Recursive terminal shoots have a continuing axis and contracting lateral
axes in local frames. Child radii depend on parent radius and relative
length, using an adapted Weber-Penn relationship. Redwood laterals form
flat sprays; sequoia laterals turn around the parent. Leaves attach to
the woody axes rather than floating beside an image of a branch.

Coast redwood uses flat needles in two ranks; giant sequoia uses overlapping
leaves in three rows. Some leaf length ranges have botanical sources;
spacing, width, recursion budgets and much of the crown remain authored.
The named spruce/Norway entries now use radial needle shoots too, while
legacy conifer references remain for comparison.

Read
[recursive foliage](conifer-foliage.md) and [spruce foliage](spruce-foliage.md)
for which sources and acceptance results apply to each family.

Increasing a shoot's reach adds more cells of physical-size needles; it
does not stretch every needle. `Taller` changes a supported whorled tree's
construction proportions and population of whorls/shoots. Scaling an entire
instance instead scales all its needles too. `Branched` adds side branches
to supported whorled recipes; it is an explicit recipe change.

## Detail and sharing keep forests possible

`hull`, `clumps`, `quick` and `demo` are representation names, not a
universal polygon multiplier. Redwood/sequoia clumps use an open leaf
diamond, quick a closed tetrahedron and demo finer closed leaves. Spruce
clumps instead use foliage proxies. Unsupported levels are refused.
Preserve the specimen's scaffold and declared physical extent across detail.

Small shoot prototypes are shared at placed instances. A material's
`MeshGroup` owns local placements and a local hierarchy. The forest adds
world instances; traversal visits world instance, shoot instance, triangle.
Groups contain flat meshes, not arbitrarily nested groups. Unique stored
faces and all repeated scene faces are different cost measures.

Use loose `coast-redwood-shoot` or `giant-sequoia-shoot` kinds to inspect
leaves before a whole crown. Test attachment, dimensions, recursive
contraction and rays through real gaps. Then inspect a whole-tree silhouette
from several sides and a moving lit view. An open diamond can be a valid
declared preview, so “every representation is a closed solid” is not the
right universal foliage assertion.

## Finish a model contribution

Provide the recipe, checked data, independent assertions, a bounded example,
supported backends and a useful camera. Keep geometry/reference changes
reviewable and record representation cost before scattering thousands.
Use [testing](testing.md) for checks and
[contributing](../CONTRIBUTING.md) for commit and review conventions.

The [city building recipes](city-buildings.md) provide standalone architectural
families and a reusable street grid with exactly 10,000 seeded buildings,
cross alleys, traffic circles and material-batched native geometry.

The [macrostructure and terrain recipes](macrostructures.md) add typed
kilometre-scale arcologies, twisted spires, inhabited bridges, mountains
and calderas, with formulae and existing biome placement metadata.
