# Rendering faster

The current [optimization catalogue](optimizations.md) explains every layer:
shared geometry, packing, retained films, native mesh traversal, field bakes,
lighting grids, streamed readback, compositing, gravity and completed-work
caches. It distinguishes reuse from numerical changes and approximations.

Use [modes](modes.md) to choose a practical science or fast-film workflow,
and [fast light](fast-light.md) for the lamp-store and shared-sky study.
The [renderer guide](renderer.md) gives the API and lifetime map.

## Status of the original roadmap

| Stage | Current state |
|---|---|
| bake expensive density fields | implemented for supported boxed, flame and wispy fields |
| retain resources between frames | implemented by `gpu.Frames`; time-changing host encoding can remain |
| hardware ray traversal | explicit native mesh shadow and closest-hit modes |
| wavefront queues | not implemented as the production tracer |

The [animation performance record](animation-performance.md) and
[native traversal studies](native-primary-traversal.md) retain dated
measurements and rejected experiments. They are historical evidence,
not the current feature checklist. Native acceleration handles meshes;
analytic shapes retain their own intersection paths.
