# authored manufacture records

geometry-reference-provenance.txt describes the preserved pre-refactor
catalogue record: 78 geometry signatures and 66 expected refusals. Its
platform and source hashes are part of that record.

fire-layout-reference.txt holds six authored source positions, phase seeds
and heights. Run `game run sample-fire-layout` from
the module root to reproduce it. These are placement choices, not measured
combustion data.

legacy_foliage_sample_test.go prints the vertices retained in
assembly_foliage_vertices_contracts_test.go. Copy it into manufacture,
run `game run sample-legacy-foliage` from the Slayer root,
then remove that temporary copy. It needs package access to the fallback
foliage primitive. The vertices hold authored shapes, not botanical data.

Review intentional asset changes with their renders before changing these
records. The physical, source and GPU acceptance checks remain separate.

light-reference.json records eighteen attached sources: campfire, spotlight
and maser, two seeds and three placements. Their beams, spectra and flicker
are authored choices. Run `game run sample-lights` to
reproduce them. The manufactured bonfire is an unlit log pile with no lamp.

monolithic-reference-arm64.txt holds the four detail levels for the authored
24 m enclosure and 8 m monolithic mirror. The main catalogue reference uses
segmented mirrors. Run `game run sample-monolithic`
to reproduce these additional geometry records on Darwin/ARM64.

later_foliage_sample_test.go prints the later-twig vertices retained in
assembly_later_foliage_vertices_contracts_test.go. Use it like the earlier
foliage producer, with TestDiagnosticPrintLaterFoliageSamples. The two
records cover the first four supporting twigs.
