# Oakland integration checks

This records the 15 November 2026 integration checks. The original
checkout used Game5 511afb9, Go 1.27.3, two processors, Go parallelism one
and a local Celestial replacement. See [the current build guide](../doc/building.md)
for checkout prerequisites.

## Contracts exercised

Oakland tests cover the supported glTF/GLB subset, exact source closure,
shared geometry, explicit material slots, scene selection, affine transforms,
normal transformation, reflected winding, grounding and height normalization.
Refusals cover malformed framing, unsupported features, invalid indices,
resource limits, path escapes, cycles and non-finite geometry.

Set, job, stage and session tests exercise explicit native bindings,
world coordinates and units, source-dependent render identity and staging
from the same frozen geometry that the job hashed. Job overrides retain
that snapshot even after source files have been deleted.

Receipt tests check nested source paths, declared rights, native definitions,
normalization bounds and transform, frozen hashes, and durable still/film
sidecar destinations. A swatch stays a named swatch; no black RGB is invented.

The bundled Larson study stages the actual manufactured spacecraft and eight
imported mesh parts. Its transformed figure vertices span zero to two metres
within 1e-12 metres. These are chosen geometry dimensions, not measured anatomy.

## Results and limits

The combined Oakland, set, job, stage, session and door run passed except
for nine legacy set digests. Retaining new material provenance for all sets
changed their private representation. Receipt retention is now restricted
to import-bearing sets; the original baseline table was left untouched.
Both complete set and door suites then passed in check32.

Vet passed all six packages. Oakland statement coverage is 80.7 percent;
the profile is checks/oakland.cover.out. This is package coverage, not a
claim of complete rendering or whole-Slayer coverage. The combined run,
vet and initial receipt checks were observed in tool output; their full
console logs were not saved. Check32 is saved in checks/32-legacy-receipts.log.

Post-implementation overlays of the previous decoder body failed the
grounding and both geometry-work budget assertions. These were fault checks
after the repairs, not test-first failures. Earlier integration red/green
records are in checks/10 through checks/24; compile and fixture failures are
distinct from the intended assertion failures.

Actual CPU/Metal images and the Ferruccio window proofs are recorded with
their outputs. A successful decoder test alone does not establish a working
scene. USD, animation, textures, skins, morphs and export are outside this
slice. Derived meshes are shared within a decoded asset and a set read;
there is no new persistent mesh cache to serve stale geometry.

## Built scene, 15 November 2026

Game build34 passed source verification and all 17 command builds at
c6a3002f. Lint33 reports 634 inherited hard findings and 24 looks, with
none in this change's files. This is not a clean whole-repository lint.

Checks39 and 40 render the final Larson study with explicit CPU and GPU
selection. Both images are 640 by 360. Their mean absolute RGB byte-channel
difference is 1.0586487268518519, RMSE 3.096204366879144 and maximum
channel difference 74; 40.55816 percent of pixels differ in some channel.

These compare developed PNG bytes, not linear radiance. Visual inspection
finds matching geometry, composition and cast shadow with localized noise.
The data and small Go comparison program are retained with the images.

The first CPU image used a flood plus sun; Metal refused that combination
in check36. The final scene uses two supported directional lamps. The first
image and refusal log, and the dimmer intermediate pair, are retained in
the gallery. The lights are authored studio choices, not a planetary
atmosphere model or calibrated optical measurement.

Packaged Ferruccio 540a635 also renders the imported study in native Metal
windows and changes the actual camera through a drag. Its captures15/16
include raw previews, source receipts and import sidecars. The image
export is from the GoGPU widget tree. See the
[original native validation](../ferruccio/native/VALIDATION.md) for measurements.
