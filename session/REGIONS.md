# Region light

A CPU still may draw a rectangle of its full frame as linear light. The
session preserves the full camera dimensions and meters the full view
before applying the rectangle. Each row uses the frame's cancellation and
retention contract. The caller must key kept rows by rectangle.

The session refuses region output on paths that do not yet preserve it.
Developed pictures and overlays may need full-frame coordinates and
neighbouring light; callers must check those effects before publishing a
picture. The rectangle is checked against the resolved size before drawing.
