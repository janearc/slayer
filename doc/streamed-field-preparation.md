# Prepare fields without keeping every intermediate

The renderer already reserves packed density and emission ranges for flames,
density for wispy puffs and optional source-light grids for smoke. The shaders
fill those fields before tracing begins.

Previously their full-grid raw results were copied into additional Go arrays,
then packed into the final scene buffers.

Production preparation now uses the same streamed readback as the tracer:

- Flame bands deinterleave density and RGB emission directly into their
  reserved, separate packed ranges. There are no split full-grid density
  and emission arrays in this path.
- Wispy density bands write directly into the final packed bake payload.
  The complete job count and offsets are checked before its allocation.
- Optional smoke-light bands write into their existing reserved RGB grids.
  Lighting is still recomputed for each frame, including moving sources.

Each consumer checks its reservation, global plane order and band length. The
scene allocator owns disjoint reservations: it advances the packed offset
after each field. Writers check their own range and do not compare it with
every neighbour.

Passing overlapping reservations directly to a writer violates that caller
contract; bounds checks alone cannot detect it. Scalar bits are packed with
the same float32 representation as before. An incomplete bake returns an
error; private, partly filled bytes do not become a completed render.

## What stays expensive

The final encoded fields still occupy their full grids. A band has both
Go and Metal output storage; a short grid can fit in one band. Scene
geometry, Metal inputs, acceleration structures and image development
remain separate costs. This does not establish an aggregate memory cap.

The density, flame and lighting shaders, grid dimensions, noise, source
tables, interpolation and integration rules are unchanged by this host slice.
It adds no temporal cache, weather lag, empty-space skipping or coarser
approximation.

The direct-copy input bridge is described in [streamed
readback](streamed-readback.md).
