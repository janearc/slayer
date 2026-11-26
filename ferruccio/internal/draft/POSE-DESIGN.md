# Scene pose edits

The draft owns source text. The editor owns undo by retaining that text in
ApplyText; a pose edit introduces no second history. Objects are fresh immutable
values with source-path identity, retained through movement and YAML reordering.
Names in native things and imports remain distinct even when equal.

Native east/north/height edits are supported for explicit placements in a
schema-checked space set. This is the metre ENU contract of session/afloat.go,
not terrain height or a relative/orbital placement. Heading is clockwise degrees
from north, as set/heading.go defines; omitted space headings mean zero.

Imports use the authored ENU at vector from set/imports.go; editing supports
metre worlds only, so the draft neither invents nor duplicates unit conversion.
Imports do not open geometry merely to edit placement.

Objects refuses unsupported documents or malformed supported placements.
ChangePose accepts finite relative deltas with explicit m and deg units. It
updates only the selected placement in a fresh checked YAML tree, preserves
comments and unrelated nodes, checks the whole candidate through its schema,
and stores text only on success.

Zero deltas preserve exact bytes. Schema bounds reject movement beyond
supported positions and headings; controls must report that refusal rather
than silently clamp the gesture. Restoring prior text exactly restores pose and
comments. Unsupported placements are omitted from object controls rather than
interpreted approximately.

## Imported model size

An imported model can say a target physical `height` in metres. The checked
asset loader normalizes its native geometry to that height. A draft size edit
sets that existing schema field for one authored import, or adds it where it
was absent. It does not pretend a generic `scale` field exists for every kind.

`SetImportHeight` accepts an authored import ID and a finite positive target
height within the schema's range. It validates the entire candidate and keeps
comments and unrelated fields. A no-op preserves exact bytes; a refusal
leaves the draft untouched. Tests reopen a saved edit through Slayer's set
reader and check the imported asset's actual normalization, not just YAML.

In Play, an imported object selected in the object picker offers a Size
panel. Its height is an explicit metre value, blank when the source has no
height; submitting it makes one undoable draft edit and does not request an
offline preview. Invalid input reports the refusal and leaves the source.
The panel is limited to imported models; it does not offer an invented size
control for other kinds.

Recipe-backed things in a space set use the same explicit metre placement as
kind-backed things. They are included in the object list when their placement
is directly authored. The object's recipe name remains separate from its kind;
relative, orbital and unsupported placements stay excluded. Pose edits change
the placement, not the recipe, and keep the same identity and undo history.

For a ground placement whose heading is generated, the editor can explicitly
replace that generated heading with a chosen angle, including zero. It labels
this as replacing the generated value and never does so merely because
coordinates were opened or moved. The transaction validates the whole set,
preserves unrelated source and can be undone back to the omitted heading.
