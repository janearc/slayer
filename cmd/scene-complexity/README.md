# Scene complexity

`game run scene-complexity -- report JOB.yaml` opens a checked Slayer job,
constructs its scene, and writes one JSON line for frame 0. The row includes
the exact scene time and the frame count.

Select other sample times with `--frames first,last,12`; `--all` visits every
frame time. An all-frame report can be expensive for long films because each
requested time builds its actual scene state. Jobs can be supplied together;
a failed job or sample remains visible as an error row and the command exits
unsuccessfully.

Set, smoke, named and authored Heard jobs use hillside's existing open and
construction paths. Heard `listening` and `continuous-field` scenes require
data-backed adapters that Session does not yet provide, so Open refuses them.

Session also refuses smokes with bands, a gauge or a palette; these show up as
error rows. The report does not call `Frame`, trace rays, render an image or
open a Metal device. A renderer may be selected by the job, but its GPU drawer
remains lazy because no draw occurs.

Every session is closed after its rows are written.

`geometry` comes from `thing.Storage`: it counts actual `World.Things`, unique
mesh and group storage by identity, shared instances, faces, vertices,
hierarchy nodes, and known shape-owned bytes. `volumes`, `fields` and `lamps`
are separate top-level counts. Unknown shape implementations are listed in
the geometry row.

Constructor open time and sample time are reported separately, along with
cumulative Go `TotalAlloc` deltas around each operation. Those allocation
deltas are process-wide counters sampled before and after the operation; they
are neither isolated per-goroutine attribution nor peak memory.

Every row records the SHA-256 of the job bytes observed immediately before
and after `job.Read`, the checked job key, the input paths that `job.Read`
recognizes, and the command's build stamp. The two reads detect ordinary
changes across parsing; they cannot rule out a concurrent replacement and
restoration during that interval.

Empty build-stamp fields are written as `unknown`. The job-file hash covers
only the job file.

The job key includes the dependency hashes tracked by `job`, but does not
necessarily include assets loaded transitively by a named scene or by a
referenced set. `input_paths` lists recognized paths, not their hashes; use
the job key only for the closure documented by `job`, and do not treat it as
a complete asset snapshot.

The `build` object keeps the Game version/time/builder/session fields separate
from Go's embedded version and VCS revision/modified settings. Each field is
`unknown` when its producer did not embed it; a VCS modified value of `true`
means the Go build saw a modified source tree.

These are current scene counts, not peak memory or a working-set estimate.
They exclude allocator overhead, materials, media payloads, textures, source
data, caches, packed geometry, native acceleration structures, GPU buffers and
frame images. A set's changing scene is sampled at the job's exact frame
times; the default first-frame row does not stand for the whole film.
