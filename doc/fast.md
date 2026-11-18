# Rendering faster

The current [optimization catalogue](optimizations.md) explains every layer:
shared geometry, packing, retained films, native mesh traversal, field bakes,
lighting grids, streamed readback, compositing, gravity and completed-work
caches. It distinguishes reuse from numerical changes and approximations.

Use [modes](modes.md) to choose a practical science or fast-film workflow,
and [fast light](fast-light.md) for the lamp-store and shared-sky study.
The [renderer guide](renderer.md) gives the API and lifetime map.

Native acceleration handles meshes;
analytic shapes retain their own intersection paths.
