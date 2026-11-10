# Broadleaf, brush and Joshua-tree construction

These factory additions are code drafts. Compilation, independent geometry
checks, appearance records and CPU/Metal views are deferred to the review
break. The older conifer captures do not validate these new families.

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

## Rendering boundary

Products use the existing closed-mesh and shared-mesh-group routes.
They introduce no new GPU primitive or special CPU-only foliage path.
That design intent still needs the scoped CPU/Metal checks at review.

The prototype cache is keyed by kind, detail and a bounded variant count.
It does not add an entry per tree, world position or animation frame.
Scaffolds, cell attachments and explicit blades also have input budgets.

The global forest selector, projected-size detail selection, mixed-biome
examples and seasonal transitions remain separate work.
