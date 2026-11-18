# Imported geometry on a set

`scene.world.imports` places a checked Oakland asset beside native set
pieces and manufactured things. The first version supports static glTF
and GLB. It requires an explicit native material for every source slot.
The importer never supplies a renderer or turns appearance values into
claims about measured materials.

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

Jobs without imports keep their current lazy input handling.
This addresses imported sources,
not the separate backlog of every other transitive dataset in Slayer.

Checks cover exact source closure, native binding refusals, coordinate
and unit conversion, geometry sharing, changed-source cache identity,
and snapshot consistency. A rendered human beside the existing Larson
provides the final CPU/Metal comparison and app proof.
