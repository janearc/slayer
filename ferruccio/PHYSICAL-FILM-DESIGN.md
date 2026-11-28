# Physical film choices

The producer has two different concepts. `job.Presets()` names output pixel
sizes. A set's `scene.film.frame` says the physical frame in millimetres,
and its rig also requires focal length, f-number, focus target, shutter and
glimpses. The gauges in `film/lens.go` provide named physical dimensions;
they are not output-size presets. A job's `stock` chooses development through
`ninjafilm`, independently of the frame, and conflicts with job `print`.

For a rigged set, `session.rigged` derives height from pixel width and the
physical aspect ratio, and refuses a job height or preset with an incompatible
shape. It also refuses a job exposure or field of view overriding the rig.
Changing gauge therefore changes the view and image dimensions, and cannot be
implemented by changing a label or selecting the nearest output-size preset.

Full-render preparation derives height from a standalone or referenced rig's
physical frame, and refuses an incompatible selected output size using the
producer's one-row tolerance. Azazel job 119 rendered a job whose requested
24 × 15 size resolved to a 24 × 16 image.

The shared working view uses a
labeled pinhole approximation of the rig at the focused lens’s actual angle.
It does not show depth of field, motion blur or shutter exposure. The older
direct display path still refuses physical rigs.

A set with an existing complete rig now offers named physical frames from
`film`; it saves and undoes the frame without changing the lens or shutter.
Jobs without an authored print curve offer the producer’s development stocks;
changing a stock keeps its seed and rating and is undoable. The same Azazel
job used the soot stock with an explicit zero seed.

The editor still needs the full rig to be authored and validated together,
explicitly explain these effects, derive or validate the frozen output size
using the same producer rules, and preserve the stock/print exclusion. Its
working view must either support the rig or clearly show why it cannot.

A submitted render and its receipt must identify the exact frame, lens, shutter,
stock and resolved pixel size; tests should check real producer interpretation
and a queue-authorized image rather than the dropdown alone.

## Full-render estimate and evidence

Read the checked film settings once when opening render setup and resolve each
request against its real dimensions. A job's explicitly authored dimensions
and sampling override its referenced set. The physical frame determines the
final height at the requested width. Check a requested height with the same
one-row tolerance as the producer. Refuse exact totals for rigs before queue
submission. Keep the ray grid for existing sources.

The estimate counts width × resolved height × spatial samples × shutter
glimpses. Keep the spatial count per glimpse separate from total primary work.
The receipt records the requested frame, lens, focus target, shutter, glimpses,
and job stock with any explicitly authored seed or rating, beside the verified
PNG dimensions. For a referenced set, compare its source hash to the owner's
file fingerprint before accepting pixels for those settings.
