# Lighting census

This is an opt-in diagnostic harness. A normal build or test does not render.
It enumerates the registered smoke scenes and seven configurations: a CPU
reference, GPU software traversal, native shadows, native meshes, cached smoke
lighting, lamp-store lighting, and lamp-store with shared sky looks.

Each case is recorded separately. A renderer error is kept; no other renderer
replaces it. The two stored-light modes and the cached smoke-light grid are
explicit approximations. GPU direct lighting still uses production density
baking and is not an exact CPU implementation.
See [the inventory](INVENTORY.md).

This first matrix covers numbered smokes, not every set or named scene. It
does not exercise the independently authored city placement, vacant-lot
index or Heard anatomy-sharing controls described in [render optimization
controls](../../doc/render-optimization-controls.md). Those axes and any
as-yet-unimplemented control combinations remain unmeasured.

Do not run this beside active renders. Request Game's GPU lane for any GPU
mode. The harness verifies the inherited Game GPU mark names a normal lane
file that is locked; the mark and lock are not proof of lock ownership. CPU
alone may use the tests lane. Set a fresh absolute output path whose parent
already exists: the harness refuses any existing directory. For example, from
the Slayer repository and only when the GPU is available:

```sh
GAME_HOME=/path/to/your/home GOMAXPROCS=2 GOFLAGS='-p=1 -parallel=1 -timeout=60m' \
  game5 lane gpu -- env SLAYER_CENSUS_RUN=1 \
  SLAYER_CENSUS_OUT=/absolute/new/census-directory \
  SLAYER_CENSUS_SCENE=9 SLAYER_CENSUS_MODE=gpu-fast-lighting \
  go test -tags draw -count=1 -run '^TestLightingCensus$' ./tools/lighting-census
```

`SLAYER_CENSUS_SCENE` is a smoke number or `all`; `SLAYER_CENSUS_MODE` is one
of the seven names in `Modes`, or `all`. Unspecified selections mean `all`.
Each mode draws the first scene time once cold and once again at the same
time; the latter is a retained cache case, not a moving-camera measurement.

`SLAYER_CENSUS_TIMES=0s,1s,2s` adds distinct scene-clock samples after the
repeat. The smoke's camera remains fixed. CPU also repeats the first clock;
the default all-scene matrix schedules up to 728 visible frames before
spectral bands and extra clocks. Unsupported and failed cases reduce that
number. Use separate explicitly selected runs to schedule manageable jobs.

Defaults use smokejudge's full real-picture size, 1074 by 600, doubled for
worlds containing gravity fields. Each smoke keeps its registered samples per
side, field of view, world, and input requirements. Width and height can be
overridden only together using `SLAYER_CENSUS_WIDTH` and
`SLAYER_CENSUS_HEIGHT`; the chosen values are recorded.

Overrides are diagnostics and do not establish full-size quality. The tool
never lowers a smoke's registered sampling.

The initial manifest records every planned scene/mode with `not-run` status.
Each selected case has an atomic `result.json` checkpoint before work and
after every saved frame or band; an interrupted case remains `running`.

A final `summary.json` counts statuses. Failures make the command fail;
unsupported or incomplete cases are not passes.

Each case records the binary digest, available Go build settings and declared
input hashes. Missing source revision, dirty state or OS version are
explicitly unavailable. Those hashes do not establish a complete dependency
manifest, and input hashes are rechecked only for the files declared by the
smoke.

Atomic replacement preserves a completed checkpoint across ordinary process
interruption. Image files and containing directories are not fully synced for
an operating-system crash or power loss. Reports compare references within
one output root; a single-mode invocation has no same-root reference unless
that mode is itself the reference. Use `MODE=all` for a self-contained scene
comparison; its CPU reference currently occupies the acquired GPU lane.

Each frame has raw linear RGB in PFM and top-left row-major little-endian
float32 coverage/depth pairs in `.cd32`. PNG files are unit-exposure
visible-light diagnostics; the authored tone, gauge, palette and spectral
composite are not applied. CPU reference cases additionally save each
prescribed non-visible band with its own duration. GPU public APIs do not
expose those bands, so such GPU cases are marked partial. No movie is made.

See [the report API](REPORT.md) for matched-frame comparisons and limitations;
completion of a render alone never certifies fidelity.
When results are interpreted, report exact and approximate winners separately;
include the measured image differences for each approximate choice. A measured
case is not a fidelity pass until an appropriate explicit budget and temporal
comparison have been evaluated.

The harness checks the output filesystem before scene construction, before
each visible or band render, and again before writing. It stops the entire
run when free space cannot be read or would fall below the shared 30 GiB floor
plus a conservative one-frame output estimate. The check cannot reserve space
against concurrent writers. It never deletes artifacts, even after a failure.

Heap figures are Go heap snapshots; reported RSS is the lifetime peak for the
whole test process, not per-frame or per-case memory. The GPU profile records
device name and stage timings where available. Command-buffer durations can
overlap and cannot be added to wall time.
