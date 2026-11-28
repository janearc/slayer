# Staged view across a process boundary

The normal Ferruccio frontend is built without Cgo. Slayer staging uses Cgo
and reads cache and shelf settings through process environment. A staging
helper can be a separate, Puzzlemouth-commanded child, with the resolved
settings set on that child alone. The editor passes its current unsaved buffer
and original path; the child stages but does not render, returns a
display snapshot, closes the Slayer session, and exits.

The parent never modifies its own process environment. It owns cancellation
and reaping.

The active snapshot carries shared meshes and authored batches, pinhole camera,
film size, focus, staged authored anchors, and notices. Supported authored
things can be selected and edited using displayed triangles. Selecting a
generated member identifies its authored stand; it does not create an
individual placement.

For terrain stands with an explicit area and a grid,
lattice or scatter habit, Region edits the four area bounds in metres from
the terrain southwest. Staging regenerates the stand, so members can move or
change and outside-terrain regions can yield fewer members. In Move mode,
arrow keys shift both bounds on one axis by 0.25 m, or 1 m with Shift. Each
key press is one undoable edit. Other generated groups and proxies remain
read-only.

An explicitly positioned v2 ground thing, including one in a set that leaves
`setting` unsaid and therefore defaults to ground, uses
its staged anchor for drag and turn; a move is re-anchored to the real terrain
after release. Turning a ground thing without an explicit authored heading is
refused because its initial heading was generated.

Rigid explicit terrain
things can also be scaled around that anchor; attached lights follow while
keeping their power. Terrain-founded villages, fire, burning things, special
stage rigs and deforming geometry remain unsupported for scale.

Coordinates edits east, north, height and an authored heading together in
one undoable change. Terrain coordinates are checked against the current
window; a generated terrain heading remains generated and is not offered
as a numeric value.

Add accepts checked constructor and profile placements on v2 terrain,
including the default-ground form. It
uses map east/north coordinates, height above ground and an explicit zero
heading. Coordinates outside the current terrain window are refused before
the edit; the default is the selected terrain object or the window centre.

Private renderer hit targets do not cross the process boundary. The receiver
rejects malformed indices, non-finite values, invalid dimensions, incomplete
or extra messages. Neither an old
snapshot nor a failed helper may be presented as current. The UI continues to
show all omission and approximation notices.

The initial wire limits of 64 MiB and one million vertices refused the real
city even after staging had succeeded. The transport now writes the snapshot
to a temporary file and validates its structure without those arbitrary size
limits. Version 5 adds staged authored anchors to version 4's binary shared
mesh and placement stream; older version 4 snapshots remain readable, and a
distinct signature prevents an older reader from misreading new metadata.

Individual batches and the complete in-memory scene can still be large.
Source buffers are limited to 8 MiB. A process failure leaves the old view
explicitly stale. A cancelled process is always waited for before return and
its temporary files are removed.

A separate shared-instance prototype keeps each native mesh once and carries
its staged placements and authored group references. On 25 November it staged
the city in 1.21 seconds with 22 meshes and places. It staged Futian in 64
seconds and extracted 32,509,431 placements of 79,796 meshes in another 41
seconds; that prototype omitted its infinite sea.

The current shared path adds a clearly labeled finite mean-level plane over
the finite scene bounds; it does not reproduce waves or shoreline. The
streamed shared path is now connected to the editor.

The working view shows opaque infinite planes as read-only, flat-colour
patches. Each patch lies on the staged plane around the captured camera's
projection. Its half-width ranges from 25 metres to 2 kilometres with
camera height above or below the plane. Moving the inspection camera can
expose the finite edge. The working view reports omitted transmissive
planes, planes with a spectral material and planes outside its coordinate
precision. Full renders use the original analytic surfaces and their materials.

A version 3 run delivered
Futian through the helper and loader in 5 minutes using a 17 GiB temporary
file. It omitted one sea, 76 volumes and 6,044 lights. The GPU batches objects
that share a mesh; a city working frame passed Metal tests.

Two Futian GPU attempts missed the existing two-second readback deadline,
including one with conservative frustum culling. Batch readback stays in
flight across UI ticks and cancels when the scene or camera changes.

A later headless Futian Metal run produced one 320x180 working frame: 137
seconds including loading, with a 15.24-second first GPU draw and a 4.30 ms
center selection. The sea was
the labeled mean-level proxy; volumes and lights were omitted.

This is not a responsive native-window acceptance. Binary mesh and placement transport has
also passed the city and protocol tests.

The first Futian probe on 25 November was stopped after about 84 seconds: the
helper had grown past 4 GiB while constructing stands and had not produced a
view. No renderer was started. An output limit alone does not bound staging
memory.

A temporary 1 GiB resident high-water watchdog sampled every 50 ms
stopped the helper in subsequent probes. A second Futian
probe stopped itself at 1,036 MiB after 29 seconds; a repeat after moving the
watchdog around input, staging and serialization stopped at 1,026 MiB in 29
seconds. These are observed overshoots, not a hard allocation ceiling. The
parent waited for both children.

The guard exits the helper without normal
session cleanup, leaving the operating system to release its resources. New
managed requests use a private temporary cache under the configured cache and
remove it only after the child exits; they do not reuse shared cache entries.
A crash of the parent can still leave a temporary folder.

The fixed 1 GiB watchdog was intentionally removed. These historical
refusals are not a Futian success; staging still needs measurement without that cutoff.

Before connecting a helper, tests must cover unsaved source, path-relative
assets, explicit child settings, cancellation, stale results, malformed output,
ordinary exit and reaping, and window closure. Test the same
candidate with an actual native window when the environment permits it. The
shared path is integrated, but native-window acceptance remains open.
