# Model maturity

Every spacecraft and observatory is being brought to a minimum of 600
physical components, with an engineering and material review alongside the
count.

`catalogue inventory` emits the current count, shortfall, physical bounds,
limitations and references for each kind. Selectors restrict it to named
kinds. Counts survive material batching, placement and prototype reuse;
compound mirrors count their separate backed panels. Triangle subdivision
never increases the count. Review remains pending even above the floor.

Two shared plans provide the required presentation:

```sh
game run catalogue-stamped -- render --plan doc/catalogue-cove.json --out out/cove spitzer
game run catalogue-stamped -- render --plan doc/catalogue-space.json --out out/space spitzer
```

Build `catalogue-binary` through Game before using these commands. Each
output directory retains the render settings and producer evidence.

The cove has a white swept wall, a checkerboard floor and the manufactory
kind `human-scale-2m`. The figure is exactly two metres tall in scene units
at every detail level. It stands beside the model at the model's lowest
point; the camera frames both without resizing either one. The figure is
an authored metal mannequin, not a claim about human anatomy.

The space plan requires a pinned Gaia DR3 CSV on `SLAYER_SHELF`. The source
key is `gaia-dr3-bright-presentation`. Its query returns at most 12,000
sources brighter than G=7 from the Heidelberg ARI archive; this capped
sample is not a complete all-sky catalogue. Its exact bytes are pinned,
so a different response cannot silently alter a render.

The file has `source_id,ra,dec,phot_g_mean_mag` columns and preserves positions
and magnitude ratios. RA/Dec are used at the catalogue epoch without
proper-motion correction. Display star size and brightness are stylized.

Space views use gentle directional sunlight. The existing per-model
camera heading and elevation remain useful for showing each object's
geometry. The supplied plans render at 640 by 480.

The first hardware upgrade is described in [Spitzer maturity](spitzer-maturity.md).
It clears the count floor but retains explicit material and dimension
limitations.
