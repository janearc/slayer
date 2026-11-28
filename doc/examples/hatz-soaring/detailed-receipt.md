# Detailed pterosaur visual receipt

The adapted detailed-geometry study was built at `ee10c3d09`. The 960×640, two-ray
textured CPU quarter and top frames (Azazel 562–563) took 8 and 12 seconds
respectively in their progress logs. The GPU `detailed-plain` quarter
(564, binary `1052391f4`) took 11 seconds; it retains the painted hide but
omits membrane paint. These are individual elapsed render reports, not
controlled resource benchmarks or an estimate for a twelve-animal scene.

The independent visual observer found that both quarter views show a more
detailed body, head and limbs than the initial glider; the textured CPU and
plain GPU silhouettes are broadly similar. The top view shows both wings,
their elongated leading supports, finger-like projections and trailing legs.
The transverse wing lines, angular edges, overlap of feet, and conspicuous
head/beak joins remain; the quarter view largely hides the far wing. Both
backend and membrane material differ in this comparison, so it does not
isolate paint or establish renderer parity. No anatomical validation or hero
quality is claimed.

The jobs now write frames and linear raw in `out/2026-11-27-hatz-detailed-`
`quarter-cpu`, `top-cpu` and `plain-quarter`, each as `proof.png` and
`light.pfm`.
