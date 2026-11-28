# Imported geometry on a set

`scene.world.imports` places static glTF or GLB geometry beside native
pieces and manufactured things. each source slot needs an explicit
native material. [the visitor beside the Larson](../sets/oakland-larson/README.md)
is a complete example with a preflight command.

```yaml
scene:
  world:
    materials:
      mannequin:
        colour: {srgb_linear: [0.32, 0.38, 0.42]}
        source: eye
    imports:
      visitor:
        file: assets/two-metre-human.gltf
        height: {value: 2, unit: m}
        at: [7, 0, 0]
        heading: {value: 0, unit: deg}
        materials: {material/0/human-fixture-slot: mannequin}
        source: "Self-authored CC0 mannequin; chosen two-metre height"
        license: CC0-1.0
```

Paths are relative to the set file and remain within that directory,
including symlink resolution. `scene` optionally selects a glTF scene
index. `height` is optional: without it, glTF metres are retained. With
it, Oakland places the selected scene's lowest point at zero and scales
its geometric height, explicitly a modelling choice.

Placement `at` is in set map coordinates (east, north, up); `heading`
follows set headings.
Geometry in metres is divided by the world's metres-per-unit once.

The required `source` and `license` record the author's provenance and
rights declaration. They do not independently certify legal ownership.

The set reader loads immutable geometry and preserves its exact source
closure. Stage binds native materials and adds ordinary `thing.Thing`
values. Meshes and the CPU/Metal tracing paths stay under their existing
owners. The set's named material definitions remain in the set digest.

A job with imports freezes the checked set, its bytes and each decoded
source digest at construction. Staging uses that same checked set. A
fresh job sees changed sources and has different light, picture and
resume keys. An existing job retains its original geometry and identity;
it never rereads new asset bytes under an old key.

the [Oakland contract](../oakland/DESIGN.md) describes the supported
geometry and the files needed beside the asset.
