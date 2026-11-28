# Omnipositor

Omnipositor combines layers into finished frames. It provides a
[Slayer set adapter](SCENE.md) and a separate pipeline for Moonlake.

## Slayer sets

`scene PLAN --out NEW-DIRECTORY` composites a full-resolution scenery
plate with foreground transport. The plan chooses foreground resolution
and how often to refresh the plate. Reusing it freezes surface geometry
and lighting. Use `--reference` to draw the whole scene every frame.
See the [scene plan and supported inputs](SCENE.md).

## Moonlake

`stretch --from FRAME --to FRAME --settings FILE --out DIR --png`
processes a half-open frame range. The CPU pipeline runs cues, lighting,
ocean, matte, mist and development without writing intermediate images.
It verifies frame identity and PNG bytes before resuming existing output.

`--encode FILE --ffmpeg /absolute/path/to/ffmpeg` writes HEVC. The
encoder produces silence and refuses an enabled song setting.
`--experimental-gpu` uses Metal for selected layers and supports encoding
only. `--experimental-gpu-ocean` also uses the experimental Metal ocean.
`--trace FILE` records elapsed stage times including CPU and transfer
work. It does not measure GPU execution time.

Check `omnipositor scene --help` and `omnipositor stretch --help` for
options. The Moonlake settings resolve input paths relative to the
settings file. Use [Azazel](../../../doc/azazel.md) for scheduled rendering.

[Earlier comparisons and performance measurements](WORK-REMAINING.md)
record the tested scenes and remaining compositor limits.
