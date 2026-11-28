# Pterosaur membrane paint on the GPU

The detailed pterosaur's membrane has rest points: the CPU interpolates the
undeformed point with the same face and barycentric weights as the ray hit
(`mesh/meet.go`, `trace/surface.go:paintPoint`), then samples the authored
membrane texture there. This keeps its markings tied to the membrane as it
moves. `gpu/rest_paint.go` currently refuses painted posed meshes because it
cannot supply that point. The refusal must remain until the GPU can do so.

GPU surface textures are evaluated by the CPU finishing stage in
`gpu/sampled.go:painted`; the hide and membrane's custom paint functions do
not inherently require reimplementation in Metal. The missing part is hit
position: `gpu/trace.metal` currently returns the posed local point. A proper
implementation would expose each mesh face's rest corners, pack and key them
with the corresponding posed triangles in the GPU tree and cache, and
interpolate using the actual hit's barycentric weights. Both software tree
traversal and native Metal triangle hits, including instanced transforms, must
carry the rest point into the paint record. Existing unposed paints and
unpainted meshes need to retain their semantics. This is a cross-cutting
renderer and mesh change, not a pterosaur-only material switch. Cache format,
geometry identity and both traversal paths would need verification by the
renderer owner. No implementation or parity claim is made here.

The optional `detailed-plain` study keeps the geometric animal and its
mottled hide while deliberately omitting the membrane texture. It uses the
existing allowed unpainted posed-mesh path. The lost gradient and streaks are
an explicit artistic compromise, and its visual and renderer results remain
to be established. The `detailed` study retains the original textured baseline.
