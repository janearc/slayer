# Working viewport

Ferruccio keeps an opaque native-geometry viewport responsive during gestures.
On release, the existing Slayer preview refines the authored scene. Working
geometry is an editing representation, not a claim about completed transport,
glass, lighting or the final image.

## Geometry and identity

The pure Go hillside/placement display adapter shares native manufacture seed
mixing and SpaceAt transforms with Slayer. It requests native hull detail,
which lowers tessellation without discarding arbitrary faces; kinds without
that representation retain their source detail. Mesh corners, smooth normals
and MeshGroup placements follow the existing GPU geometry encoders.

Analytic
spheres and tubes are tessellated from native parameters. The pad is made by
stage.Add, matching the rendered world's geometry.

DisplayCache retains immutable object-local meshes. Native source identities
match draft object IDs; the world is an unselectable occluder. During movement
the UI copies DisplayScene and its Objects slice, then changes only placements
and camera values. A pose or camera change does not upload replacement meshes.

## Drawing and gestures

The native workingview adapter retains GPU vertex/index buffers, depth and
offscreen colour targets. Solid, edge and occlusion-correct wire modes share
the same geometry. Camera and object uniforms change per draw. Current
integration reads the completed GPU target back for the existing image widget.

Shot.Focus supplies the authored camera anchor. SetViewport adjusts the native
sensor pixel aspect; GPU projection uses authored width times Lens.Pixel over
authored height. Pick uses native film rays and cached mesh intersections at
current placements. Drag uses the selected object's horizontal plane and
accepts finite captured coordinates outside the image.

draft.PreviewCamera exposes the exact existing Navigate orbit for transient
east/north/up camera poses. It does not edit YAML or add undo entries. The UI
commits the gesture to the source and its existing undo history on release.

## Current limits

The adapter supports fixed pinhole space scenes with native manufactured
objects and finite supported world surfaces. Recipes, burning objects, lens
rigs and unsupported set features are refused. Glass is opaque, materials use
native display colours and live lighting is simplified. Volumetric transport,
reflections and refined surface detail remain Slayer's work. Selection during
a gesture follows the displayed coarse geometry.

Mesh pointers must remain immutable. Rebuild through the cache after structural
source changes; use placement/camera updates for pointer motion. Draw-resource
statistics report completed work, not inferred frame-rate or render progress.

## Native use and packaging

The Live app opens in Wireframe. The display selector also offers Solid,
Edges and Rendered; Rendered temporarily uses Edges during manipulation,
then returns to accepted Slayer pixels. Chroma supplies the background and
edge colours. The existing opening app remains separately installed.

The filled fragment writes depth offset by one pixel of its measured depth slope before the line overlay; line depth tests remain enabled. This avoids the pinned GoGPU Metal binding's integer argument path for setDepthBias:slopeScale:clamp:. That binding issue is an upstream follow-up, not a claimed library repair. Single-sample line rasterization still has ordinary pixel aliasing.
