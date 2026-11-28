# a visitor beside the Larson

open [set.yaml](set.yaml) in Ferruccio to see a manufactured Larson beside
a static glTF mannequin. use the [Ferruccio quick start](../../doc/applications/ferruccio/quick-start.md)
to open the set and connect the preview queue. keep the glTF, its buffer
and [license](LICENSE.txt) beside the set.

`scene.world.imports.visitor.at` places the visitor at east 7, north 1,
up 0 metres. `heading` turns it. the imported figure is two metres tall
as a geometric reference. the Larson is a seeded kind under
`scene.set.things.larson`. the camera uses the same map coordinates.

from the checkout root, build the Slayer command and check the set or
write a still:

```sh
game build slayer
game run slayer -- assess sets/oakland-larson/set.yaml
game run slayer -- draw sets/oakland-larson/set.yaml --renderer cpu --out out/oakland-cpu.png
```

on a Mac with Metal, use `--renderer gpu` and a different output path to
compare the two paths. a render writes an `.imports.json` receipt beside
the image with the geometry files and material bindings. the
[import guide](../../set/IMPORTS.md) describes asset paths, rights and the
supported part of glTF.
