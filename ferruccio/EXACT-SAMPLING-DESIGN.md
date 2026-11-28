# Exact samples per pixel: integration work

The existing `rays` contract is a side length: a value of 4 traces 16
samples per pixel. Existing jobs, CLI overrides, region renders and retained
frames must keep that meaning. An explicit total needs its own field and
precedence rules; it must never be converted to an approximate square.

`trace.Exact` is a CPU primitive for a total count. It traces precisely the
requested film sequence points and takes their linear mean, retaining nearest
depth. For an arbitrary count the legacy grid filter uses a box distribution;
the explicit box, trapezoid and tent filters retain their shapes. The original
`trace.Sampled` and the old grid remain unchanged. `trace.ExactKept` can stop
between samples and retain completed rows; its caller must key those rows by
the exact count and every other input. Synthetic tests cover its row scheduling
and averaging. CPU still jobs now carry `film.samples` or `--spp` through
the session; the render sheet offers those counts on CPU and both GPU backends.
Azazel job 82 completed a 16 by 12 CPU smoke at seven samples. GPU job 83
completed the same request and produced an identical PNG. Jobs 84 and 85
rendered the playground on CPU and GPU at seven samples: the mean absolute
RGB difference was 0.0044/255, with a maximum difference of 2/255. Jobs
86–91 completed GPU smokes at 1, 4, 7, 16, 25 and 64 samples; the logged
completed sample-record counts were 192, 768, 1344, 3072, 4800 and 12288,
respectively, for 192 picture pixels. These are bounded examples, not a
claim of parity for every scene or material. Native GPU uses the same
per-pixel rays with Metal mesh traversal. Rig, region, wire, film and
accelerated lighting paths refuse exact totals.

The original GPU path expands both image dimensions by a side length. The
exact still path instead dispatches `width * count` samples across each
picture row, supplies the corresponding film rays, and averages each pixel's
consecutive samples. Shutter timing, sky sharing and retained/region paths
still use square addressing and are refused for exact totals.

Dispatching the next square and discarding extra samples would still trace
more than requested and would misstate costs.

For supported stills the render sheet exposes 1–64 total samples per pixel.
See [full still rendering](internal/preview/FULL-RENDER.md) for invocation,
queue routing and receipt behavior. Extending support to other drawing paths
requires those paths to honor the exact total.

Square-count sky comparisons do not prove parity for materials whose
footprints depend on sample width. The CPU primitive uses the finished
pixel's angular width for each sample; that and GPU behavior need explicit
material and depth-of-field comparisons before claiming full parity.
