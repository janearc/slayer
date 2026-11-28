# Physical lens editing

The Physical film panel reads the focal length and f-stop of a set with
a complete physical rig. Set lens changes both values in one undoable
source edit. The focus target, frame gauge, shutter and comments stay in
the authored set. Saving the set makes the same values available to the
working view and to a subsequent full render.

Focal length is in millimetres. Both values must be finite and within
the producer's lens ranges. Invalid input leaves the whole draft intact.
The working view uses the new focal length in its pinhole approximation.
The f-stop affects the offline exposure and depth of field.

The Physical film action sits beside Add on the lower control row.
Selection controls and the film action fit at the largest type size
in the smallest window.
