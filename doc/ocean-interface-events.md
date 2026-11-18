# Reflections inside a wavy water interface

`world.ocean.interface-events` opts into bounded CPU optical transport at the
tagged ocean surface. Zero or omission preserves the legacy one-mirror path.
The authored positive value, at most 16, bounds the number of ocean-interface
events followed on each ray branch. A reflection, including total internal
reflection, and a transmission each consume one event on their own branch.

This permits a ray reflected by one wave face to refract out through another.
The legacy renderer does not refract after a mirror bounce. That remains its
default behavior for every surface and for oceans without this opt-in.

Fresnel reflection and transmission retain their respective energy shares.
When the event budget is exhausted, further reflected and transmitted branch
radiance is zero; it is not converted to opaque colored water or extra fill.
The budget is a finite-depth transport approximation, not a fluid model or an
anti-aliasing filter. It does not alter wave geometry or surface normals.

Only the exact tagged infinite-mean continuation boundary uses this budget.
Finite height fields using the legacy marcher are refused: its near-surface
tolerance may immediately re-hit the departing ray. This option does not
change that marcher or silently opt every height field into a new solver.

Other mirrors and dielectrics retain their existing traversal rules. The CPU
admission requires a staged clear ocean boundary and an integer budget 0–16.
GPU rendering refuses
the new positive budget until the same traversal is implemented there.

The infinite continuation also checks departure-origin offsets against its raw
height. At a steep bilinear crease, the ordinary normal lift can cross into an
adjacent cell and emerge on the wrong side. Only such an offset is halved until
it remains on the requested side.

After 32 reductions, a coordinate-precision fallback takes one representable
vertical step from the local height. Safe offsets and all other shapes retain
their existing origins. No ray direction, wave sample, normal, extinction or
radiance is changed by this numerical guard.
