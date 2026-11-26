# Staged working view

Ferruccio needs the scene Slayer actually stages, including its terrain frame
and generated objects. A session stages a job without drawing and already
inspects a frame's world and camera. The bridge will consume that inspected
world instead of rebuilding a parallel scene from the set document.

The first slice converts staged opaque surfaces at the requested scene time
using the existing finite-mesh builder. It returns the surface mesh and counts
of volumes, fields and lamps which have no guides yet. Unsupported surfaces
and empty geometry are refused, not silently omitted. Hidden surfaces remain
hidden as they are to the renderer. The result is an aggregate: it has no
writable authored identity and must not be presented as individually editable.

The session-facing snapshot takes its world and camera from the same frame
inspection and preserves those explicit omission counts. It rejects physical
lenses and rigs until their representation is designed. The owning session
must remain open while the mesh's hit targets refer to its staged shapes.

The UI must build it off its thread, identify stale work, and keep the present
view usable. Session opening currently sets process-wide stamp variables;
its concurrency and cancellation need attention before using it in the UI.

Authored paths, shared instances, terrain, infinite surfaces and lights need
deliberate representations before claiming Futian or direct authoring works.
Staging and viewing must not start a render. Tests compare the bridge with
actual staged geometry and camera, and include missing and timed cases.

A finite native height field can be shown as two triangles per source grid
cell, at the original sampled heights. A cell's true surface is bilinear,
so triangles between samples are an explicitly labeled approximation. Exact
native hit testing remains separate. To bound a single field's mesh work,
fields larger than 262,144 cells are refused pending a level-of-detail design.
No unreported downsampling is done.

Tests check sample locations, the proxy
label, a translated field and the size limit. The staged mesh carries the
notice; a UI must display it before using this path. The older working-view
path continues to refuse height fields until it can surface that notice.

For staged surfaces with retained authored references, the working mesh can
report the nearest hit's structured reference. An unannotated nearer surface
blocks a reference behind it, and an invalid or missed ray returns none.
This is read-only identification: the interface must still verify that the
source representation supports a particular edit before enabling controls.
