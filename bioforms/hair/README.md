# Surface-grown hair

Build one groom from the rest skin, using an outward-wound mesh in metres.
Give scalp, body hair, or pelage their own layers. A layer describes a
material, a candidate density, and a profile of the rest skin. The profile
can select named face groups or use rest-space landmarks to feather a
hairline and keep eyes, palms, or other bare regions clear.

```go
groom, err := hair.New(hair.Surface{
    Points: restPoints,
    Faces: skinFaces,
}, []hair.Layer{scalp, bodyHair})
if err != nil {
    return err
}
parts, err := groom.At(posedPoints)
```

For a still, pass the rest points to `At`. For a bound skin, pass the
posed points in the same order. Apply the same world transform to those
parts and the skin. Draw the returned parts with their normal surface
materials.

`Layer.Units` can place two to four candidate follicles near each
sampled centre. The centre count is divided by the group size, so
`Density` continues to measure individual candidate shafts per square
metre. `Spread` bounds their rest-space offsets. Offsets shorten at
triangle boundaries. The profile keeps or rejects each candidate
separately, and each member retains individual barycentric weights.
Use the zero value for independent roots.

A scalp layer can vary coverage, shaft length, and flow at the front,
temples, nape, and part. Choose a flow field that points away from the
part and adjust with the crown and curl. For shorter body fibres, mask the
selected regions and use shorter, finer, flatter shafts. The same controls
can follow a nonhuman animal's coat flow. Each profile describes a shaft
at a sampled rest point and can vary settings by that point and face.

The generator samples triangle area and saves each follicle as a face
index and barycentric weights. At each pose, it reads the same weights on
the posed triangle. Flow components projected into the two rest triangle
edges are read along the posed edges, then laid in the posed surface's
tangent plane. Shaft distances stay in metres under skin stretch, and
every shaft vertex retains the rest root as a texture coordinate.

The input winding determines the outside. Regenerate the groom after
changing topology. Coverage and placement are deterministic for a fixed
rest mesh, face order, seed and profile. Curves describe static authored
shapes. Strand collisions, gravity and interpolation across guide locks are
outside this geometry implementation. Long
hair can cross the skin or clothing under large changes in pose. Dense
grooms add triangles and can be costly to draw.

For visible CPU fibre shading, put a [single-fibre model](../../materials/fibre)
in the layer material’s `Fibre` field. The groom retains the shaft frame for
scattering and strand-aware shadows. Use ordinary surface materials when
drawing a groom on Metal.

`Surface.Chart` can supply canonical coordinates on any indexed skin.
`Site.Chart` gives the authored point, and `Site.ChartFlow` carries the
heading to the physical rest skin. Use `Layer.FaceFilter` to skip faces
that are entirely outside a habitat without changing their indices.
The native [human hair profile](../../doc/model/bioform/human/hair.md)
uses this path to keep regional placement across body proportions.

A strand can use a local cubic `Guide` to lift, turn, and settle its
shaft. Its control offsets are fractions of the configured length, so
the actual arc need not equal that length. A layer can use seeded
`Clumps` in its chart to cluster the free ends while the roots stay in
place. Set `Layer.SkinGap` to check the nearest oriented face at each
sampled shaft centre after posing. These are shape controls. Between
samples and at concave joins, the geometry can still intersect the
skin. Strand and garment collisions need separate treatment.

Use `Site.ChartNormal` to build a flow in the authored tangent plane
before carrying it to the measured rest surface with `ChartFlow`.
The chart uses its own face winding and interpolated vertex normals.
An absent or collapsed chart face uses the rest normal.

A `Strand.SkinFollow` share can blend the lifted guide toward the same
height above the nearest posed face. This lets a shaft stay close as
it runs along a curved surface. The minimum radius and any layer gap
still apply at ring centres. `Clumps.BlendTarget` feathers contributions
from neighbouring guide centres in the chart. Both controls affect
the free strand while leaving its barycentric root in place.

`GuideWave` bends a cubic guide laterally and out of plane along its
free length. Width, height, cycles and phase are authored per strand.
Give neighbouring shafts related phases for a coherent wave. The
root remains at its original attachment. The layer chooses enough
segments to capture its wavelength. Surface following and clearance
still operate on the sampled ring centres.

`GuideTail` adds a cubic after the first guide segment. It joins with
the same tangent and lets the free length change direction. Surface
following eases after the join and the clearance floor remains.
`Site.ChartVector` transfers a guide displacement from the authoring
chart into the measured root frame. The whole path still deforms
with the follicle frame. These controls can also shape a hanging
coat or other rooted appendage.

`ShaftTaper` controls where a shaft starts narrowing and the radius it
approaches just before the last cap. The zero value keeps the original
pointed fibre. An authored crop can retain the main shaft diameter
farther along a cut end. This changes the geometry rather than its
material or coverage.

Set `Strand.GuideEnd` to stop along a cubic guide while keeping its
follicle and initial course. The zero value uses the entire guide.
The fraction is in curve parameter space. Taper is applied over the
shaft that remains, and guide pins and clearance are evaluated along
that portion of the original curve. A layer can feather its lengths
along a coat boundary while keeping neighbouring shafts on related
paths.

`NewSurfaceGuides` seeds sparse guide anchors at vertices. It selects
nearby anchors by path length along eligible skin edges, retaining
up to four per vertex. `Weights` blends those choices at a follicle
using the triangle weights. Assign separate regions along a part or
a species-specific skin boundary to prevent guide blending through it.
The resulting weights can drive guide headings, lofts or control points.
The edge graph approximates surface distance. Rendering still uses
independently rooted strands.

`GuideWeights.Leaders` combines an anchor's contributions from all
three corners before comparing anchors. A profile can blend toward
one lock where its guide clearly leads and ease back to the shared
field as the two strongest weights approach one another. This gives
the authored lock a continuous blend at a boundary between guides.

`NewPinSurface` indexes a selected chart region. Use `Project` to get
the nearest attachment and check the returned chart point against
your habitat before assigning it to a strand's `GuidePins`. A pin
blends a cubic handle toward its own face, carried through poses by
barycentric weights and lifted in metres along the posed normal. The
correction fades before the first cubic ends, conserving the join
tangent and the free tail. A pin targets a control handle. The shaft
need not pass through that point. Projection uses local Euclidean
distance. Limit the indexed region and reject distant projections on
folded or layered chart surfaces. Clearance still applies at sampled
ring centres.

`NewTangentField` smooths directional constraints on the indexed
chart. The domain map excludes edges between domains, and the
solver transports neighbouring directions between tangent planes
before averaging. `WeakVertices` exposes near-zero directions for
inspection. Each `SurfaceCourse` follows the field with small
projected midpoint steps. The course stores a skin attachment for
each contact ring and carries its free rings in the last attached
face's frame. A weak heading, boundary, or requested contact
length releases the path into a smoothly authored fall.

Use the course's segment count for the layer. Each shaft keeps
its own root. The field is piecewise interpolated and its paths
approximate transport on the chart. Clearance checks ring centres
on the posed skin. The static controls leave collisions and image
coverage to separate evaluation.

`NewWispField` adds shape controls at sparse surface guides. It
reads nearby same-region contributors, transports their frames to
each root and attenuates headings that disagree with the root flow.
The blend supplies relative lift, shoulder and tip offsets. Each
strand still follows its own course and keeps its own root.
`SurfaceCourseSpec.Envelope` applies the free offsets in the
release face's frame, easing in with zero slope at release.

`TangentFieldSpec.Barrier` can make a finite boundary within a
domain. It applies to graph edges, interpolated directions and
projected steps. The field can continue across the chart beyond
the endpoint.
