# Broadleaf, brush and Joshua-tree construction

The names are `valley-oak`, `alder`, `poplar`, `manzanita`,
`chaparral-brush`, `joshua-tree` and `saguaro`. Oak forks without a continuing
leader. Alder and poplar keep a stronger leader. Manzanita and brush start
with several basal stems. Joshua trees end their forks in blade rosettes.

Every dimension, colour, angle and count in `plant-profiles.yaml` is an
authored scenery choice. The scale range controls the scaffold, rather
than promising an exact final tree height. These are family recipes,
not measured specimens or a simulation of plant growth.

## Branching and leaves

The scaffold is a bounded parametric recursion. Child axes contract,
turn in their parent's frame, and take a radius from a picked length
power. A continuing leader has its own angle and length multiplier.

These rules adapt the branching constructions in [The Algorithmic Beauty of
Plants, chapter 2](https://algorithmicbotany.org/papers/abop/abop-ch2.pdf) and
the radius rules in Weber and Penn's *Creation and Rendering of Realistic
Trees*, listed in [the conifer sources](conifer-foliage.md).

Neither publication supplies our species profiles.

Foliage uses physical-size cells: a short continuing twig, lateral twigs
and individual blades. Oval, lobed, lance and deltoid outlines are closed
thin meshes with a raised midrib. Their dimensions stay fixed when the
tree gets larger. A larger branch carries more cells, within a budget.

Each kind has a finite set of shared cell prototypes for each detail.
`hull` uses one crown proxy; `clumps` uses a proxy per foliage cell;
`quick` and `demo` draw individual blades at different surface sampling.
The same seed retains the scaffold and physical foliage envelope.

Proxy clumps represent groups of leaves, rather than giant leaves. Joshua
trees instead use stiff green blades around each tip, plus downward brown
blades below them.

The qualitative rosette and persistent old-leaf habit follows [Oregon State
University, Yucca
brevifolia](https://landscapeplants.oregonstate.edu/plants/yucca-brevifolia).
Blade counts and angles remain picked.

Valley oak is the deciduous, lobed-leaf oak here.

See [UC ANR, California oak identification and
ecology](https://www.ucanr.edu/site/uc-oaks/california-oak-tree-species-id-and-ecology)
and the [USDA valley-oak
guide](https://plants.sc.egov.usda.gov/DocumentLibrary/plantguide/pdf/pg_qulo.pdf).

Alder, poplar, manzanita and generic brush are authored family treatments.
Their meshes do not identify a particular local species.

## Saguaro columns and arms

Saguaro has its own construction. A pleated upright column carries zero
or more curved arms. Each arm starts inside the trunk, bends through a
quarter-circle into an upright run, and ends in a rounded cap.

The [National Park Service account of saguaros](https://www.nps.gov/tont/learn/nature/saguaro.htm)
describes their arms and expandable pleats. Our dimensions, pleat count,
modulation, arm placement and spine geometry are picked scenery choices.
The mesh does not model age, stored water, growth or flowering.

`hull` and `clumps` use plain columns with fewer curve sections. `quick`
and `demo` resolve the pleats and place shared physical-size spine cones
along upright runs. Bends and rounded caps currently have no spines.
Arm paths, radii, cap dimensions and reserved bounds stay fixed.

## Seasons

A named construction recipe can choose foliage independently of kind:

```yaml
recipes:
  winter-oak:
    kind: valley-oak
    season: winter
```

Use this inside the existing scene recipe block and refer to `winter-oak` from
a thing or stand. `summer` is the default; `autumn` changes the picked
deciduous palette; `winter` removes deciduous blades but retains woody
branches and twigs. Evergreens keep foliage in every season.

The six tree/shrub kinds accept this control; saguaro has no foliage season.

These are discrete authored states, not dates or simulated phenology.
In Go, `manufacture.WithSeason(kind, "winter")` returns a new `Kind`.
Unsupported kinds and unknown seasons return an error.

Changing season keeps the seeded scaffold and reserved leaf-on envelope; this
avoids moving a stand's spacing when its leaves disappear. Winter `hull`
retains the woody scaffold and twigs rather than a leafy crown proxy.

Its face cost can therefore exceed the summer hull's cost; season is not a
promise of cheaper geometry.

## Mangroves

The names are `kandelia-obovata`, `aegiceras-corniculatum`,
`avicennia-marina`, `bruguiera-gymnorhiza` and `sonneratia-apetala`: the
low native kinds of a tidal shore and the tall planted one behind them.
They are profiles in `plant-profiles.yaml`, branching and leafing as the
other broadleaf kinds do, and every seed grows a different tree.

What marks them is the roots, a `roots` block in the profile. A profile
without one shows none, as every other kind here does.

```yaml
roots:
  habit: pencil      # pencil or prop
  count: {minimum: 40, maximum: 80}
  height: {minimum: 0.1, maximum: 0.3}   # metres above the ground
  radius: 0.007      # metres, at the foot
  tip: 0.6           # radius at the far end, as a share of the foot's
  spread: 0.4        # as a share of the tree's scale
```

A `pencil` is an upright peg or cone, standing in a field about the trunk
out to `spread` times the scale, each patch of that ground as likely as
another, each leaning a little. Avicennia has pencils and sonneratia has
taller, thicker cones, a `tip` near nought. A `prop` arches from the trunk
at its `height`, out and down to the ground `spread` times the scale away,
the props of a trunk spaced about it. Kandelia and bruguiera have them.

Every root starts a little below the ground, so it stands in it, and
every detail shows the same roots: only their roundness changes.
Heights and radii are in metres and do not grow with the tree.

Heights, leaf sizes and each kind's root habit are recalled from memory,
not looked up and not measured, and each profile says so in its `by`.
The number of roots, the width of a field and every other number are
picked. Bruguiera's knee roots are drawn as propped buttresses. A root
system here is a few dozen roots, not the hundreds a shore has.

### A shore

A stand grows where the ground lies between its `above` and `below`, in
metres on the terrain's datum. On a terrain with `sea: {value: 0, unit: m}`
the sea is nought, so a band over it reads as a shore:

```yaml
set:
  stands:
    kandelia a:
      kind: kandelia-obovata
      count: 40
      seed: 101
      complexity: clumps
      above: 0.1
      below: 1
```

A stand makes four distinct trees and repeats them, so a shore wants
several stands of a kind, each with a seed of its own. See
[the mangrove shore](../../../sets/mangrove-shore.yaml) and
[the study](../../../sets/mangrove-study.yaml). The bands in them are picked
and are not any shore's tide.

## Native kinds of the island and coast sets

The names are `pandanus-tectorius` (hala), `scaevola-taccada` (naupaka
kahakai), `metrosideros-polymorpha` (ohia lehua), `barringtonia-asiatica`
(utu), `quercus-ilex` (holm oak), `erica-arborea` (tree heath) and
`salix-alba` (white willow). Each is a profile row and nothing else.

Each row cites the page that calls it native where it is sown, and quotes
what was read there beside the number it set: heights, leaf lengths,
outlines and habits. A number with no quotation is picked. Leaf widths
are picked throughout; no source read gave one.

Hala uses the rosette form, as the Joshua tree does, with prop roots for
its aerial roots. The rosette's downward blades stand for the older,
hanging leaves of its crown. Tree heath's needle leaves are lance blades
five millimetres long, in pairs, not the whorls the plant bears.

Kinds the forms cannot make are left out: palms, pines and reeds. See
[the study](../../../sets/native-flora-study.yaml), one of each.

## Rendering boundary

Products use the existing closed-mesh and shared-mesh-group routes.
They introduce no new GPU primitive or special CPU-only foliage path.

The prototype cache is keyed by kind, detail and a bounded variant count.
It does not add an entry per tree, world position or animation frame.
Scaffolds, cell attachments and explicit blades also have input budgets.
