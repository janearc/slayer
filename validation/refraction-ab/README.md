# Synthetic clear-interface refraction A/B

This is a controlled validation of Slayer's clear-material interface. All
chart geometry, colours, lighting, and dielectric values are synthetic. The
1.333 index is not a measured water value or a claim about San Onofre, Hawaii,
or another site. The setup is a finite-thickness clear solid with an opaque
blue sphere crossing its plane; it has no liquid region, absorption,
scattering, participating medium, or underwater camera transport.

The four actual rendered original scene/job inputs remain in the shared render
gallery at `frames/2026-11-16-studio-clear-interface-refraction-ab/`, alongside
all four images, assessment/render logs, and provenance. This repository keeps
one canonical scene and job plus standard unified diffs, avoiding four copies
of the 107-line scene. `sha256sums.txt` records both the original captured
input hashes and the exact hashes of the generated scene variants.

## Inputs and equivalence

`scene.yaml` is attempt 1's index-1.0 control (`index: 1.0`, `gloss: 1`). The
three files under `patches/` change only the interface's single authored
`clear` line:

- `attempt-1-index-1.333.patch`: index 1.333, gloss 1.
- `attempt-2-index-1.0.patch`: index 1.0, gloss 0.
- `attempt-2-index-1.333.patch`: index 1.333, gloss 0.

The attempt-1 pair therefore differs only in index. The attempt-2 pair holds
gloss at zero and differs only in index. Attempt 1's gloss 1 lets Fresnel
reflection vary with index too, so it is illustrative but does not isolate
bending.

In the inspected attempt-2 CPU images, the index-1.333 chart is
visibly displaced beneath the slab compared with the straight-through 1.0
control. That is evidence of index-dependent ray bending in this renderer and
setup, not a complete water model.

`job.yaml` is the single canonical job: it fixes the relative `scene.yaml`
reference, CPU renderer, 240 × 135 film, one ray per pixel, and time zero. For
each variant, copy it next to the generated scene. The normal Slayer `--out`
override selects a distinct image destination; no job parser or extra format
is involved. After creating `$WORK/a1-10`, `$WORK/a1-1333`, `$WORK/a2-10`, and
`$WORK/a2-1333`, the reproduction commands are:

```sh
cp validation/refraction-ab/scene.yaml "$WORK/a1-10/scene.yaml"
cp validation/refraction-ab/job.yaml "$WORK/a1-10/job.yaml"
cp validation/refraction-ab/scene.yaml "$WORK/a1-1333/scene.yaml"
cp validation/refraction-ab/job.yaml "$WORK/a1-1333/job.yaml"
patch -d "$WORK/a1-1333" -p1 < validation/refraction-ab/patches/attempt-1-index-1.333.patch
cp validation/refraction-ab/scene.yaml "$WORK/a2-10/scene.yaml"
cp validation/refraction-ab/job.yaml "$WORK/a2-10/job.yaml"
patch -d "$WORK/a2-10" -p1 < validation/refraction-ab/patches/attempt-2-index-1.0.patch
cp validation/refraction-ab/scene.yaml "$WORK/a2-1333/scene.yaml"
cp validation/refraction-ab/job.yaml "$WORK/a2-1333/job.yaml"
patch -d "$WORK/a2-1333" -p1 < validation/refraction-ab/patches/attempt-2-index-1.333.patch
```

Each patched scene is byte-for-byte equal to the original scene used for that
render; its SHA-256 appears in `sha256sums.txt`.

The original four job YAMLs
in the gallery retain their explicit relative output paths. The compact job
plus `--out` replacement changes only the scene/output path coordinates: all
other validated job settings are identical. To assess/draw a generated job
through the Game tests lane, use the pinned Slayer binary and the existing
Puzzlemouth interface:

```sh
game5 lane tests -- "$SLAYER_BIN" assess "$WORK/a1-10/job.yaml" --out "$OUT/a1-10.png"
game5 lane tests -- "$SLAYER_BIN" draw "$WORK/a1-10/job.yaml" --out "$OUT/a1-10.png" --fresh
```

Repeat with the other job directories and output filenames. The checked-in
render evidence used Game5 511afb9, source commit
`f7889fb8fdfa167c2c12bb671d2d0f0ffab742c9`, and an unstamped custom Slayer
binary with the hash in `sha256sums.txt`; exact commands and output logs are in
the gallery `recipe.txt` and variant directories. No GPU or window was used.

## Compact reproduction verification

I generated all four variant directories from this canonical scene/job and the
three patches, then assessed each copied job with its own `--out` destination
through Game5's tests lane.

All four assessments said the job would draw as a
still and named the requested destination. I drew all four through the same
CPU lane and pinned binary into a temporary directory.

Each resulting PNG was
byte-for-byte equal to its preserved gallery original; all four `cmp` checks
returned zero and their hashes are recorded above. The temporary reproduced
images are not part of the shared gallery; the four original images and their
original input YAML remain there.
