# Camera trackball

The floating camera ball is an actual native control. Drag it to roll a
virtual sphere; its visible graticule follows the current camera orientation.
The selected object and the current object manipulation mode remain intact.
Shift constrains movement to the dominant screen axis. Escape or lost pointer
capture cancels. Camera changes create one source undo entry on release.
Look mode retains keyboard arrows and zoom for the same camera operation.

The ball projects a sphere rotation into Slayer’s existing upright camera
model. It retains the authored look target and camera distance, and uses the
same pitch limit and saved ENU values as normal navigation. Banking/free roll
is not represented by the current fixed-camera source schema and is not
silently previewed then lost at render time. The camera ball does not add
inertia or animate the camera after release.

Working meshes update on Metal while held. A settled release requests one
shared-Slayer preview. Render fully preserves authored film dimensions and
rays with exact quality; the accepted image appears in Rendered mode. Every
render keeps its source and producer receipt in the shared evidence tree.

The scene menu discovers bundled YAML scenes; the app packages the current
sets library alongside the authoring examples. Examples are copied into
private drafts, and switching protects unsaved edits.

Dataset-backed scenes
still require their source shelf. Unsupported live terrain/volume geometry
is reported explicitly; a successful Slayer still does not prove the live
adapter supports manipulating it. The regression suite keeps those outcomes
separate.

## Adjacent movement ball

The second ball translates the camera and its look target together. Horizontal drag moves right/left in the camera frame; vertical drag moves forward/back. Shift changes vertical movement to world up/down. One radius of travel is one quarter of the initial target distance. It preserves direction, lens, selected object and object mode. Both balls preview cached Metal geometry while held; release creates one undoable source edit, Escape or lost buttons restores the exact starting pose. Source, pending generations and captured input remain owned by the current document.
