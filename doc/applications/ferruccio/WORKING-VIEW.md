# Working viewport

The working view shows the geometry Slayer staged for the current draft. Use
**Solid**, **Edges** or **Wireframe** to inspect shapes and placements.
**Rendered** shows accepted Slayer pixels. In Play, manipulating a camera
or object can display working geometry while you drag. Release commits one
undoable source edit. Use Preview or Render fully to make a new photograph.

The staging helper accepts sets, legacy sets and still jobs. Coverage
depends on the source. The viewport approximates water with a finite
mean-level surface, draws glass as opaque and uses a pinhole approximation
for physical lenses. It can omit unsupported lights, volumes and other
features. Open **View details…** for the current scene's notices or staging error.
The offline image uses Slayer's full render path.

## Implementation

Staging preserves authored IDs and shared object-local meshes. Rendering
and picking use those meshes and their staged transforms. During a gesture,
only camera or placement data changes. Cached geometry stays immutable.

The lightweight cache can anchor a compact world mesh using its first
visible producer placement or the midpoint of that surface’s native bounds.
It does this when the anchor is more than 100,000 scene units from world
zero along any axis and every component fits within 100,000 units of that
anchor along each axis. Camera motion leaves the mesh anchored. Dispersed
worlds can still lose precision during working GPU upload, and anchoring
cannot recover precision already lost in the source vertices.

A background worker reads the current source snapshot. It can publish
the parsed set for authoring before the helper finishes assembling geometry.
The app accepts results only for the current document, path and buffer. Undo
can cancel a pending stage and restore the previous source and its view.
Closing joins helper work before deleting its cache.
