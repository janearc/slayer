# Camera controls

The **Orbit** ball rotates the camera around its look target while keeping its
distance and the upright camera convention. The **Travel** ball moves the
camera and target together. Drag horizontally to move right or left, and
vertically to move forward or back. Hold Shift to move up or down instead.
On Orbit, Shift constrains the drag to the dominant axis.

Release commits one undoable edit for the set's editable camera. For a job
or a borrowed camera, the balls move only the working view. Use **Reset view**
to restore the staged pose. Renders continue to use the authored camera.
Escape or loss of pointer capture cancels the gesture. The balls preserve
the selected object and its mode.
See [the working view](WORKING-VIEW.md) for the difference between interactive
geometry and accepted Slayer images.
