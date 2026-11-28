# Memory stages trial

Set `SLAYER_MEMORY_STAGES=1` to write process-memory observations as JSON
lines to standard error. With the variable unset or empty, the checkpoints
do not sample memory or write output. This is an opt-in diagnostic; it
does not change the scene, impose a memory budget, or attribute bytes to
particular objects or Metal allocations.

Each line has `event: memory_stage`, a stage, PID and timestamp, the Go
heap's allocation, in-use, mapped and released bytes, Go system bytes
and GC count, and the current and lifetime-peak physical footprints with
separate availability flags. The Go and physical samples are taken at
different instants. These figures overlap and must not be added together.

A lifetime peak is the maximum since process start, not the maximum of a
stage. Physical footprint is an OS process measurement, not a breakdown
of GPU or Metal driver bytes. Unknown native values have a false flag.

The sampler does not force garbage collection; it can affect timing,
memory and scheduling. Concurrent draws share one process: PID and time
do not identify a particular frame, so use an isolated serial run when
attribution matters.

The retained renderer reports checkpoints around draw, scene encode,
pool copying and packing, discard, bake, native-plan creation, native
build or reuse, Metal grid open, and closing retained state. The per-call
path also reports its native plan and build. Not every checkpoint fires:
reuse, software traversal, per-call paths and failures take different
branches.

ReSTIR uses the shared encode checkpoints. acceleration and
reservoir work have no separate checkpoints here. scene construction,
animation, development and file output are outside these observations.

A CPU-only smoke check that exercises real pool packing is:

```
SLAYER_TEST_PACK_TRACE=1 SLAYER_MEMORY_STAGES=1 \
  game test -- ./gpu -run '^TestMemoryStagesObserveScenePacking$' -count=1 -v
```
