# Imported asset receipt

Successful draws of jobs with imported sets write a JSON sidecar after the
picture or movie output completes. A still uses `picture-path.imports.json`.
A film with persistent frames uses `frames/imports.json`; a movie rendered
through temporary working frames uses `movie-path.imports.json`, outside the
temporary directory.

The receipt records the exact canonical job and input hashes held by the
checked job snapshot, its render keys and build stamp, and each import's
authored source and license. It includes the decoder options key and verified
source closure paths, byte lengths and SHA-256 digests.

It also records native material bindings, material sources and definitions,
requested height normalization, and asset and world placements. Height
normalization includes original source bounds and the applied scale and
vertical offset. Receipt construction reads this frozen data without reopening
imported files; a later job read computes a fresh closure and key from current
bytes.

Oakland's CC0 fixtures are decoder tests, not measured or licensed human
anatomy. Each real imported source must supply its own rights declaration.
