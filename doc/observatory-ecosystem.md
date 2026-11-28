# an observatory above the forest

[the set](observatory-ecosystem.yaml) places a manufactured observatory
on volcanic terrain with a seeded forest and mist shared by the grove.
after the [Game setup](developing.md), draw it from the checkout root:

```sh
game build slayer
game run observatory
```

the PNG is written to `out/observatory.png`. the USGS elevation model must
be unpacked on the source shelf. the scene requests the GPU and requires
Metal.

the elevation model is from Kilauea. the observatory site, its coast
redwoods and weather are authored.

## recipes and specimens

`scene.recipes` describes what can be made. `scene.set.things` places
individual specimens and `scene.set.stands` places groups. each has an
explicit seed. changing a specimen's placement keeps its construction.

the observatory recipe supplies dimensions and pointing for its enclosure,
opening, mount and telescope. the set uses the
[bundled v1 schema](../set/ecosystem.schema.json). see
[sets](sets.md#schemas-and-stage-support) for validation and version choice.

## detail and clearance

the scene defaults to `quick`. the forest chooses `clumps`, and the
observatory chooses `demo`. these names select model representations.
see the [catalogue](catalogue.md) for the levels available on each kind.

`exclude` leaves a clearing 40 metres beyond the observatory’s enclosing
circle. the stand places up to 5,000 trees above 1,150 metres. the model’s
shared meshes let the renderer reuse foliage across specimens.

## atmosphere

the forest and its weather share one mist volume over the union of the
trees’ canopy footprints. the terrain forms the bottom. the mist tops out
at the lower of the canopy height and the prescribed depth. overlap
keeps the same extinction. coverage sets a noise threshold, wind moves
the patches, and boundary fade softens their edges.

the CPU samples the field directly. the GPU samples a bounded density
grid prepared on the CPU, so thin canopy edges can be missed. the mist
describes an authored appearance. [redwood atmosphere notes](model/vegetation/redwood-atmosphere.md)
record the sources that would inform a physical model.
