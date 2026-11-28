# Distant-disc shadow study

`flowing.sun_disc_degrees` specifies the full angular diameter of a
uniformly bright distant disc, from 0 to 10 degrees. The default 0 keeps
the prior single-direction hard shadow. A positive value requires multiple
world bodies and averages fixed directions over the disc using the existing
golden-angle `film.Sun.Spread` pattern, on CPU and GPU. Optional
`sun_disc_samples` chooses 8–64 directions, default 8; point-source shadows
still use one ray. The same deterministic pattern is used at every frame,
but flowing films are not yet supported and no temporal stability has been
observed. The visible fraction
modulates the prelit face at each surface hit; it is not a blur filter.
An optional separately authored `shadow_fill` retains its previous artistic
meaning. These study scenes set it explicitly to zero.

The disc is infinitely distant and has uniform brightness; its direction
and angular diameter do not vary with surface position. The giant's spectral
face is still computed for the central sun direction, so multiplying it by
visible fraction approximates the extended source's illumination, especially
as the disc grows. It does not integrate the spectral response separately for
each direction, model a finite star or stellar limb darkening, account for
volume shadows, or light ordinary scene bodies with this extended disc: they
still use the directional scene lamp. Other existing single-sun, opaque,
static-still and camera guards remain. Eight fixed samples can produce eight
discrete visibility levels in a partial shadow and may not look smooth in
motion; flowing films are still explicitly unsupported. Sixty-four samples
can require eight times the shadow visibility queries of eight samples per
surface hit; overall render cost depends on the scene. CPU directions are
prepared once per placement, while GPU directions are evaluated per hit.

NASA's [eclipse geometry explanation](https://science.nasa.gov/eclipses/geometry/)
distinguishes complete and partial obstruction; the existing source shelf's
`nasa-cassini-shadows-on-saturn` record pins a related Cassini observation.
These references motivate modeling visibility, not these invented numbers.
The comparison uses the same weather, camera, moon, exposure and artistic
split lamp balance as the earlier detailed-moon study; `penumbra-point` uses
0 degrees and `penumbra-disc` uses an intentionally enlarged **10 degrees**
full angular diameter (5 degrees angular radius) to make the transition visible.
That is an artistic source size, not a
solar-system measurement. Both use the close camera and 1024-square output;
CPU and GPU jobs retain unaltered PNG and linear PFM separately. A visual
assessment must check full shadow, partial visibility, clear region and any
sample stepping before the effect is characterized as successful.

Azazel jobs 424–427 rendered the matched point and disc scenes on CPU and
GPU with candidate `fc14bd1ac`; the jobs now write PNG and PFM under
`out/2026-11-27-penumbra-{point,disc}-{cpu,gpu}/`. An independent
visual researcher saw a hard black oval in both point-source controls. The
10-degree disc gives a smaller black center surrounded by a wider partially
shaded region on both backends. The GPU transition shows conspicuous
overlapping stepped ovals, consistent with the eight fixed directions; CPU
speckle makes its transition harder to grade. This is a visual demonstration
of partial visibility under the stated approximation, not evidence of
pixel parity, physical eclipse realism or animation stability.

## Denser fixed angular pattern

Azazel jobs 433 and 434 used the same GPU candidate `c334f1811`, scene,
camera, 10-degree full disc diameter and zero fill, changing only the
sample count from 8 to 64. The jobs now write PNG and PFM under
`out/2026-11-27-penumbra-samples-{8,64}-gpu/`. The independent
visual researcher found that 64 removes the conspicuous overlapping oval
bands, yielding a much smoother partial shadow with a clear dark core and
visible underlying weather. Some unevenness remains near the core. This is
one still with an exaggerated diameter; it does not establish animation
stability or realistic stellar geometry.

The two individual progress logs report approximately 23 seconds at 8 and
21 seconds at 64 samples. This one pair is not a controlled benchmark and
does not imply that 64 samples are cheaper; workload and shared machine
variation can dominate. Up to eight times as many visibility queries may be
cast per hit. Retain the original outputs and logs for context.

CPU job 435 rendered 64 directions with `c334f1811`; its job now writes PNG and PFM in
`out/2026-11-27-penumbra-samples-64-cpu/`. Compared with CPU job
426's eight-sample image, the independent researcher saw the dark core and
partial surround in both and a smoother edge at 64. Both images are heavily
speckled, limiting the comparison. The binary also changed from `fc14bd1ac`
to `c334f1811`, so the images do not independently isolate sample count or
establish CPU/GPU parity or temporal stability. Their individual progress
logs report approximately 16 and 57 seconds, respectively; they are single
shared-machine runs using different binaries, not a controlled benchmark.
