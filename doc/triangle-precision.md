# Thin supports need a stable intersection

The ELT cutaway exposed dark pixels on ordinary closed trusses: the CPU
lit the face while the GPU shadowed it. The old single-precision
Moller-Trumbore distance calculation lost accuracy on long, thin triangles.

At one recorded pixel the CPU distance was 80.1030999 m and the GPU
reported 80.103264 m. That displacement could put the next ray inside
its own surface.

The kernel now translates each triangle into the ray frame, shears along
the dominant direction and computes compensated edge determinants. This
follows the approach already used in mesh/meet.go.

The source is Sven Woop, Carsten Benthin and Ingo Wald,
[Watertight Ray/Triangle Intersection, JCGT 2(1), 2013][paper].

The paper also uses a rare double-precision fallback and conservative
hierarchy bounds. This float-only implementation does not establish that
full guarantee. Slayer's grazing rejection, edge slack, exclusive distance
interval and secondary-ray offsets remain unchanged.

No larger bias, framebuffer tolerance or instrument geometry hides the
original discrepancy.

The existing 49 by 49 ELT cutaway is the regression witness. Its old
relative RGB error was 1.66%, above the unchanged 0.5% limit. After the
change both software and native Metal paths reported about 0.00016%.

These are results for that test scene on the M3 Max, not a bound on all
triangles or a performance benchmark.

Transformed and scaled meshes, a grazing large floor, smooth shading and
native planar face ranges provide adjacent checks. Personal telescope
trusses are additional witnesses in the observatory-sites branch.

[paper]: https://jcgt.org/published/0002/01/05/

## Native shadow candidates

A later scaffold study isolated a second issue: native shadow queries
blocked two lit support pixels even with software primary intersections.
Hardware any-hit results now name a candidate standing to check with the
same ordinary-mesh predicate.

A rejected candidate triggers software traversal of the full shadow ray;
it must not imply the ray is clear. This preserves masks, finite lamp
segments and shared groups without a larger origin offset.

The unchanged scaffold comparison fell from about 0.59% relative RGB error
to about 0.00028% on the M3 Max. The standing check adds traversal work;
its cost in a dense misty forest still needs a matched measurement. Older
native-shadow timings predate this change.
