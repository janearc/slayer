# Thin supports need a stable intersection

The old single-precision Moller-Trumbore distance calculation lost accuracy
on long, thin triangles, so the CPU could light a face the GPU shadowed.
That displacement could put the next ray inside its own surface.

the kernel translates each triangle into the ray frame, shears along
the dominant direction and computes compensated edge determinants.
`mesh/meet.go` uses the same approach.

The source is Sven Woop, Carsten Benthin and Ingo Wald,
[Watertight Ray/Triangle Intersection, JCGT 2(1), 2013][paper].

The paper also uses a rare double-precision fallback and conservative
hierarchy bounds. This float-only implementation does not establish that
full guarantee. Slayer's grazing rejection, edge slack, exclusive distance
interval and secondary-ray offsets remain unchanged.

[paper]: https://jcgt.org/published/0002/01/05/

## Native shadow candidates

A later scaffold study isolated a second issue: native shadow queries
blocked two lit support pixels even with software primary intersections.
Hardware any-hit results now name a candidate standing to check with the
same ordinary-mesh predicate.

A rejected candidate triggers software traversal of the full shadow ray;
it must not imply the ray is clear. This preserves masks, finite lamp
segments and shared groups without a larger origin offset.

The standing check adds traversal work.
