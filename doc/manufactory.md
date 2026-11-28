# the manufactory

Slayer builds a tree, a spacecraft or a city from construction rules.
a kind describes the object. a seed selects repeatable variation, and
a detail level selects a representation. the scene places the result
and its attached lights together.

- [sets](sets.md) shows how to place a catalogue kind or a checked recipe
  in YAML.
- [making models](making-models.md) shows how to make and place a specimen
  in Go, with worked spacecraft and tree examples.
- [object constructors](constructor-reference.md) lists the public
  factories and placement APIs.
- [model resources](model-resources.md) explains how to count the geometry
  in a specimen or a scene.

## design origins

the factory replaced separate triangle generators with shared forms and
placement rules. `BranchOf` in
[the reference conifer](../manufacture/conifer_proper.go) can locate a
branch by seed, whorl and index without storing the preceding branches.
tapered tubes model trunks and branches. thin blades model leaves and
needles. seeded perturbations and cleaves give stones their shape.

materials group the resulting surfaces. the factory describes its attached
lights, and the scene adapter supplies them to the emitter package. a
spectrum, power and angular spread describe a light before the scene
applies exposure. see [emitters](emitters.md) for the light model.
