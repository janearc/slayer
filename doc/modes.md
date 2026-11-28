# Choosing how to render

Slayer offers several independent controls, rather than one science/video
switch. Choose the backend, sampling, physical model, lighting shortcuts
and development explicitly. A job records these choices. See
[flags](flags.md) for precedence and [optimizations](optimizations.md)
for the mechanisms behind them.

## CPU reference

Start with a small CPU reference, an explicitly stated physical model and
all optional lighting shortcuts disabled:

```yaml
renderer: cpu
quality: exact
vibes: off
fastlight: off
shortcuts:
  lamp-store: off
  sky-looks: off
  restir: off
```

This is a fragment to add to a job. Keep geometry, units, seed, time,
source data, samples and development fixed. Increase sampling and field
resolution to test convergence. Use independent analytic controls and
the `burning-trousers` harness for the quantity being measured. Retain
linear PFM output as well as a developed image.

For a fresh-frame control
in an ecosystem film, set `scene.film.sections: {keep: false}` in the set;
positive section drift is another approximation outside the umbrella flags.

`exact` means this configured tracer's answer. It does not remove finite
volume steps, the transmission stopping threshold, limited bounces or
single scattering. Nor does it promise CPU/GPU equality. A source-backed
input is not proof that the transport model describes every relevant effect.

`hillside -prove` replays 64 selected pixels through Fermat where supported;
it refuses lighting shortcuts. Replay checks implementation repeatability,
not physical completeness. There is no `--science` or `--strict` switch.
Signed-build verification is not an implemented end-to-end workflow here.
Read [flags and proof](flags.md) before describing a result as validated.

## Film preview

First get the composition right at 320 by 180, one or two rays per axis,
and a short duration. inspect the resolved duration and rate with
`--dry-run`. use Metal for supported scenes and retained sessions for
films.

for mist or smoke in a GPU set with a pinhole, use the lamp store
and shared sky looks.

```yaml
renderer: gpu
film: {preset: "720", rays: 2}
quality: exact
fastlight: on
shortcuts: {restir: off}
```

`quality: exact` draws a full frame with the lighting specified in the
job.
`fastlight` and `vibes` both request all three lighting shortcuts.
`restir` takes priority when enabled. use [ReSTIR](restir.md) for
direct light from many positioned lamps. the lamp store supports
up to sixteen lamps. see [flags](flags.md) for the full limits.

For dense meshes, compare `gpu-native` against `gpu` with shortcuts off
first. The fast-light modes themselves use native primary and shadow
traversal, even when the job says `renderer: gpu`. They therefore need
native ray-tracing support too.

cached smoke lighting also selects its own native mode. the regular
Metal frame path chooses ReSTIR, shared sky, lamp store, cached
lighting, native primary, native shadow or software, in that order.
compare the resolved modes when attributing a speedup.

set the final dimensions, frame rate and sound in the job. use
`outputs.movie` for an MP4 on macOS with cgo or `outputs.frames`
for a frame sequence.

## What each control actually changes

| Control | Meaning |
|---|---|
| `renderer` | CPU, Metal software or native mesh traversal |
| `quality: preview` | preview drawing policy; not a physical transport model |
| width, height, rays or samples | raster and per-pixel sample budget |
| `vibes`, `fastlight`, shortcuts | explicit approximate lighting policy |
| volume illumination model | legacy lamp response or normalized isotropic single scattering |
| density/source grids | spatial representation and interpolation |
| set detail and populations | the scene's represented geometry |
| `fantasy` | declared authored departures from measured/derived scene values |
| shutter glimpses | multiple scene instants per output frame |
| print or stock | development of computed light |
| caches and retained buffers | reuse under their invalidation contracts |

Shortcut resolution first checks the shortcut's own setting, then
`fastlight`, then `vibes`. At each level, CLI wins over job, then set.
Sets expose the two umbrella flags, not individual shortcuts. Consequently
an explicit job shortcut can override a CLI umbrella flag. To force a
direct reference, turn all three shortcuts off by name as well as the umbrellas.
`sky-looks` currently requires `lamp-store`.

## Increase speed without guessing

Measure cold setup, warm frame time, host encoding, baking, tracing,
finishing, development and movie encoding. Keep the picture fixed while
trying one change. A smaller picture is less work, not a measured
quality-preserving speedup. Cached frames must not masquerade as new traces.

For a film, examine fine silhouettes, narrow beams, shadow motion, mist
boundaries and temporal noise. State which model and approximation produced
the accepted result. The [optimization guide](optimizations.md) lists the
implemented savings.
