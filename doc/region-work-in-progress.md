# Region work in progress

This branch is not ready for use or review. The job interface, CPU region
path, compact Metal band support, an experimental software Metal path, and
comparison primitives are drafts. Focused region tests passed for the job,
CPU tracer, Metal band origin, software GPU, session and door on small
scenes. Focused comparison, manifest, reference and snapshot package tests
also passed; the full project gates have not been run.

The public draw path admits a narrow set-still case and
refuses unsupported development and overlays; the street result has not been
established as an exact crop.

The experimental software GPU path passed small-scene region checks; other GPU
modes are refused. The reference store and comparison orchestration have an
experimental registered command; its dry run, small-scene draw, reference
reuse and damaged-reference refusal passed focused checks. The command relies
on the caller's asserted input closure and stable caches; the complete street
closure is not yet established.

The command can save and select a reference plan across builds. Its current
session source still requires matching declared inputs and effective settings
apart from sampling; comparisons across input changes remain unfinished.

Five street rectangles, old-image cuts and fresh regions are recorded. Their old-image
comparisons cannot establish equality because the old build and GPU mode are
unknown. No street equality claim follows from the small-frame checks.

At one ray per side, a fresh water region and a crop of a same-build whole
street frame produced identical delivered pixels in a BMP byte comparison.
Raw-light equality, other sampling rates and the remaining street regions
still need same-build checks.

A smaller water patch was drawn at 1, 2, 4, 16 and 32 rays per side. Raw-light
scores against the 32-ray reference are recorded in
`doc/region-water-linear-report.txt`; they are not monotonic. The reference is
finite and not proven converged. Repeat developed images matched at each
setting, but raw-light variation was not measured.

The wall patch has the same raw sampling sequence, recorded in
`doc/region-wall-linear-report.txt`. Its mean absolute difference falls as
sampling rises, while its isolated-positive count does not. The reference is
finite and not proven converged. Haze, lamps and the street end still need
the sampling sequence and raw comparison.

The original street render is recorded in the render notes as four rays
per side, sixteen samples per pixel, on the GPU with `street-2k.yaml` and
its chosen exposure. The renderer's exact GPU mode and complete input closure
still need confirmation. A request for 64 samples per pixel corresponds to
eight rays per side; 64 per side would instead mean 4096 samples per pixel.
