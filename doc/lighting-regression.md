# Lighting direction checks

The summit's apparent backwards lighting came from two sources. The Moon
backlit the mist, while a Sun below the opposite horizon still added warm
skylight to the front of the dome. Raising the exposure amplified both.

The three Moon-only night recipes now omit the Sun block. The renderer
already supplies the full-Moon spectrum without it. Dusk scenes can still
include solar twilight deliberately; the ordinary reader keeps that option.

The new input guard failed on all three original night recipes. After the
repair, all three pass and a daylight control still retains its Sun. The
[before-fix log][before] records the actual failures.

The independent rendering references use small, controlled scenes. White
Lambertian paint under unit normal irradiance must return 1/pi; a light
behind the face contributes zero. A separate uniform blue sky can fill that
face without turning the red direct light around.

A ceiling blocks an overhead light from a mist puff, but cannot block a
light below it. The camera's horizontal ray misses the ceiling. CPU, Metal
and native Metal each pass these references against the known expectations,
using triangle meshes rather than comparing their images with one another.

The [focused run][after] passed through Game with two Go workers, serial
packages and no skipped backends. It took 4.82 seconds including the
wrapper, with a sampled process-tree peak of 290832 KiB.

Fermat remains a CPU replay check. It can faithfully reproduce a bad setup;
it does not replace these independent direction and scene-intent checks.

Sky illumination of mist remains Light-1. Surfaces sample the environment,
while the current mist source term uses explicit lamps. This patch fixes
the night inputs and adds regressions; it does not change that transport.

To repeat the focused checks:

```sh
game test -- -p=1 -parallel=1 -count=1 \
  -run='IndependentLighting|MoonOnly' ./gpu ./cmd/hillside
```

The linked logs are companion local receipts beside this checkout.

[before]: ../../lighting-regression-checks/night-before-v2/check.log
[after]: ../../lighting-regression-checks/lighting-after-v3/check.log
