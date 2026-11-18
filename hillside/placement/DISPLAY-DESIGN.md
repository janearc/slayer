# Native working geometry

This adapter converts native manufacture/stage surfaces to an opaque working
display. It is not the Slayer transport renderer. The same canonical scene
seed, manufacture seed and SpaceAt placement apply. Native hull detail lowers
circular tessellation while retaining closed native surfaces; unsupported hull
detail keeps the exact source representation with an explicit display notice and diagnostic. A native hull failure at construction or display extraction gets one attempt at authored detail; both failures remain errors. No authored stand-in shapes or bounding
boxes substitute for the models.

Object-local mesh pointers are cached across pose and camera changes. Native
mesh corners/normals and MeshGroup placements follow gpu/parts.go and
gpu/instances.go. Analytic shapes are tessellated from their native parameters.
The world pad is made by stage.Add, exactly as in the rendered scene.

DisplayScene keeps native Eye/Focus/Lens and authored raster dimensions.
SetViewport adjusts native sensor pixel aspect once; GPU projection uses the
same effective sensor aspect. Pick/Drag use native film rays and cached native
mesh intersection against the current At transforms. World geometry occludes
but does not return a selectable draft identity.

Picking retains each original validated native component and its transform.
GPU buffers aggregate triangles without welding independent touching solids.
The Webb inspection exposed why a globally welded picking mesh was invalid:
separate native parts may share an edge while remaining separate solids.
No native manifold checks are relaxed.

Component identity also matters when building the selection accelerator. Webb
contains separately valid parts that touch. Combining their positions into a
single newly welded mesh introduced non-manifold edges in the display adapter,
although the native scene rendered successfully. The display now aggregates
GPU corners but retains each native component hitter and its transform.

Validation is unchanged. If the coarse native representation is unavailable,
an explicitly recorded authored-detail fallback may be used; failures of both
representations remain errors with both causes.
