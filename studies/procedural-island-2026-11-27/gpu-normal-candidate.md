# Isolated GPU mesh-normal candidate

This records diagnostic commit `6528b5ba3` from a separate GPU debug
checkout. It has not
been integrated into the shared renderer. The candidate copies the existing
CPU `trace/surface.go` `meshNormal` front-facing correction into GPU
`shading_normal`, immediately after `float3 normal = facing(hit, way);`:

```metal
if (any(hit.flat != 0) && dot(hit.flat, way) < 0 &&
    dot(normal, hit.flat) < 0) {
    float3 side_normal = -normal;
    float3 view_way = normalize(way);
    float seen = -dot(hit.flat, view_way);
    if (seen > 0) {
        float share = (0.01f + dot(side_normal, view_way)) / seen;
        normal = normalize(side_normal + hit.flat * share);
    } else {
        normal = hit.flat;
    }
}
```

The 0.01 value follows the CPU's picked `leastFacing`, not a new physical
constant. This candidate does not implement CPU `raised`'s separate
tangent-lift suppression for strongly opposed normals. It has not been
reviewed or tested; the image below is a bounded visual observation.

Native job 429 rendered the exact `island-dense-lattice.yaml` scene
with this candidate, retaining raw and receipt in the November 27 archive
under the normal-parity view. Native control 406 is the dense-lattice view. The researcher saw most thin
black scratches across horizon and midground disappear without a major
loss of terrain form, though broad angular blue-grey shadows and some
facets remain. Both receipts say cell-target-limited with 259,697 vertices.

At (391,90), the candidate's native raw RGB is (0.16446051, 0.141668096,
0.136346623), close to the CPU replay (0.164460063, 0.141667783,
0.1363464); the prior GPU raw was zero. Other raw pixels (392,140) and
(531,210) also became nonzero. These are three observations in one view,
not evidence of general CPU/GPU parity or a complete terrain fix. The
earlier normal, occlusion and renderer controls are in the adjacent README.

Job 432 applies the same isolated candidate to the previously retained
8 km camera scene, with no other scene changes. The archive contains the horizon-parity and original horizon views. Independent inspection again found that most thin black
scratches across the distant hills and midground disappeared while the
skyline and rolling forms remained. Angular blue-grey patches and
soft/faceted terrain remained. This adds a second view of the same regional
seed; it does not establish general parity, near-ground quality, temporal
stability, or the separate CPU `raised` behavior.
