# Camera detail trial

This is an opt-in experiment. It selects a representation per planned
city building before batching geometry. With
`ClusteredCitySpec.DetailView` unset, the authored detail is unchanged.
The caller supplies a camera, film width, horizontal field of view and a
pixel threshold; the trial has no default threshold.

For a quick, night building below that threshold, the trial retains its
hull and the same seeded warm and cool window panes, and omits unlit
panes. This is an approximation: the omitted glass can change
reflections and shadows. The trial does not cull off-screen buildings.

The projection assumes a fixed square-pixel pinhole camera and building
bounds in the same flat city space. It is not connected to an orbital,
a moving camera, or a shot renderer; those require the corresponding
coordinate transforms and a selection before geometry is constructed.
Infill, trees and infrastructure are not simplified. No whole-process
memory budget, image parity or production-ready detail policy is proven.

A no-draw experiment used a flat 10,000-building night city with no
trees and an artificially distant camera selecting every building.
Across one run per version, faces fell from 2,009,772 to 1,106,816,
while the warm and cool lit-pane face counts remained 634,208 and
158,508. Peak process footprints were approximately 585 and 327 MiB,
respectively.

These are construction measurements, not full Dark City or
GPU-render results. The measured binary predates a final conservative
camera-frame validation check; the experiment used an identity frame.

Focused tests cover projection, uncertainty fallback, seeded lights and
default behavior. Before use on a shot, establish image and memory
comparisons with the actual camera, resolution, lights and secondary
rays; in particular, verify that distant lights, reflections and shadows
remain acceptable as the camera moves.
