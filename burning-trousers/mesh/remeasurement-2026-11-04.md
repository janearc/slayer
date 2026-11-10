# Mesh comparison remeasurement, 2026-11-04

The independent picture now expands each group into ordinary world objects.
Each prototype is copied into `shape.Mesh`; the reference uses the world's
ordinary hierarchy rather than `MeshGroup` traversal.

Source prototype
faces and placements are shared inputs, so lost construction geometry still
needs separate construction checks.

The original three eyes are restored for all pre-existing kinds, full trees
and the optical observatory. The two new horizontal shoot specimens use
`(1, 0.1, 0.2)` for their second eye; the other eyes are unchanged. This is
an explicit new-specimen input, not a change to existing kinds' cameras.

At seed 3, 320 by 240, all eighteen kinds passed both sun-only and sun/sky
comparisons from all three eyes. Every picture met the unchanged floor of
300 covered pixels.

Every new tree, shoot and observatory view had zero
pixels past one JND and zero placement differences. The old bounds of two
sun-only and eight sun/sky pixels were retained.

The full run took 31.562 seconds with two Go CPU slots. The adjacent
`remeasurement-2026-11-04.log` preserves every comparison and the exact test
names. This remeasures acceptance margins for these inputs; it does not
establish all seeds, complexity levels or a biological shape reference.

Reproduce from the repository with no other rendering job active:

```sh
GOMAXPROCS=2 GOFLAGS=-p=1 game run check-mesh
```

Full project acceptance still uses `game check`. The harness's import and
history guards remain unchanged; these harness edits are committed alone.
