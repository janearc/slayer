# The manufactory's catalogue

The [browsable specimen book](browsable-catalogue.md) adds local search,
detail views, source notes and actual GPU render records to this written list.

The previously reviewed catalogue held 70 kinds. This code draft registers
nine more and changes the two spruce entries: OWL, its shelter, seven
vegetation families and the replacement spruce foliage await review. Each name
makes a specimen from a seed.

The dimensions below describe the current rows, not caller-set options. Except
where noted, these are authored choices, not measured botanical or engineering
models.

## Plants and fallen wood

- [spruce](#spruce): a whorled conifer with draft recursive needle shoots.
- [norway](#norway): a taller spruce with hanging branchlets.
- `coast-redwood`: an authored 18 to 42 m conifer with flared bark
  and flat sprays, [construction row](../manufacture/redwoods.go).
- `giant-sequoia`: an authored 20 to 45 m conifer with a thicker
  flared trunk and scale-like detail, in the same construction file.
- `coast-redwood-shoot` and `giant-sequoia-shoot`: the actual reusable
  foliage specimens, exposed for close views. Their recursive rules,
  published leaf lengths and authored choices are recorded in
  [conifer foliage](conifer-foliage.md).
- [fern](#fern): fronds carrying paired leaflets.
- [log](#log): fallen wood with broken branch stubs.
- `valley-oak`, `alder` and `poplar`: draft deciduous trees with separate
  seasonal foliage. `manzanita` and `chaparral-brush` use several stems;
  `joshua-tree` has terminal rosettes and old-leaf skirts. See
  [vegetation construction and sources](vegetation-families.md).
- `saguaro`: draft ribbed columns, curved arms, rounded tips and physical
  spine cones. All dimensions and pleat counts are authored.

### spruce

A tree 4 to 11 m tall, with whorls spaced 0.4 to 0.6 m apart and
5 to 8 branches in the lowest whorl. Bark and needles have separate
materials. The draft named-factory wrapper replaces foliage sleeves with
shared recursive shoots and physical needles around each twig.

The seed varies dimensions and branch placement. The original whorled
row remains as a legacy reference. New geometry and appearance checks are
deferred. [Draft foliage and its sources](spruce-foliage.md).

### norway

A Norway spruce 6 to 13 m tall, with whorls spaced 0.5 to 0.75 m
apart and 5 to 7 branches in the lowest whorl. Branch pieces carry
paired rows of drooping branchlets, each wearing foliage sleeves.

It shares the spruce's architecture, with a different row and added
branchlets. The draft named-factory wrapper carries recursive physical
shoots at all four detail levels. The legacy reference row is retained.
[Draft foliage and its sources](spruce-foliage.md).

### fern

Six to eleven fronds, each 0.4 to 0.9 m long, with 14 to 24 leaflets
per frond. Stalks and leaflets share a green appearance. The seed
selects the dimensions, counts, and arrangement.
[Construction row](../manufacture/kinds.go).

### log

A fallen log 1.5 to 4 m long, with two to five broken branch stubs.
It tapers and rolls about its length, resting above the ground.
The material is bark; there is no separate exposed end-grain material.
[Construction row](../manufacture/kinds.go).

## Stones and litter

- `pebble`: longest axis 4 to 64 mm; wornness 0.6 to 1.
- `cobble`: longest axis 64 to 256 mm; wornness 0.3 to 0.9.
- `boulder`: longest axis 256 mm to 2 m; wornness 0 to 0.5.
- `can`: a fixed-size aluminium can, standing or lying on its side.

All three stones use seeded lumps with different proportions,
perturbation, and cleaving. Wornness is the generator's tuning value,
not a measured erosion rate. Their appearance is granite. The size
class boundaries were recalled from memory, not verified against a table.

The can is 2 11/16 inches across and 4 13/16 inches tall, also recalled
from memory. Its seed changes whether and how it lies, not its size.
It is a closed tube with an aluminium appearance, without a pull tab
or label. [Construction rows](../manufacture/kinds.go).

## Fire and lamps

- [campfire](#campfire): stones, leaning sticks, and an attached light.
- [bonfire](#bonfire): a large pile of logs; the scene supplies fire.
- [spotlight](#spotlight): a mounted head with a thermal cone beam.
- [maser](#maser): a mounted dish with a radio line and aperture spread.

### volumetric fire recipes

a named YAML recipe can also make fire independently of fuel geometry:
a single source, a seeded cluster, or a line, with optional smoke.
these are recipe kinds, separate from the static names above.

[fire recipes](fire-recipes.md) gives the schema, working sets, sources
and current GPU support. place logs or other fuel geometry separately.

### campfire

Seven to ten stones surround four to six leaning sticks. The stone
ring's radius is 0.35 to 0.5 m. Its attached light is isotropic,
flickering, 1100 to 1500 K, and 2000 to 6000 W. These are picked
parameters, not a fuel or combustion model.

The geometry and the attached light description are made together.
The scene converts the description through `emit` and chooses exposure.
[Construction row](../manufacture/lamps.go).

### bonfire

Eleven to fifteen logs, 6 to 8 m long, lean together over a ring
whose radius is 2.4 to 3 m. This is a large pyre, not an enlarged
campfire with the same attached lamp.

The manufactured object carries no light. A set's `alight` treatment
supplies the fire through the scene's fire physics.
[Construction row](../manufacture/bonfire.go).

### spotlight

A stand 1 to 1.6 m tall carries a head tipped down 10 to 25 degrees.
Its attached 3200 K light has picked power of 500 to 2000 W, a beam
half-angle of 12 to 25 degrees, and an additional 6-degree soft edge.
[Construction row](../manufacture/lamps.go).

### maser

A mount 1.5 to 2.5 m tall carries a dish 2 to 4 m across, aimed
30 to 70 degrees upward. The picked power is 0.5 to 5 W. The line
comes from 1420.4 MHz, recalled from memory, and the aperture sets
its spread by diffraction.

Its emission is outside visible light. A visible picture of the housing
does not demonstrate the radio beam. [Construction row](../manufacture/lamps.go).

## Buildings

### home

Two or three cantilevered storeys, each 6 to 14 m wide, 6 to 12 m
deep, and 3 to 3.5 m tall, with flat roof slabs and window panes.
The seed selects the plan, offsets, glazing, and which rooms look lit.

The panes use reflective and glowing materials. They are not transparent
windows into furnished rooms, and the glow is not an attached emitter
from `Lights`. [Construction row](../manufacture/home.go).

## Personal telescopes

- dobsonian-36: open 36-inch Newtonian, timber rocker, mirror box,
  paired trusses, altitude bearings and side focuser.
- apo-refractor-5: her 5-inch refractor with dew shield, rings, diagonal,
  finder and counterweighted tripod.
- sct-24: her 24-inch short-tube instrument, corrector proxy, secondary,
  fork, pier and rear focuser.

All four details preserve the requested apertures and physical envelope.
[Construction and limits](personal-telescopes.md) distinguish scenery from
a functioning optical prescription.

## Ground observatories


- `optical-observatory`: the configurable dome, mount, truss and primary.
- `palomar`: a long tube on a tilted horseshoe support inside a dome.
- `keck-i` and `keck-ii`: a shared 36-panel primary and open-truss template.
- `subaru`: a cylindrical enclosure around an open alt-azimuth assembly.
- `rubin`: a short survey instrument with primary and tertiary figures on
  one backing, a larger annular secondary and a camera housing.
- `griffith`: a civic observatory with wings, a domed rotunda and rooftop
  instrument domes. Its east refractor has a nominal 12-inch objective;
  [building dimensions and materials](civic-observatory.md) are authored.
- `elt`: a five-mirror assembly around a 798-panel primary with a central
  opening. Its [merge checks](large-observatory.md) are still in progress;
  the exposed CPU/GPU image comparison is not yet accepted.

The new named templates have fixed pointing and dimensions, with four surface
details. Their ordinary closed meshes use the existing CPU and GPU routes.

Source dimensions and authored support proportions remain separate; the Keck
enclosure dimensions are authored. [Family design](observatory-family-design.md)
records the sources and work still required for the remaining instruments.

## Space observatories

- `spitzer`: an octagonal bus, cryostat, optical baffle and solar shield.
- `hubble`: a long telescope tube and two gridded solar wings.
- `webb`: 18 gold-looking mirror tiles, a secondary and five shield layers.
- `space-observatory`: the default Spitzer template; a checked YAML recipe
  selects a design and an aperture explicitly.

These are visual templates, with fixed structural dimensions for each
aperture. Seeds retain the scene's placement-yaw rule. Circular tessellation
changes with complexity; the physical envelope stays fixed.
[Recipes, sources and limits](space-observatories.md).

## Making and placing a specimen

Look up a kind with `KindNamed`, sow its seed with `Sow`, and pass both
to `Make`. The implementation types remain private.

The returned object's `Things(placement)` gives meshes grouped by material.
`Lights(placement)` gives attached light descriptions at the same placement.
Both return an error for invalid placements; callers now handle
`lights, err := specimen.Lights(placement)`.

Up is y for these generated objects. Some older reference constructors
and kernel tree representations use z up; their transforms matter.
The same kind and seed reproduce the same specimen. Scattering that
specimen over terrain is the scene's job.

`cmd/hillside` retains its original kind count flags. The newer trees,
loose shoots and observatory recipes use the [YAML scene](observatory-ecosystem.md).
Its `-variety` option reuses a limited number of specimens at many places.
That saves repeated geometry; it does not create additional kinds.

## GPU support belongs to the kind

A new kind must work through the GPU kernel before it is called
supported. Its catalogue entry needs to say how it reaches the kernel
and what has actually been checked. A CPU-only picture is insufficient.

There are two existing routes. Named kinds expose meshes or shared mesh
groups through `Things`; the GPU pools and instances their geometry. Separate
kernel conifer representations reconstruct geometry from compact tree
parameters. Adding a named kind does not add that second route for it.

The existing hillside CPU/GPU comparison exercises boulder, fern,
spruce, campfire, and spotlight together. It is not a per-kind GPU
certification of the whole catalogue. The kernel also has dedicated tests
for its older procedural trees, cones, transformed meshes, and lights.

For additions, check intersections and normals, placement and scale,
materials and shadows, and attached light where present. A procedural
kernel representation also needs seed and parameter agreement with the
CPU construction, plus bounded traversal and storage.

Choose geometry whose CPU and kernel representations can be maintained
together. A new primitive or growth rule may require encoding, shared
GPU structure declarations, intersection code, and validation. Those
changes are part of delivering the kind.

## Where the catalogue is thin

There are four conifers, one fern, three stone sizes, one house style,
three spacecraft templates and a wider family of summit and personal
telescopes.

The optical recipe exposes diameter, slit, aperture, primary
style and pointing; yard equipment supplies a working site's surroundings.
The [working scene](observatory-ecosystem.md) describes its limits.

There is no named furniture or general optical component family.
The spotlight and maser demonstrate
geometry carrying light; they do not provide a general instrument kit.

New species that share an architecture can mostly add rows. New
families may need construction rules shared by several kinds. Keep
those rules reusable, with dimensions and provenance beside each row.

The quality ladder and further tree architectures in
[families.md](families.md) remain partly planned. The older export
`ConiferProper` is a reference constructor, not another named kind.
[manufactory.md](manufactory.md) records the original design and history.

[Redwood groves and their atmosphere](redwood-atmosphere.md) records
the transpiration and fog references for future ecosystem recipes,
including what they do and do not justify as model parameters.

[An observatory above the forest](observatory-ecosystem.md) gives the
working YAML recipe joining observatory construction, vegetation,
complexity, terrain placement, and a shared grove atmosphere.

[The night-fire scene](fire-night-design.md) adds a reusable volumetric fire
recipe, firebrands and a prescribed front over placed fuel. These use
bounded GPU fields and shared instances; they are scene recipes, rather
than additional names in KindNamed.

[The run report](fire-night-result.md)
records the completed 4K animation and the checks it actually passed.

## Summit yard

See [the yard equipment](yard-equipment.md) for cranes, vehicles, scaffold,
utility equipment and access structures. These are authored scenery kinds
with four details and closed ordinary meshes.

The `guide-star-launcher` adds an open launch tube, collars and attachment
shoe. The [launcher notes](guide-launchers.md) describe the shared beam
direction, optional visible housing and explicit placement heading.

## Hawaiian summit families

The [Hawaiian observatories](hawaiian-observatories.md) add optical, solar,
radio and tracking families, including shared antenna and dome layouts.
Source apertures and nominal classes stay separate from picked buildings.
