# Measured workload profiles

This package stores observations, not predictions or memory limits. The runtime
probe owner selects frames (heard supplies `Edit.ResourceProbePlan`), measures
them, and records evidence. Azazel owns admission and live pressure monitoring.
The package starts no render, changes no model detail and reserves no resources.

`Open(absoluteCacheHome, lockWait)` selects `workload-profiles.bolt` beside the
existing geometry caches. `Record(identity, samples)` atomically adds samples;
an identical attempt/sample retry is idempotent; changed bytes in that slot
return `ErrConflict` and roll back the batch. `Lookup(identity)` returns independent
decoded data. Every operation closes its database, so independent processes
hold file locks only for database work. Never render inside a transaction.

All five identity fields are mandatory lowercase SHA256 digests:

| Field | Hash input contract |
| --- | --- |
| Source | Full transitive input manifest with actual content digests, scene/recipe, seeds, detail, clocks and animation parameters. A picture key that omits nested inputs is insufficient. |
| Build | Actual executable bytes plus the measurement algorithm/version. |
| Device | Hardware/device, driver and OS identity affecting cost. |
| Render | Complete backend, raster, rays, quality shortcuts, lens, materials/volumes where not covered by Source, and CPU/GPU worker and pipeline settings. |
| Probe | Scene identity, exact requested timestamps, sample roles and scope of the measurements. |

The caller constructs these manifests and hashes them. Missing dependencies
cannot be repaired by this package; unknown provenance means a miss and a new
measurement, not a guessed key. Schema v1 has a separate key/bucket namespace.

`SourceManifest(ctx, absoluteRoots, Resolver)` provides the source adapter.

The format/scene resolver receives the actual opened file and must explicitly
report a complete dependency list, including implicit assets; unknown format,
dynamic input selection or incomplete scene knowledge must return incomplete.
The adapter recursively streams actual file hashes, preserves logical relative
path contexts, sorts/deduplicates metadata, and refuses missing files, cycles,
inspection-time mutation and unknown dependencies. It returns no partial key.

It does not infer completeness from `job.Inputs` or implement every scene's
resolver. A runtime owner must supply that scene knowledge and reverify the
manifest before execution; a source graph is a snapshot, not a file lock.

Heard's `Edit.WorkloadProbePlan(job.Times())` adapts the exact cut selection to
`workload.Probe` without rendering. Coincident roles in short shots stay on the
deduplicated frame; nanosecond timestamps are preserved. `Probe.Key()` hashes
the exact frame/role plan. These adapters do not execute startup probes.

Record cold process, disk-hit and retained moving-frame observations separately,
with their real cache descriptions, timestamps, run and frame identifiers.

`Run` is the scheduler's attempt ID; `Sample` is a stable role/repeat ID within
that attempt. The runtime owner must fence the queue attempt before calling
`Record`. This database cannot validate a queue lease, stop an old process or
authorize publication; old evidence never overwrites a newer observation.

Optional counters are nil when unavailable. CPU time, GPU execution and wall
time have different meanings; overlapping stage durations are not additive.

Physical footprint, lifetime physical peak, group RSS, native allocation and
scratch remain distinct. `Peaks()` returns independent observed maxima;
it does not sum aliases or claim these maxima cover unobserved animation.

`Estimate(profile, inputs)` is a pure preflight calculation for an exact
profile identity. It requires nonempty valid observations, a matching expected
identity, host capacity, current host use, positive additional independent
process count, positive frames-per-process context, explicit headroom and a
whole-process lifetime physical peak.

Current physical footprint alone is
reported but cannot produce a fit status because sampled current use is not a
high-water mark. The calculation is:

```text
current host use + (lifetime physical peak × additional independent processes) + headroom
```

Current host use must already include all work that is running; the process
count is only new independent copies. Frames and workers in one process are
reported as an assumption and never multiplied.

The supplied profile must
match the exact source, executable/build, device, render settings and
concurrency for each process being added. `PhysicalPeak` may include earlier
work in that process, making it conservative for the sample but not an upper
bound. `Physical`, group RSS, native and scratch remain independent
measurements: they are exposed in the report but never added together.

Missing
metrics are listed; if the lifetime physical peak is absent, the estimate
status is unknown even if current footprint exists. Arithmetic overflow also
returns unknown.

The statuses `within_capacity` and `over_capacity` only compare the stated
inputs and observed peak.

They are evidence-based advice, not admission,
reservation or a safety guarantee. The estimator does not account for child
or group processes, unobserved frame peaks, external host use changes or
sharing between separate processes.

It never enables optimizations, changes
quality settings or authorizes rendering. A real preflight caller still needs
to resolve the exact profile identity and obtain current host capacity/use;
unknown provenance or a missing profile must remain unknown.

The `workload-preflight` command reads a complete identity JSON file and an
existing profile database. A usable identity file has all five lowercase
SHA256 fields, for example:

```json
{
  "source_sha256": "<64 lowercase hex digits>",
  "build_sha256": "<64 lowercase hex digits>",
  "device_sha256": "<64 lowercase hex digits>",
  "render_sha256": "<64 lowercase hex digits>",
  "probe_sha256": "<64 lowercase hex digits>"
}
```

Run `game run workload-preflight -- report` with explicit `--cache-home`,
`--identity-file`, `--capacity-bytes`, `--current-use-bytes`, `--processes`,
`--frames-per-process`, and `--headroom-bytes` flags. All numeric values are
unsigned integers; memory values are bytes.

The command opens the profile
store read-only, emits one JSON report, and never creates a missing cache
directory or database. It cannot derive complete workload hashes from an
ordinary job path: the caller must supply the exact identity produced by a
runtime that knows all transitive dependencies and render settings.

A missing
profile, invalid identity, or corrupt store is reported as `unknown`.

For example, after writing a real identity manifest and gathering current
host values:

```sh
game run workload-preflight -- report \
  --cache-home /absolute/cache \
  --identity-file /absolute/workload-identity.json \
  --capacity-bytes 68719476736 \
  --current-use-bytes 34359738368 \
  --processes 1 \
  --frames-per-process 8 \
  --headroom-bytes 8589934592
```

Do not replace a prior cold peak with a smaller warm measurement. Use profiles
as estimates, validate current pressure and swap trend before increasing work,
and update from real frames when a scene exceeds the startup samples. A cached
peak is neither a hard cap nor permission to start another full world.

Values have content digests and are copied out of read transactions. Corrupt
payloads, identity mismatches and invalid observations return `ErrCorrupt` or a
validation error; a caller must not admit work from a partial corrupt profile.
No arbitrary disk or process-memory cap is introduced here.
