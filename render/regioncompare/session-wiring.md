# Compare rendered regions

`slayer compare-regions SET` draws named rectangles from a frame and compares
them with a higher-sample reference. The command writes a linear-light
`report.json`, a labelled `sheet.png` and the reference manifests in
`references.json`. See `slayer compare-regions --help` for the options.
Submit the draw through [Azazel](../../doc/azazel.md).

Supply an absolute frozen `--snapshot` and explicitly list every input
relative to it. `--complete-inputs` asserts that the list is complete and
stable. The command cannot discover missing dependencies. Keep
`SLAYER_SHELF` in the snapshot and use a stable `SLAYER_CACHE` outside it.
Reference and output directories must also be outside the snapshot.

Set the full frame's width and height and the candidate and reference
rays per side. The reference must use more rays. Keep its manifest fixed
on another run with `--reference-plan` and measure candidate variation
with `--repeat`. The command can install a missing reference without
replacing an existing one. The digest checks stored bytes. Coverage of
the source inputs and convergence of the reference need separate checks.

Native GPU traversal, bloom and overlays are outside this comparison path.
