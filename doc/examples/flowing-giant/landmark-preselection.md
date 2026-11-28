# Weather landmark preselection

Before either landmark-view frame was submitted, Azazel job 334 ran
`weather-landmarks scan landmark-views.yaml`, using the GPU checkpoint
already kept by the 315–316 weather. The tool read the checkpoint without
advancing it and independently projected the candidate positions into the
two declared 1024 × 1024 pinhole cameras. Full run log remains at
`queue/00000000000000000334.0001.log`.

```text
body=amber radius_m=69595212.898129418 cells=256 seed=7 requested_s=1000000 checkpoint_s=1001182.1140183887 field=GPU
candidate=1 lat_deg=13 lon_deg=277 spin=0.26479411220822358 origin_deg=12.599635536942449 disc_px=(610.974,607.148) second_px=(829.120,359.544)
candidate=2 lat_deg=-3 lon_deg=255 spin=-0.23539251274308998 origin_deg=-4.4738624498002562 disc_px=(326.155,772.502) second_px=(576.623,541.099)
```

The selector used a one-degree grid, chose the largest absolute spin among
points that were sunlit and comfortably visible in both cameras, and chose
the next strongest point at least 20 degrees away. A candidate is not yet
a verified visible feature. Coordinates are body-frame latitudes and
longitudes; pixel coordinates are predicted, not measured from the images.
Compare the specified neighborhoods regardless of whether they match, and
retain untouched frames and linear PFM files alongside any marked copies.

## Observation

After this preselection, Azazel jobs 335 and 336 rendered the declared
views with the pinned integrated Slayer `1c1a1bae2`: 1024 × 1024 pixels,
two rays per axis, CPU. The jobs now write their `proof.png` and 32-bit linear
`light.pfm` in `out/2026-11-27-flowing-landmark-disc/` and
`out/2026-11-27-flowing-landmark-second/`, respectively.

The independent visual researcher inspected both preselected locations with
a top-left image origin. Candidate 1 lies in mottled brown/gray in the disc
view and on a brown wavy boundary in the second; no distinct common feature
could be identified. Candidate 2 lies in a mottled cyan band in the disc
and a cyan band with blurred dark shapes in the second; no individual
feature could be matched confidently. Neither observation establishes
camera coherence. No replacement feature was selected afterward, and the
predicted projections themselves have not been independently verified from
an identifiable surface marker. The close views also show soft block-like
structure. The PFM files and originals are retained.

## Separate coordinate markers

The `landmark-marked-views.yaml` diagnostic is a separate render: it adds
small red and green glowing spheres for candidates 1 and 2 at the calculated
body-frame locations, 300 km above the nominal radius, each 300 km in
radius. They are made-up visible markers, not clouds, storms or vorticity.
Their known world positions are compared with the precomputed projections;
the altitude and finite radius can shift the observed centers slightly.
The original frames remain untouched. This can show whether a geometric
point projects consistently in both views; it cannot establish whether the
underlying fluid field itself remains fixed across cameras.

Azazel jobs 345 and 346 produced separate marker re-renders. Their jobs
now write to
`out/2026-11-27-flowing-marker-disc/proof.png` and
`out/2026-11-27-flowing-marker-second/proof.png`. The independent
visual researcher saw both red and green markers close to the predicted
neighborhoods in both views. This is a visual, not subpixel, comparison;
the markers' elevation and radius account for some offset. The underlying
weather at those locations still did not show distinctive features the
researcher could confidently match. Thus the geometric projection is
supported at image scale while visual weather coherence remains inconclusive.

## Separate checkpoint-field projection

Azazel job 359 used `weather-landmarks map` to project the same retained
GPU checkpoint independently into the two camera views. Its four images
are recorded in the original queue receipt: `field-disc.png` and
`field-second.png` are unmarked; their `marked-` counterparts add yellow
and green rings at the two preselected locations. Red and blue show positive
and negative vorticity, respectively, and gray is near zero; the invented
color mapping saturates at magnitude 0.3. These are diagnostic field colors,
not optical colors. The projector does not call the shaded renderer.

The independent visual researcher found a red patch inside candidate 1's
yellow ring in both views and a blue patch inside candidate 2's green ring
in both. Neighboring shapes are broadly compatible, but the dense pattern
does not establish exact correspondence by eye. The diagnostic shows one
cached field under two projections; it establishes neither the renderer's
field sampling nor evolution over time or physical validity. The shaded
weather landmarks remain visually inconclusive. Original shaded images,
linear outputs and unmarked field images remain separate and untouched.
