# Render optimization controls and preflight

Every optimization must have its own independent control, including changes
that preserve the exact picture. A preflight is advisory: it never enables a
shortcut or changes a job's authored settings.

Record the effective settings with the job and include them in
workload-profile identities. A preset may fill in choices, but each choice
must remain separately inspectable and changeable.

## Before starting a job

`game run workload-preflight -- report` compares an exact-key recorded process
lifetime peak with explicit host capacity, current use, process count and
headroom. It gives evidence-based advice, not an upper bound or admission
promise. It does not derive a complete identity or host readings from a job
and will report unknown if evidence is missing.

See [workload profiles](../render/workload/README.md) for the command, input
contract and excluded memory. `scene-complexity` measures constructed geometry,
but constructing a scene can itself be expensive and is not a safe substitute
for an upfront peak estimate.

## Separate controls

| Choice | Current state |
| --- | --- |
| City batch placement | `CityGrid.BatchPlacement` and a named set recipe’s `batch-placement` select fused or nested transforms independently; both preserve exact geometry. The authored option supports individual city things; other contexts are explicitly refused. |
| City vacant-lot index | Green and night clustered-city recipes accept `vacant-lot-index: on|off`; on is the existing default, off rescans with equivalent landscape geometry. Other city contexts explicitly refuse the setting. |
| Mesh prototype sharing | Authored Heard individual/multitude stages accept `share-anatomy-prototypes`, default on; off retains equivalent geometry, materials and motion in separate meshes. |
| Retained GPU geometry | Frame sessions retain tree and packed geometry; disk tree caching has a separate Go API. |
| Projected-size geometry | Some terrain and forest adapters select view-dependent representations; the general Heard film path does not. |
| Texture filtering | Selected textures can use pixel footprints when sampling. Filtering alone does not reduce stored geometry. |
| Volume/scattering shortcuts | Lighting shortcuts exist under job `shortcuts`. |
| Output delivery | A job can choose raster, rays and frame rate. |

Six frames per second means fewer temporal samples; it does not make each
frame's world smaller. A 320-pixel frame may still construct the complete
world and its acceleration structures. Even a tiny object can cast a visible
shadow, reflect light or emit light, so projected size alone is not permission
to delete it. Approximate detail policies must state their effect and must not
claim a certified pixel-error bound without the corresponding proof.
