# GPU open-water marching primitive

`March` accepts three host-provided wave fields and a bounded batch of rays. Its Metal kernel samples periodic height and horizontal displacement fields, applies one choppy fixed-point iteration and footprint filtering, and returns a distance plus sky or flat-water status. It follows the open-water march structure in `oceantex`; it does not shade the hit, handle shores, or generate the wave fields.

The caller must supply fields ordered from largest to smallest texel, correct per-layer crest bounds and fields from the same sea and time. Calculations are float32; rays that leave the supported periodic-coordinate range are refused. The resulting intersections are not yet compared against the CPU ocean. The primitive allocates a session per call and is not connected to the compositor. The tests pass in the draw tier as of 2026-11-14.
