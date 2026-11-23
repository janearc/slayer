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
