# A moon behind the grove

The woodland study uses existing redwoods, the Kilauea DEM, lunar geometry
and the pinned NASA lunar face. Its fire is still the surface heat/fuel
prototype. Crowns do not ignite, and terrain does not accelerate the model.

The lunar sphere revealed an unrelated scale problem: its distance enlarged
the scene-wide ray-origin offset for every leaf and patch of ground. Pure
emitters need no diffuse-light queries and must not enlarge that local scale.
A reflective, textured or glossy body still contributes its bounds.

CPU and GPU must use the same offset scale. Geometry stays present for
primary visibility and occlusion; no sphere is moved closer or made smaller.
At distant hit points, the point's own coordinate magnitude still supplies
the numerical scale. This keeps the lunar model and authored sky intact.

Check unchanged local scale when adding a distant pure emitter, changed
scale when that body reflects or has a reflectance texture, and a nonzero
far-point offset even when every surface emits without reflecting.
Check CPU/GPU image agreement and the actual wooded lunar composition.

The disc is a visual counterpart of a distant illumination source. It must
not block that source's own light. Marking the disc clear to shadow queries
preserves its primary visibility. An ordinary occluder still shades the
moonlight; a focused render test checks both cases with actual lighting.
