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

Full-render preparation for an open set now derives height from the rig’s
frame, and refuses an incompatible selected output size using the producer’s
one-row tolerance. A producer staging test verifies the derived dimensions;
a rigged queued render has not yet been run.

The shared working view uses a
labeled pinhole approximation of the rig at the focused lens’s actual angle.
It does not show depth of field, motion blur or shutter exposure. The older
direct display path still refuses physical rigs.

A set with an existing complete rig now offers named physical frames from
`film`; it saves and undoes the frame without changing the lens or shutter.
Jobs without an authored print curve offer the producer’s development stocks;
changing a stock keeps its seed and rating and is undoable. A job producer
read and frozen input test verify the choice; no new stock render has run.

The editor still needs the full rig to be authored and validated together,
explicitly explain these effects, derive or validate the frozen output size
using the same producer rules, and preserve the stock/print exclusion. Its
working view must either support the rig or clearly show why it cannot.

A submitted render and its receipt must identify the exact frame, lens, shutter,
stock and resolved pixel size; tests should check real producer interpretation
and a queue-authorized image rather than the dropdown alone.
