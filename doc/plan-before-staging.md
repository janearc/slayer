# Planning a set before staging

```sh
slayer assess SET --plan --width 480 --height 270
```

the first-frame plan reads a terrain set and its height data, then
reports each thing and stand’s bounds, projected size, estimated
count and verdict. it measures from recipe extents where available.
an unavailable extent falls back to a build verdict. `--camera` and
`--at` select the view and time for the report.

the report is advisory. ordinary draws stage the authored scene.
the separate `SLAYER_TRIAL_PLAN=1` experiment omits out-of-view
things and stand members. that changes the transport: offscreen
objects can still cast shadows, reflect in the scene or illuminate
visible surfaces. use the experiment to measure staging cost with
a same-build reference that keeps the full set.

## Recorded workload

the Futian shore study staged 108,083 things, 703,667 instances,
4.3 billion referenced faces and 5,161 lamps. one run reached about
14 GB of process memory and spent about two and a half minutes before
tracing. its 16 by 9 render took 2 minutes 48 seconds overall. the
Dark City tiling study peaked at 13.4 to 14.2 GB from 2 by 2 to 16 by 16
tiles because each tile staged the set again.

the Futian planning report counted 30,995 entries:

| Frame | Build | Average | Drop |
|---|---:|---:|---:|
| 16 by 9 | 182 | 14,744 | 16,069 |
| 480 by 270 | 14,865 | 61 | 16,069 |

these are planning counts from the recorded studies, not the savings
of a renderer applying every verdict.

## Design for selective staging

a complete plan would classify each entry as build, aggregate or
omit. it would use conservative bounds, seeded placement, projected
detail and a recipe’s estimated coverage and light. aggregate
representations would preserve average coverage and light and need
comparison against a full-detail reference.

the proposed occlusion pass visits the set’s hierarchy near to far.
coarser cells cover distant nodes and store remaining transmission.
light and reflection paths need their own coverage checks before
a node can be omitted. the [Scrooge design](scrooge.md) describes
the corresponding node, footprint and memory policies.
