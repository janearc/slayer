# Native scene sampling

This is the prototype's development/measurement record. Its full-suite
percentage and Carrie timings belong to that earlier checkpoint. Use
[the renderer guide](renderer.md) for current capabilities and
[gravity integration](gravity-rc-integration.md) for the merged RC checks.

cells.New accepts one optional Sampler. Omitting it preserves the CPU
tracer. A sampler receives the same world, eye, lens, animation time,
pixel size and rays per side; cells still owns timing and tone mapping.
Nil or multiple explicit samplers are refused.

On macOS, gpu.NewRenderer selects a longest-side bound of 2 to 192
cells for CPU-baked volume grids. Its Sampled method fits cells.Sampler.
The policy belongs to that renderer; it does not change global defaults,
surface resolution or another scene's volume sampling.

This bound is an approximation for small output images. Compare a fixed
pose against the CPU before selecting it for a scene. Existing
gpu.Sampled keeps its original detailed-volume policy.

The Metal tracer now supports analytic infinite planes, including local
texture coordinates, transforms, primary hits, reflections and shadows.
Planes use the analytic primitive buffer's kind tag. Unsupported shapes
and lights continue to return named errors.

Game's check-native-cells measures the complete cells package at 100%.
The complete GPU suite passes at 96.9%; its existing uncovered error paths
remain visible. New plane, grid policy and renderer methods are covered.
Go coverage does not instrument the Metal shader.

check-native-anti proves that substituting a sphere for a plane, ignoring
the selected sampler and dropping the volume bound fail named assertions.
A counted medium tests density-query counts rather than timing thresholds.

The local Carrie scene compared a 32-cell flame grid against a fixed-pose
CPU render and checked the resulting PNG. Its three-render measurement
was about 103 ms per 480 by 319 frame on an M3 Max with two Go workers.
This is a scene-producer measurement, not end-to-end GUI frame rate.
