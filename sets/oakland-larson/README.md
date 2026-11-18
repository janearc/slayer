# A visitor beside the Larson

Open `set.yaml` in Ferruccio and render its preview. The scene combines
Slayer's existing `mcrn-larson-landed` manufacture with Oakland's static
glTF import. The eight-part mannequin and its binary buffer are original
CC0 fixtures; their rights declaration is in `LICENSE.txt` and the set.

The mannequin's two-metre height is a chosen geometric reference. It is
not a scan or a claim about a particular person's anatomy. Its matte
colour, the landing pad and lighting are authored choices. The Larson's
geometry and materials come from the existing manufactory unchanged.

The set places the figure at east 7, north 1, up 0 metres. Change `at`
or `heading` under `scene.world.imports.visitor` to place it elsewhere;
the Larson's placement is under `scene.set.things.larson`. A camera's
`location.at` and `orientation.look_at` use the same map coordinates.

For a local comparison, use the Game-built Slayer:

```sh
game lane tests -- bin/slayer draw sets/oakland-larson/set.yaml \
  --renderer cpu --out out/oakland-cpu.png
game lane gpu -- bin/slayer draw sets/oakland-larson/set.yaml \
  --renderer gpu --out out/oakland-metal.png
```

Each imported render writes an `.imports.json` receipt beside its image.
It identifies the decoded source closure and native material bindings.
Changing a source, the height choice or the set's material definition
changes a fresh job's render identity. An admitted job keeps its frozen
geometry and exact digests, including across command-line overrides.

See [the import contract](../../set/IMPORTS.md) and
[Oakland's supported subset](../../oakland/DESIGN.md) before importing
another asset. USD, textures, skinning and animation are not implemented.
