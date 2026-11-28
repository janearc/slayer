# Drawing a region on Metal

A region is a half-open rectangle in the pixels of a complete picture. Its
output has the rectangle's width and height, but its rays, sample locations,
pixel footprint and light are those of the complete picture. A region does
not change the camera or the dimensions used to prepare the scene.

## Coordinates and dispatch

the host prepares the scene and camera header at full-frame sample
dimensions. it passes the rectangle’s full-picture sample origin to
Metal and dispatches compact bands. global coordinates select the ray
and filter position. local coordinates address the output storage.
finishing keeps down-then-across sample accumulation within each
pixel. the separate `SLAYER_ONLY` diagnostic still returns a complete
frame.

## Current support

the Metal path supports software traversal through a pinhole.
native traversal, ReSTIR, gravity, retained frame drawers and
CPU-generated lens rays have separate dependencies. the public
[region draw](frame-regions-plan.md#supported-draws) also checks
development and overlay limits before writing. region equality must
cover the developed image as well as raw light.

## Evidence

First draw the same small scene twice on the same build and backend, retaining
both linear and developed results and reporting any differences. Then compare
a region with the corresponding crop of a complete frame from that build:
include nonzero origins, edges, a one-pixel region, multiple sample counts,
and bands crossing pixel boundaries. Exercise both software and native Metal
on a real device. A skipped device test is not GPU evidence.

include spread filters and nonzero origins in supported cases.
a new lens or shortcut needs tests for its own global dependencies. Check that global
coordinates and full-frame footprint survive dispatch and that a region cannot
reuse a complete picture's kept rows or another region's rows. Test invalid,
overflowing and out-of-bounds rectangles, named refusals, cancellation and
compact output dimensions.

Compare the street region against a whole frame
drawn by the same build and settings; preserve any difference and its cause
instead of declaring exactness from a passing small test.
