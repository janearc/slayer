# GPU sky glow

`SkyGlow` applies the CPU lighting renderer's sky band and moon halo to top-down RGB pixels on Metal. It takes the same camera, sky mask and rim description as the CPU helper, returns a new image, and does not modify the input. Its calculations use float32, so historical-frame agreement remains to be measured.

The experimental hybrid lighting renderer calls this helper after fire and before strobes, matching the CPU effect order. The tests in this package pass in the draw tier as of 2026-11-14.
