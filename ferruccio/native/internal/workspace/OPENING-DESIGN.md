# Opening a document

Opening a document, choosing an example and starting the workspace do not
start an offline render. The working view can still construct geometry.
Preview and full render are available as explicit actions; camera navigation
can also request a preview. The default transport sends requests to Azazel.
Opening retains the existing save, discard and cancel protection for unsaved
work.

The window root handles command and O before dispatching to its children.
It asks the next UI tick to invoke the same protected open action as the
button. A dialog in the overlay stack keeps priority over the shortcut. The
application menu also offers Open. Ordinary O goes to the focused control.

The local bundle can capture an operator's Azazel executable and queue at
build time, with launch-time environment overrides. An unconfigured queue
leaves rendering unavailable and reports the missing configuration; it never
falls back to direct Slayer. This bundle still depends on external software
and scene data and is not the portable distribution.

A working-view refusal reports the refusal without asserting that the scene
will render. The blank image is a placeholder and is never evidence of a
successful producer result.

A job document remains editable; until the working view can stage jobs, it
states that limitation instead of trying to validate the job as a set. An
invalid job reports its validation error instead.
