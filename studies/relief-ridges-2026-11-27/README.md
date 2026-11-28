# Ridge height: matched visual study

Jobs 304–307 used experimental Slayer build `c4fe86832`, with 256 by 128
displaced mesh, 10 km base height scale, thin authored atmosphere, and cloud
cover zero. The sole intended difference in each camera pair is ridge scale:
zero versus an exaggerated 30 km. This is an artistic scale, not a claim
about plausible terrain. The ridge contribution is tapered to zero at sea.
The YAML relief wiring is experimental and is not a supported constructor.

Both views use seed 1, fixed lighting and exposure 0.003, 800 by 800 pixels,
and 2 rays. The `grazing` target was chosen from a measured low-sun terrain
sample; it is not necessarily a ridge maximum. The saved jobs write
`proof.png` and `light.pfm` to these directories:

- `out/studies/2026-11-27-ridge-0-orbit`
- `out/studies/2026-11-27-ridge-30-orbit`
- `out/studies/2026-11-27-ridge-0-target`
- `out/studies/2026-11-27-ridge-30-target`

Independent full-size inspection found a brighter angular protrusion at the
lower limb in the 30 km orbit image, but the planet is dark and neither
mountain chains nor a discrete cast shadow can be confidently identified.
The target pair looks nearly identical, with fuzzy green ground and no
recognizable ridges or silhouette change. These frames show a change, but do
not yet demonstrate compelling or clearly resolved terrain. The cause is not
established by this comparison.

The saved jobs reference the adjacent sets. Their checkout-local outputs
are separate from the November 27 render archive. Reconstruct the
experimental binary from its pinned commit to reproduce the study.

## Attempted ridge-target view

Jobs 317–318 aimed the camera toward a 30 km field probe at direction
(-0.36995004, -0.92775, 0.04916202). Both images are nearly featureless
blue-green, and no ridge can be recognized. The measured-view jobs write
frames and raws to `out/studies/2026-11-27-ridge-{0,30}-measured/`.

This does not establish the behavior at the probed ridge: the probe built a
standalone 24-hour planet with a single sun, whereas the rendered planet is
locked and generated with a binary. Its direction is body-space and has not
been mapped through the actual rendered body's transform. The probe's field
and the render's field also cannot be assumed identical. A valid targeted
study needs the actual constructed body's field and transform.

## Staged mesh target

The experimental `probe-relief` stages the same two jobs and compares
their actual mesh vertices in scene coordinates. It selected vertex 6087,
point (525101.215335, 5355873.584442, -3539946.413977), with 13724.74
scene units of displacement, a 0.21328 radian normal change and radial
sun cosine 0.09208 relative to `pair primary`. The source is selected by
uniform surface glow, not total received illumination. This is a sampled
mesh vertex, not a maximum of the continuous height field. The probe build
was `ab389b7ba`; the original paired render build remained `c4fe86832`.

Jobs 320–321 aimed at that point from (383969, 5453657, -3594902),
with the same camera, lighting and 0.003 exposure for both. Their originals
are nearly black. The staged-view jobs write frames and raws to
`out/studies/2026-11-27-ridge-{0,30}-staged/`. Identical
standalone developments from the retained raws used knee 0.8, top 4 and
either +6 or -6 stops, in the matching `-staged-print6/` and
`-staged-printminus6/` frame directories. The +6 prints clip almost white;
the -6 prints reveal dark mottled green and a blue edge, but independent
inspection could not distinguish a ridge or confidently tell the variants
apart. These images do not establish visible terrain relief.

A third matched development (jobs 326–327), knee 0.8, top 4, -3 stops,
revealed the green mottling more clearly without revealing identifiable
ridges or a difference, according to independent inspection. The November
27 archive includes the matched -3-stop prints.
The two raw PFMs are byte-identical: both SHA-256
`1ae169c54a362263b43d5aaa09c5dfd4877183b5da73c868566828408cc7de98`.
Both -3 prints also match: SHA-256
`a3ac8bd1fc17b97cd81caa0695e2dd1f6e232885c4daceb92140dce5904e473b`.
Logs report 800 rows drawn and zero rows taken for each original render.
This rules out resumed rows for these frames, but does not explain why the
raws agree or establish which surfaces the rays encountered.

## Corrected target and neutral-material comparison

The earlier camera mistakenly supplied world coordinates as set camera
coordinates. `set/shots.go` maps set (east, north, up) to world (x, z,
-y). After applying the inverse map, probe build `655e1061b` measured the
camera centre ray's first mesh hit within 0.565 scene units of the selected
vertex. The original camera hit over nine million units from it. Jobs
337–338 render the corrected pair with build `c4fe86832`. The saved jobs
write raws and frames to `out/studies/2026-11-27-ridge-{0,30}-corrected/`.

The 30 km variant is brighter and more uneven, but both originals are dark.
Identical -3-stop developments (knee 0.8, top 4) in the corresponding
`-corrected-print/` directories show blue-grey mottling but no recognizable
ridge or cast shadow, according to independent inspection.

For jobs 347–348, experimental diagnostic build `b309aabe9` replaces the
painted rock material with a made-up neutral 0.5 grey diffuse material;
the matching sets, mesh, camera, light and air remain the same. Raw and
frames are in `-matte/`; the same -3-stop developments are in
`-matte-print/`. The 30 km image has brighter, broader smooth shading, but
neither image shows recognizable ridges, silhouettes or discrete shadows.
This material override is diagnostic code and is not a constructor option.
