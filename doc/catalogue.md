# The manufactory's catalogue

The [browsable specimen book](browsable-catalogue.md) adds local search,
detail views, source notes and actual GPU render records to this written list.

The catalogue menu holds 125 kinds. The specimen book draws one picture
of each, and `game run catalogue-list` lists them. Each name makes a
specimen from a seed.

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
row remains as a legacy reference. [Draft foliage and its sources](spruce-foliage.md).

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
- [stage lasers](#stage-lasers): seven snouts on stands, each shining
  one visible line or a mix of lines through a narrow opening.
- [strobe](#strobe): the spotlight's cold wide cousin.

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

### stage lasers

`laser-green`, `laser-red`, `laser-blue`, `laser-cyan`, `laser-amber`,
`laser-magenta` and `laser-white`: a slim snout on a stand 0.8 to
1.2 m tall, tipped level to 12 degrees skyward, with a picked power
of 2 to 8 W through a 3 mm opening, whose diffraction sets the spread.

The wavelengths are the usual show modules' lines, recalled from
memory: 532, 638, 445, 488 and 589 nm. A magenta or white head mixes
two or three lines from one mouth, sharing the one power, as real
projectors do.

A beam this narrow lights what it hits; the shaft itself appears only
where a scattering medium stands in it, as [guide-star
beams](guide-stars.md) explains for the scene-level kind.
[Construction row](../manufacture/stagelights.go).

### strobe

A stand 1.2 to 1.8 m tall carries a short wide head tipped down 0 to
10 degrees. Its attached 6500 K light has a picked power of 400 to
1600 W, a beam half-angle of 35 to 55 degrees, and a 15-degree soft
edge. Its flashing is not here: a light's timing belongs to the scene,
not the manufactory. [Construction row](../manufacture/stagelights.go).

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
- apo-refractor-5: a 5-inch refractor with dew shield, rings, diagonal,
  finder and counterweighted tripod.
- sct-24: a 24-inch short-tube instrument, corrector proxy, secondary,
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
  opening. See [large observatory](large-observatory.md).

The new named templates have fixed pointing and dimensions, with four surface
details. Their ordinary closed meshes use the existing CPU and GPU routes.

Source dimensions and authored support proportions remain separate; the Keck
enclosure dimensions are authored. [Family design](observatory-family-design.md)
records the sources for the remaining instruments.

## Space observatories

- `spitzer`: an octagonal bus, cryostat, optical baffle and solar shield.
- `hubble`: a long telescope tube and two gridded solar wings.
- `webb`: 18 gold mirror tiles and a secondary on a tower, five aluminium
  shield layers, and the bus, five solar panels and antenna under them.
- `space-observatory`: the default Spitzer template; a checked YAML recipe
  selects a design and an aperture explicitly.

These are visual templates, with fixed structural dimensions for each
aperture. Seeds retain the scene's placement-yaw rule. Circular tessellation
changes with complexity; the physical envelope stays fixed.
[Recipes, sources and limits](space-observatories.md).

## Visitors

- `mcrn-larson`: a flying saucer of weathered titanium under a glass dome.
  It carries fourteen lamps by name: `cabin`, a red light under the dome;
  `belly`, a floodlight under the hull; and `port-0` to `port-11`, the
  running lights at its portholes, whose glass glows.
- `mcrn-larson-landed`: the same saucer on three legs, its stairs down from
  the belly. Its own zero is the ground it stands on.

Patterned on the plain saucers Gary Larson drew in The Far Side. Every
number of it is picked by eye, in metres; none is engineering.

## Crewed spacecraft

- `apollo-csm`: the Apollo command and service module, standing on its
  engine bell, 10.11 m tall and 3.91 m across.
- `apollo-lunar-module`: the Apollo lunar module on its four footpads,
  7.04 m tall and 9.4 m across them. It is no stack: an eight-sided
  descent stage in a blanket of Kapton over aluminium, an eight-sided
  ascent stage on it, and four legs of one strut each.
- `gemini`: the Gemini spacecraft on its adapter, 5.736 m tall and 3.048 m
  across its base.
- `mercury-capsule`: the Mercury capsule on its heat shield, 3.3 m tall and
  1.8 m across, without its escape tower.
- `soyuz-ms`: Soyuz MS on its service module, 7.48 m tall, without its
  solar arrays. Every size of it is unverified.

## Planetary landers

- `venera-lander`: the lander of Venera 9 to 14, standing on its impact
  ring, 1.97 m tall and 2.15 m across its drag plate. A placeholder of
  plain shapes: a ring, eighteen legs, a round payload module, a plate
  and a parachute can, all titanium.

## Engines

- `j58`: the Pratt & Whitney J58, the A-12's engine, lying on its axis,
  4.572 m long and 1.27 m across its largest case. Its counts, nine
  compressor stages, two turbine stages, eight cans and six bypass
  tubes, are NASA's; its length and diameter are a 1966 register's,
  read through a secondary page.

  Its stations, its radii but the
  largest and its looks are picked, in
  [its sizes file](../manufacture/j58.yaml). It is its own kind so that
  a nacelle or a stand can place it by its mount.
- `j58-test-stand`: a J58 on a test stand, a cradle on flexures over a
  pad, holding the engine by its four mount pins, a load cell between
  the cradle and a thrust post, and a bellmouth on the face. No source
  for a stand was found; every number of it is picked, in
  [its sizes file](../manufacture/j58-stand.yaml).

## Vessels

- `alfa`: Project 705 "Lira", NATO's Alfa class, the titanium attack
  submarine, on its keel, 81.4 m long and 9.5 m in beam: its hull, its
  sail with the rescue sphere inside it, two bow planes, a cruciform
  stern with rudders, stern planes and a thruster pod on each
  stabiliser's tip, one sourced five-blade non-skewback main screw, and
  two sourced two-blade auxiliary screws on motor pods at the stern-plane tips.

  Its
  length, beam, draught and displacements are russianships.info's, and
  the rescue sphere and the sail's rubber coating are the CIA's history
  of the boat; both are on the shelf.

  Its form, its parts' sizes, the
  propeller dimensions and its looks are chosen from photographs, in
  [its sizes file](../manufacture/alfa.yaml), where every number says
  whether it was measured or chosen. `alfa` keeps its bare-titanium
  default; `alfa-hero` selects a black coat without changing it.

  At demo
  complexity the model adds approximate photo-derived mast/antenna forms,
  sail covers and a starboard condenser scoop.

  Feature identities and
  propeller blade counts come from *American Submariner*, Third Quarter
  2023, pp. 22–23; dimensions and placements are not measured.

  See
  [the detail-model notes](alfa-model.md).

For the water, the boat's own space is in metres with y up, the keel at
y nought, the centreline on x nought, the bow toward minus z and
starboard toward plus x.

Its hull is one closed solid, wound
counterclockwise as seen from outside; the sail, planes and fins are
closed solids of their own, sunk into it.

`manufacture.Floating("alfa")`
gives the surfaced waterline above the keel (6.24 m, the default of the
two cited draughts; Wikipedia's 7.6 m is the other, by name), the name
of the look the hull wears, `hull`, and the look `wet hull`, which no
face wears: the water decides where the hull is wet.

The finish is a
choice in the sizes file: bare titanium by default, or painted dark.

## Launchers

- `saturn-v`: the Saturn V's three stages and instrument unit, 85.6 m tall
  and 10.06 m across, without the Apollo spacecraft on top.
- `saturn-v-apollo`: the Apollo-Saturn V as it stood on the pad, with the
  Apollo spacecraft and its escape tower, 110.64 m, NASA's 363 feet.
- `saturn-ib`: the Saturn IB's two stages and instrument unit, 43.16 m
  tall and 6.61 m across, with no spacecraft on top. Its sizes are read
  from a secondary page and not pinned.
- `sls-block-1`: the Space Launch System's core stage with a solid rocket
  booster on each side, its adapter and its upper stage, 86.7 m tall,
  without Orion on top.

Neither has engines, fins or its own paint yet.

The crewed spacecraft and the launchers are stacks: bodies of revolution
listed in
`manufacture/stacks.yaml` as sections from the bottom up, each with what
it is made of, how it is finished and where its numbers were read. A new
capsule, stage or launcher is rows in that file. Its numbers are read off
pages and are not yet pinned sources; its looks are picked.

## Vehicles

- `hemtt`: Oshkosh's HEMTT, the M977 A4 eight-wheel-drive cargo truck.
- `kamaz`: the KAMAZ-43118 six-wheel-drive cargo truck, cab over engine.
- `tatra`: the Tatra T 815-7 eight-wheel-drive cargo truck, on its backbone
  tube with swinging half-axles.
- `unimog`: the Unimog U 5023, cab behind a short bonnet, with a short bed.
- `scania-r` and `daf-xf`: two-axle tractor units, each with a fifth wheel
  for a trailer.

The menu gives each one's overall dimensions and says which were read from
the maker and which were picked to look right. They are built at `quick`
detail only. `service-truck` and `utility-pickup`, the summit's own, are in
the menu's Summit yard group.

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

`hillside` retains its original kind count flags. The newer trees,
loose shoots and observatory recipes use the [YAML scene](observatory-ecosystem.md).
Its `-variety` option reuses a limited number of specimens at many places.
That saves repeated geometry; it does not create additional kinds.

## GPU support

Named kinds expose meshes or shared mesh groups through `Things`; the GPU
pools and instances their geometry. Separate kernel conifer representations
reconstruct geometry from compact tree parameters. A named kind does not use
that second route.

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

The older export `ConiferProper` is a reference constructor, not another
named kind.
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
