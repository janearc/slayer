# Drawing a region on Metal

A region is a half-open rectangle in the pixels of a complete picture. Its
output has the rectangle's width and height, but its rays, sample locations,
pixel footprint and light are those of the complete picture. A region does
not change the camera or the dimensions used to prepare the scene.

## Coordinates and dispatch

Keep two coordinate systems distinct. Full-picture sample coordinates select
the ray, filter position, shutter time, aperture point and shared sky looks.
Compact coordinates address the region's ray tape, readback band and output
frame. Convert the pixel rectangle to sample coordinates using the samples
along a pixel's side, checking bounds and multiplication before allocation or
narrowing to the Metal index type.

Prepare the scene, camera header and pixel footprint with the full picture's
sample dimensions. Dispatch only the region's sample rectangle. Pass its
full-picture sample origin separately from the dispatch and band origins:
the shader adds that origin when finding a sample's place, while writing to a
compact band with local coordinates. The existing band interface has a
vertical band origin; it does not supply a horizontal region origin.

Extend
the interface and its ABI deliberately, and test the host and shader sides
together. Do not alter the coordinate convention of the bake kernels.

For lenses whose rays the CPU makes, produce the same full-picture rays at
the region's global coordinates and store them compactly. Index that tape
locally in the shader. The existing striped lens path resets its local ray
origin for each stripe, so it must retain a separate full-picture origin.

The finisher consumes compact bands in order and retains each pixel's existing
down-then-across accumulation order. Progress and cancellation describe the
region's rows, not an assumed complete picture.

The existing `SLAYER_ONLY` diagnostic shades a window at its full-picture
sample coordinates but still dispatches and returns a whole picture. It can
help investigate differences; it is not the region interface.

## Modes and whole-picture dependencies

Start with stills in the ordinary software and native Metal traversal paths.
Keep world-space lighting and density bakes identical to the complete picture;
measure their cost, since each independently staged region may pay it again.
The shared sky-look pattern uses the global pixel coordinate. The spread
filter, aperture and shutter use global sample coordinates. Preserve these
inputs and compare their results before admitting the corresponding modes.

ReSTIR has per-sample reservoirs, spatial neighbours and temporal history.
Refuse it by name for a region until its dependencies are reproduced or a
measured difference is an explicit contract. The gravity field path creates
its own full-frame ray seeds, and the retained-frame path owns a separate
stream and size key. Refuse these paths until region handling is implemented
and verified; do not silently switch renderer or draw a whole frame.

Linear light equality alone does not prove equality of the delivered image.
Bloom reads neighbouring pixels, and stock, grain, exposure and other
development may depend on the whole picture or its coordinates. Use the
complete picture's development context and any required halo, or refuse an
unsupported combination by name. Check the actual operation before claiming
that any particular development setting is region-safe.

## Evidence

First draw the same small scene twice on the same build and backend, retaining
both linear and developed results and reporting any differences. Then compare
a region with the corresponding crop of a complete frame from that build:
include nonzero origins, edges, a one-pixel region, multiple sample counts,
and bands crossing pixel boundaries. Exercise both software and native Metal
on a real device. A skipped device test is not GPU evidence.

Include lenses with CPU-generated rays, spread filters, shared sky looks and
any supported shutter or aperture in targeted comparisons. Check that global
coordinates and full-frame footprint survive dispatch and that a region cannot
reuse a complete picture's kept rows or another region's rows. Test invalid,
overflowing and out-of-bounds rectangles, named refusals, cancellation and
compact output dimensions.

Compare the street region against a whole frame
drawn by the same build and settings; preserve any difference and its cause
instead of declaring exactness from a passing small test.
