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

## Saving a copy in another directory

Save As keeps a job's scene inputs resolved to the same files or folders.
Only the job's `scene.set`, `scene.play` and `scene.frames` path values
are rebased. A relative output path still follows the saved job. The
original files and same-directory saves keep their authored paths.

Apply the same relocation to the editor's undo and redo snapshots when
the copy succeeds, so history changes remain meaningful in the new
directory. An unreadable draft can still be saved verbatim. A failed
write leaves the current source, destination and history as they were.

A set's imported geometry must stay inside its scene folder. When saving
a set to a new folder, copy the producer-verified entry files and buffers
at their existing relative paths. Keep the import declarations as written.
Accept existing identical resources and refuse conflicts before writing
the YAML. Remove new resources if the save fails. Recheck the destination
with Slayer's set reader before saving. A source link that cannot resolve
inside the new folder causes the copy to fail.

An unfinished set remains recoverable when the YAML or the source reader
cannot check its imports. Save the draft as written and report that its
dependencies could not be checked or copied. Destination conflicts still
refuse the copy before writing YAML.
