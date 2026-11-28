# Edit a scene in Play

Open a Slayer set with **Open…** or Command-O. You can also choose an example
from the scene menu and edit its copy. If the current document has changes,
Ferruccio asks whether to save, discard or keep editing before switching.
Press Command-S or choose **Save** to save. The first save of an example
asks where to put the copy. Saving a job to another folder keeps its scene
references pointed at the original inputs. Keep a set's external dependencies
available when moving it.

## Place an object

Click an object in the working view. Use **Move** or M and drag on the
horizontal plane through the object. Use **Turn** or T and drag horizontally
to change its heading. **Coordinates…** lets you enter supported positions
or headings directly. For terrain, height is relative to the ground at
the placement. Editing a generated terrain heading replaces it with the
angle you enter, including zero. Undo can restore the generated heading.

**Size…** sets the target height of an imported mesh in metres or the uniform
scale of an eligible native object. Generated stand members select their
authored group. **Region…** edits the explicit region of an eligible stand.
The available controls depend on the placement and its constructor.

Use **Add…** to search the current set's recipes, Slayer constructors and
animal or plant profiles. Choose an entry, supply a unique name, seed and
position, then choose **Add object**. A profile uses its built-in book unless
you choose a custom one. The draft keeps that dependency for subsequent
staging and rendering. **Source** opens the YAML. Command-Z undoes an edit
and Command-Shift-Z redoes it.

## Inspect and render

Choose **Look** or L for camera inspection. Use the wheel to approach and
the [Orbit and Travel balls](TRACKBALL.md) to change the camera. A committed gesture joins
the same undo history as a source edit. Edits in Play update the working
view. Choose **Preview** or **Render fully** when you want a Slayer image.

During staging, the current draft can remain editable while the helper builds
geometry. **View details…** shows staging errors and describes proxies or
omissions. **Render fully** offers output size, camera, renderer and sampling.
Use the authored ray grid for a physical rig. The previous accepted image
stays visible until the
new image is ready. **Stop** requests cancellation of the current job. See
[queue setup](howto-run-dis.md#render-your-scene) and
[full-render settings](../../../ferruccio/internal/preview/FULL-RENDER.md).
