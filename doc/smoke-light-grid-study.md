# Smoke-light grid

The lighting grid's 64-cell longest side is an authored engineering choice.
It is a per-frame approximation; density keeps its original resolution and
the source term is rebuilt for each frame. It does not reuse old lamp light.

For hierarchy limits, GPU instance packing counts both world placements and
shared local members before allocating their packed buffers. It uses the
same two-gigabyte input cap as `metal.Grid`; this is not a total memory
budget for the host or device.

Median splits bound each software walk's pending nodes by tree depth plus
one. World, local group and triangle walks have separate 64-entry stacks.
The signed index and packed byte limits are stricter than that depth bound.
