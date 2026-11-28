# Camera navigation

The preview displays the last completed render. A supported camera gesture
changes the editable plan. Play updates the working view, while other
compositions can request a preview after input settles.
During a drag the view shows the pending angle; it does not rotate the bitmap.
The camera edit participates in the editor's undo history.

The workspace retains at most one pending navigation request. It captures the
document object, source path, text and specimen. Each gesture replaces that
request and moves its deadline forward by 300 milliseconds. A changed document,
manual edit, specimen selection, reference image or Stop discards it. The delay
is an authored interaction choice to group a continuous trackpad gesture.

The workspace cancels its stale request, waits for that client to finish and
processes only the latest settled navigation edit.
A redraw timer wakes the demand-driven window at the deadline. Results still
need matching path, text and specimen; a generation counter also rejects a
completed result after Stop or a view switch, even when the text is unchanged.

Navigation is enabled for accepted catalogue previews and fixed set cameras
with explicit `location.at` and `orientation.look_at` coordinates. The first
authored camera is controlled.

Zoom needs an explicit film `wide` angle;
moving cameras, relative rigs and lens rigs remain outside these controls.
Mission images, opened image files and indirect jobs are ordinary image views.

A small
click does not change the camera. Pointer capture permits releasing a drag
outside the image. Keyboard controls provide the same camera operations.
