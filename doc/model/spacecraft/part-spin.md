# Native material-group rotation

A manufactured thing in setting: space may opt into a part-spin block:

    part-spin:
      material: propeller blade
      pivot-metres: [0, 4.75, 38.9]
      axis: [0, 0, 1]
      rpm: 300
      phase-radians: 1.5707963267948966

The exact material name selects only parts appended for this placed object.
No match is an error. Selection and every wrapper are validated before any
motion changes. Other objects, lights and the retained manufacturing prototype
are untouched. Shapes, mesh groups and material pointers remain shared.

Pivot and axis use native factory-local x/y/z, with y up; they are not map
east/north/up coordinates. Pivot-metres does not undergo world.Unit conversion.
The parent's placement, heading, scale and clock carry the local rotation.
RPM is signed and bounded to +/-1000000; the axis must be finite and nonzero.
Phase is radians and defaults to zero. Exact zero angular offset returns
the parent transform unchanged. Omitting the block retains legacy behavior.

A common numeric YAML anchor can explicitly share RPM with a cavity field:
define the anchor on its earlier rpm value and reference it on the later one.
Cavity presence never enables rotation implicitly.

This adds no mesh generation or remeshing. It rotates the selected native
material group's geometry through Motion.At(time), on CPU and GPU. Individual
blade selection, arbitrary component graphs and moving embedded lamps are
outside this first contract. Ground settings refuse the block explicitly.
