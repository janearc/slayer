# Native regression run

Run the suite directly so Azazel can admit its jobs. It opens serial Metal
proof windows in background mode, leaving other authoring windows alone.
Its PNGs are widget exports from the native draw path.

```sh
bin/ferruccio regress --out /path/to/existing-output-folder \
  --examples ferruccio/examples --sets sets \
  --slayer /path/to/slayer --azazel /path/to/azazel \
  --azazel-queue /path/to/queue --evidence /path/to/existing-output-folder
```

Use an existing queue and worker. `--dry-run` starts no child. See
`bin/ferruccio regress --help` for optional source and runtime paths.

The baseline checks working views and held trackball gestures, plus full
stills for a small example and bounded terrain cases. `--inventory` adds
wire and bounded full probes for up to 20 additional set sources.
Bounded probes copy the input and set raster film to 160 × 100 and one ray
per side. Physical rigs are unavailable to those probes.

Each case retains its original and executed source, hashes, child log,
capture receipts and result. Read `summary.json` first. Unavailable inputs
and invalid or missing outputs fail the run. If a timed-out child cannot
confirm cleanup, later cases are skipped. A render can succeed even when
the working view lacks support for part of its scene.

## Working-view latency

Launch Ferruccio with `SLAYER_METRICS_ADDR=127.0.0.1:9401` and keep it open
while you measure. Stage the scene, then continuously drag the Orbit or
Travel control for several seconds. Drag a supported placement separately.
Scrape `http://127.0.0.1:9401/metrics` immediately before and after each pass
and compare the histogram counts, sums and buckets.

- `ferruccio_working_draw_attempt_seconds` measures host time inside each
  draw call. It includes polls that do not yet return pixels.
- `ferruccio_working_cull_seconds` measures host placement and visibility
  selection, excluding buffer preparation and geometry uploads.
- `ferruccio_working_upload_prepare_seconds` measures host-side mesh upload,
  camera uniform writes, and instance preparation.
- `ferruccio_working_encode_seconds` and `ferruccio_working_submit_seconds`
  measure command encoding and queue submission separately. These phase
  histograms record new draws, excluding subsequent readback polls.
- `ferruccio_working_frame_seconds` measures completed working frames from
  geometry placement through GPU readback.
- `ferruccio_working_motion_to_pixels_seconds` measures from the last
  accepted held drag update entering the transform handler to installing
  its working pixels in the viewer.
- `ferruccio_working_motion_frame_interval_seconds` measures time between
  completed motion frames within one drag. Pause-free drags make this a
  measure of the working view's cadence under input.
- `ferruccio_working_motion_coalesced_total` counts superseded updates.
  `ferruccio_working_readback_late_total` counts calls where pixels are
  still pending.

Use a fresh process for a cold scene and repeat the passes in the same
process for warm resident geometry. Record stage time and window dimensions
separately. These measurements end in the app, before window presentation.
