# Measured relief target

Jobs 293–294 used experimental combined binary 8f2377983, the same seed,
thin authored air, light, camera, 800 × 800, 2 rays per axis and exposure
0.003. The flat scene uses an analytic sphere; the other uses the 256 × 128
diagnostic mesh with a 10 km per unit height scale. Both choices are made up.

A numerical probe of the underlying field selected body-space direction
[-0.94452599, 0.21625, -0.24719747], height 2440.867 m and radial/normal
dot 0.999819946. Its dot to the drop was 0.090757. The set-space target is
[-6068014, 1588096, 1389277] m, and the eye is
[-6047719, 1768809, 1398784] m, about 32.5 km above the sphere. The target
is about 182 km from the eye; the same center ray reaches the analytic
sphere at about 198 km. The coarse mesh interpolates the sampled field and
is not guaranteed to pass exactly through the probed point.

Both images show green mottled surface and a curved atmospheric edge; their
small tonal differences do not establish recognizable relief, silhouette
changes or distinct cast shadows. This is a failed visual demonstration of
the desired ground, not investor-ready. No scene conclusion should be
inferred beyond this view. The saved jobs write PNGs and PFM raws under
`out/studies/2026-11-27-relief-target-{flat,10km}/`.
