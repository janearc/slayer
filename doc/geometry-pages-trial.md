# Geometry page packing trial

This is an experimental CPU packing foundation. `visitGeometryPages`
receives the encoder's pending mesh trees, validates the page budget and
all global index limits, and visits packed pages in original mesh order.
Pages preserve global face, normal and node offsets. It preflights input
before emitting and refuses an oversized mesh; subdivision is not yet
implemented.

The byte budget applies to the packed page data, not to the
process, source trees, staging arrays, GPU acceleration or ray queues.

The page visitor is not wired into the renderer. A page cannot be sent
independently to the existing trace kernel: the renderer and native
acceleration plan still expect global scene data. This is not a working
paged renderer, image-decomposition implementation, or a 2 GiB guarantee.
The page visitor's pending source trees are themselves held in memory.

Focused `go test -tags=draw ./gpu -run '^TestGeometryPage' -count=1`
passes through the Game tests lane. It compares reassembled packed buffers
and native host mappings to the whole encoder for branched and smooth
meshes, shared groups and repeated placements, and tests budget boundaries,
preflight and visitor errors. No GPU render or full-suite test was run.
