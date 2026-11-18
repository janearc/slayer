# Adding guide beams last

Two complementary passes separate direct beam scattering along the camera
ray. WithoutPrimaryBeams keeps the rest of the ordinary picture. Its
surfaces, their beam illumination and secondary paths remain complete.
PrimaryBeams returns only the camera ray's direct beam scattering.

Here a beam is the finite Gaussian column represented by `emit.Beam`.
The polar pair `emit.Beams` is a different source and stays in the base.

Both passes keep the same geometry, density, extinction, volume order,
beam refinement and lamp-selection weights. Every volume attenuates both
contributions. A foreground surface stops the beam pass. Neither pass adds
a visible beam skin in vacuum.

At the same camera, time, resolution and samples, their linear RGB sum
must reconstruct CompletePicture within float32 arithmetic error. Add
before exposure, bloom and tone mapping. Beam-pass coverage is geometric
metadata, not an alpha by which to blend the light again.

The source selector is local to primary transport. It excludes nonbeam
lamps, sky and emission from PrimaryBeams; it excludes only direct beam
scattering from the complementary primary transport. Lamp selection still
uses the full scene, so sampled-light weights match the complete draw.

CachedLighting's source grids contain an approximate sum of mixed light.
The two beam passes refuse that mode: a direct source split cannot promise
exact reconstruction of an approximate mixed sum. Per-lamp visibility
stores remain supported. No cache is relabelled as a separated source.

Check reconstruction with mixed source colours and volume emission,
beam-only and no-beam scenes, more lamps than the sampler's capacity,
and coloured overlapping extinction. Check foreground blocking and vacuum
independently. Exercise direct and cached/native modes on actual Metal.

This separates transport contributions; it does not make a low-resolution
fog reconstruction exact. Searching will initially draw both at its full
output resolution. Any later reduction needs its own image/error and
timing comparison.
