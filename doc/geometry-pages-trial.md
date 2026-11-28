# Geometry page packing trial

the draw-tagged CPU experiment `visitGeometryPages` validates the
mesh trees, page budget and index limits before visiting packed
pages in original mesh order. pages preserve global face, normal
and node offsets. an oversized mesh is refused.

The byte budget applies to the packed page data, not to the
process, source trees, staging arrays, GPU acceleration or ray queues.

the page visitor is test-only. the renderer and its native acceleration
plan still require global scene data. pending source trees remain in memory.

the focused CPU tests compare reassembled buffers and native host
mappings against whole-scene encoding, for branched and smooth
meshes, shared groups and repeated placements. they also check page
budgets and visitor errors.

```sh
game test -- -tags=draw -count=1 ./gpu -run '^TestGeometryPage'
```
