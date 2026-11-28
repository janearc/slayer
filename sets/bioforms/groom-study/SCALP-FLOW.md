# Scalp flow comparison

The two front and two rear frames at `5905580c` and `7fb6379f` are
matched. The earlier front view has a long exposed scalp seam. The
revised guide loft makes the silhouette fuller, shortens the seam into
a radial patch above the forehead, and fills much of the rear crown.
A point at each temple and speckle in the brighter rim persist. The
bare controls show a smooth cranial surface under the same lights.

The preceding groom blended a downward direction with a backward
direction in three-dimensional chart space. On the rising front of the
scalp, projecting those directions into the surface plane can make
their up-and-down components oppose one another. Around the boundary
between the two fields, the across-head component takes over or the
heading falls back to a triangle edge. This offers a geometric
explanation for the spokes and the gap. The render alone cannot say
how much of the bright speckle comes from unresolved shafts and how much
comes from their directional light response.

The new groom starts with a rearward course in the scalp tangent
plane. It sweeps up from the forehead, across the upper scalp, then
down the back. A turn within that plane creates the part and a limited
bend around the rear crown. The part now removes at most nine percent
of the candidate roots within a narrow strip. A broader temple
transition and a small rise in front of the ears replace the earlier
localized drop. The curvature and height of neighbouring shafts vary
together within a seeded lock. The maximum guide length, main density
and individual radius budget stay at 58 mm, 67 candidates per square
centimetre, and 69 µm. The revised flow and shared lift should give the
scalp a clearer direction. That appearance awaits rendering.

## Separate sampling from shape

1. Render the new `scalp-fibre.yaml` at the existing 384 × 384 and
   three rays per axis, with `hairline` and `crown`. Keep the matching
   older images and bare controls. Look for a rearward sweep across
   the former radial patch, an exposed forehead with rounded temple
   transitions, and visible waves or lock direction at the crown.
2. For the same pinned binary, set and `hairline` camera, add
   `--rays 6` to a separate [Azazel request](request.example.yaml) and
   give it a distinct key and image, `scalp-fibre-hairline-rays6.png`.
   Keep output dimensions, filter, seed, materials and lighting fixed.
   At this framing, object-space pixel width near the scalp is roughly
   0.5–0.7 mm. Even the broadest shaft diameter is 0.138 mm. Six rays
   per axis halve the distance between projected samples. A reduction
   in speckle would support a sampling contribution. The position and
   extent of a coherent bare patch should agree between the two samples.
3. If the bright stipple persists at six rays, use
   [scalp-surface.yaml](scalp-surface.yaml) at the same six rays,
   saving `scalp-surface-hairline-rays6.png`. Its seed, geometry,
   camera and lighting are the same. The conventional surface material
   provides a diagnostic for the single-fibre appearance. It lacks
   the fibre model's directional transmission. Retain both results.

Commission each still through the
[shared Azazel queue](../../../doc/azazel.md), record the binary revision
and digest, and set the gallery count to the stills actually submitted.
The six-ray trace has four times as many primary samples as the three-ray
trace. Its actual running time is unknown. Strand intersections,
dynamics and between-fibre transport remain outside this groom.

## Hairline and lock refinement

Revision `2375d266` removed most of the radial gap and showed a
clearer rearward direction. Its six-ray front still showed the same
even wall of fine bristles and the same rounded temple protrusions.
The three-ray crown showed curves, but also coarse dark bands and
pale streaks. The sampling comparison supports a contribution to
the visible grain. The edge and silhouette persist.

The terminal shafts previously reached their regional coverage
over a narrow band. All shaft shapes raised their early segments
from that band, and volume largely followed chart height. Short,
fine shafts now extend across a separately seeded outer margin,
intermediate shafts overlap them, and terminal shafts fill in at
a staggered depth. The profile varies the edge along the scalp
perimeter at two scales and keeps the central part a weak density
change. The root boundary and the amount of visible forehead may
vary slightly by seed. The terminal roots lead away from the face.

Shorter side and nape guides use lower lift. The side guides follow the
nearest posed skin more closely, including in places where the skull
curves away from the initial shaft tangent. Neighbouring locks blend
their free-shaft targets and loft across the join. These controls
should round off the hairline less uniformly and reduce the wing-like
temple shape. Compare with the earlier three-ray hairline and crown to
see whether they do. The six-ray hairline remains a separate sampling
comparison, and the profile shows the side guide clearance.

## Crown and nape shape

At `7a119fd2` the temple protrusions are substantially smaller. The
profile turns continuously from the forehead into the nape and shows
recognizable short strands. Its lower edge is even and its side view
has dark parallel bands. The front has a steep rise to a flat-looking
top. The crown remains low in individual strand readability. The
profile bands and nape outline have structure at a much larger scale
than the finest speckle.

The next cut blends separate crown, side and nape lengths using the
scalp normal and anatomical position. The loft favours a rounded
crest and gets shallower going toward the sides. Lock attraction is
weaker on short shafts and overlaps neighbouring guides across a
wider transition. The nape gains independent irregularity along its
root boundary. A modest wave in the free part of the shaft gives the
crown another direction at the lock scale. Individual strands keep
their scalp attachments. Compare the new profile and front first for
the nape edge and crown-to-side silhouette. Compare the crown under
unchanged light and sampling for the broad bands.
