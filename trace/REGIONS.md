# Drawing part of a frame

`SampledRegionKept` draws one nonempty rectangle in the full frame's
coordinates. It uses the full frame for the camera and pixel footprint,
and allocates only the requested picture and one row of its fine samples
per worker. Each pixel's samples are averaged in the same order as a
whole draw.

The draw stops at the next sample after its context is done and keeps
only completed rows. Row numbers start at zero in the region. A caller
supplying retained rows must key them by the complete frame inputs and
the rectangle, including its origin; equal widths do not make rows from
different rectangles interchangeable.

The work limit counts samples in the requested region. The full frame's
dimensions are checked separately for safe fine-coordinate arithmetic.
Initial clear media are checked as for a whole sampled draw. Output is
linear light, coverage and depth; development and its neighbouring-pixel
effects belong to the caller.
