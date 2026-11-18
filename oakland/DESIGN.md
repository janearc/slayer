# Oakland static glTF adapter

Oakland is a narrow import boundary from a static glTF 2.0 scene into
Slayer's immutable triangle meshes and native `scene/thing` values. It does
not own world placement, material policy, render jobs, cache insertion, or
intersection code. `mesh`, `maths`, and `scene/thing` do not import Oakland.

## API

`Read(path string, policy Policy) (*Asset, error)` reads a path relative to the
allowed root, the selected static scene and its complete external-resource
closure. `Policy` names the resource root, an optional scene index, optional
target height and unit intent, and resource limits.

Zero limits select bounded defaults: 32 MiB source closure, 256 MiB buffers,
1,000,000 vertices, 3,000,000 indices, 100,000 nodes, 500,000 instances, and
256 hierarchy levels. Otherwise each limit must be positive and no greater
than its default. A scene index is required when the document contains more
than one scene, even when glTF names a default scene.

Target height uses the caller's declared unit (`m`, `ft`, or `in`). Oakland
records that intent in the key and does not claim a physical measurement. All
decoder options are part of the asset options key.

The vertex and index limits also bound aggregate unique baked geometry. Height
normalization bounds aggregate vertex visits across unique transformed
geometry keys before measuring scene bounds. Repeated placements that share a
canonical transformed key count once for those operations.

`Asset` is immutable after `Read`. It exposes copies through
`SourceClosure() []Source`, `OptionsKey() [32]byte`, `MaterialSlots() []string`,
and `Instances() []Instance`. `HeightNormalization()` reports original bounds,
target height and unit, scale factor, and vertical offset when requested.

Each `Source` records canonical relative path, byte length, and SHA-256 of the
exact byte slice decoded. Closure order is canonical path order. The asset key
combines this closure and canonical decoder options; it is not a complete
render key when a caller supplies materials.

`Instantiate(map[string]*thing.Material) ([]thing.Thing, error)` requires an
explicit native material for every glTF material slot. The caller owns those
material pointers and must not mutate them during a render.

Job owners must add canonical definitions of bound materials to their render
key; material slot identifiers alone are insufficient. Meshes remain shared
for repeated instances with the same canonical geometry and placement.
Transformed variants are shared by identical canonical transform keys.

## Supported input

The first adapter accepts indexed and non-indexed `TRIANGLES`, positions,
optional vertex normals, node hierarchies and repeated mesh instances. It
refuses requested skins, animation, morph targets, texture, image or sampler
data, material properties beyond the slot name, node matrices, extensions,
sparse accessors, unsupported attributes, and primitive modes other than
triangles. Materials are names and slots; the caller binds Slayer materials.

glTF JSON is parsed and validated with Puzzlemouth `yamlfile.Schema.Read`, then
decoded from the checked node. Unknown members are refused. GLB framing uses
checked binary reads; its JSON chunk follows the same Puzzlemouth path.

Buffers and accessor ranges use overflow-safe bounds checks. File references
must remain under the policy root after symlink resolution. Every source is
read once; hashing and decoding use those same bytes.

Node transforms are accumulated at the format boundary. The pinned `maths`
package provides vectors, rotations and uniform subspaces, but no affine
matrix or inverse-transpose normal API.

Oakland uses private bounded affine operations to normalize glTF transforms
into canonical local Slayer meshes. It transforms normals by inverse transpose
and reverses winding for negative determinants. Singular transforms and
non-finite results are refused. It adds no general math or intersection engine.

Tests pin coordinates, normals and winding for nested rotated non-uniform
scales, and check forward/inverse properties.

## Claims and limits

The self-authored `testdata/two-metre-human.gltf` and matching `.bin`, plus a
small `testdata/embedded-triangle.glb`, are licensed CC0-1.0. They are decoder
fixtures, not physical measurements.

Normalization scales the selected geometry and grounds its lowest point at
y=0. Oakland reports the source bounds, scale factor and vertical offset; it
does not claim the source anatomy, dimensions, material or lighting are
physically accurate. The source closure and policy key make that limitation
auditable.
