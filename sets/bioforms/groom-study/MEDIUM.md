# Parted medium cut

The short study at `df07211b` has a rounder outline and softer temple
protrusions than `7a119fd2`. Its front and crown still read as a
uniform cap. Profile shows the sweep, with an abrupt boundary around
the ear and nape. The medium study uses a separate cut so its part,
hanging lengths and silhouette can be assessed independently.

[Medium fibre](medium-fibre.yaml) and
[short control](medium-short-control.yaml) use the same native neutral
adult, pose, complexion, lighting, fibre optics, and seed. The
medium cut follows a narrow off-centre part over the crown. The locks
release near the temples and hang alongside the jaw and upper neck.
Forehead and eyes are open. Each shaft starts on its skin triangle.

Use `front` to inspect the part opening and the side sweep, `profile`
for the ear transition and hanging tips, and `crown` for the root
direction and rear coverage. Compare the two sets at all three
cameras. The matched 416 × 480 frames use a 33° horizontal field of
view and three rays per axis. Use the ordered front and profile
pairings to start a small visual check, then the crown. Study
sampling is below strand resolution in parts of the frame.

The medium scalp uses 37 candidate shafts per square centimetre
before its mask. The widest shaft diameter is 152 micrometres.
Fine edge hairs and a sparse set of longer tips use different
seeds. These are authored display budgets. The comparison changes
style, populations and geometric shaft widths. It keeps the short
cut and its material available for the control.

Use the shared [Azazel queue](../../../doc/azazel.md), pin the
binary and copy both set files beside their outputs. Follow
[request.example.yaml](request.example.yaml) with the medium set
paths and the selected camera. Example stem
`medium-fibre-profile`. Distinct stems and request keys identify each
view. Use the actual number of commissioned stills for `pictures`,
six for the complete paired study. Keep the earlier short studies.

The cut is static and uses the existing direct single-fibre material.
It approximates clearance at sampled shaft centres. Strand or
garment collisions, hanging motion, tight curls and multiple scattering
remain outside this study. The authored views await their first
renders.

## Part and hanging-lock refinement

The first medium front and profile at `dc8ab401` show an open forehead,
a wide straight scalp stripe, and dark side masses that thin into
speckled tips. In profile the rear guides extend abruptly away from
the neck. Those images resolve the style outline. They cannot separate
individual shafts at the tips.

The first guide sent roots on either side of the part in opposite
directions before they swept back. The part also removed up to 74
percent of roots across a shallow strip. The extension now varies
laterally along the head, removes at most 32 percent of roots across
a narrower strip, and turns the nearby shafts back first. Its sides
remain independently rooted. The new front and profile should show
whether the part reads as a thinner, less regular strip of scalp.

The original general taper began 40 percent of the way along the
whole path and narrowed almost to a point. In the medium cut that
choice removed much of the last visible length. Main strands now
keep their radius until roughly 84 percent of the path. Fine fringe
and loose tips retain their finer taper. Neighbouring locks converge
toward staggered length and depth targets. At the back, the targets
turn toward the narrowing neck where the original targets extended
behind it. This shapes the actual strands with the same main density,
largest radius, optics and framing.

At the study's focal distance a pixel covers roughly 0.8 mm, several
times the widest shaft. The first matched comparison should use the
same three rays per axis. If the hanging edge still speckles, repeat
one frame with six rays per axis before attributing the remaining
appearance to strand shape. The clearance uses sampled centres and
the local face. The revision has no inter-strand collision or gravity
solve. Its revised part, silhouette and terminal coverage remain
unverified until the matched views are drawn.

## Surface guide interpolation

At `06ee8ef6` the lower locks are fuller and hang nearer the neck
than at `dc8ab401`. The front still has an exposed stripe wider
than the root mask alone predicts. At this viewing distance, the
mask's full authoring width is only a few pixels. The divergent
emergence directions likely contribute to the broader stripe.
The profile still shows strong bands through the side mass and a
stepped temple transition. Individual tips remain below pixel size.

The updated medium cut seeds sparse guides on the actual scalp and
blends their initial headings, release points and end points. Each
guide influences vertices it can reach along nearby mesh edges.
Edges across the two sides of the part are excluded. A shaft's
follicle keeps its own triangle and interpolates the guide weights
from the three corners. Close to the front of the part the tangent
turns to follow the scalp rearward before the guide bends sideways.
Fine fringe and loose tips use their independent curves. The cut,
materials, root density, taper and cameras retain the existing budgets.

This is a local graph interpolation inspired by surface-aware
production grooming. It uses shortest paths on mesh edges and a
bounded set of neighbours. It has no vector Poisson solve, and it
still uses sampled clearance rather than collisions. Compare the
matched front and profile to earlier medium views for the part
width and the temple transition. Use a six-ray frame separately
if grain prevents assessment. Neither sampling comparison proves
clearance under motion.
[Hair Emoting with Style Guides in Turning Red](https://doi.org/10.1145/3532836.3536253),
[Discrete Differential Operators on Polygonal Meshes](https://research.pixar.com/docs/2020.SiggraphPapers.GBD.pdf).

## Emergence at the part

At `8b5130e2` the profile is smoother than at `06ee8ef6` and
its locks are less distinct. The apparent front stripe is similar
in both. A horizontal colour sample near the front of the part
spans about 12–16 pixels depending on the skin-colour threshold.
The 3.1 mm full width of the partial density mask projects to only
a few pixels at this scale. The colour span also depends on
scattering, shadows and under-sampling. It cannot show which
follicles are present. The old control curves curved toward the
sides early, even where their initial heading followed the part.

The main and margin shafts nearest the front part now pull their
first two handles toward points projected onto the scalp on the
same side. The first handle follows the curvature of the part.
The second begins the outward turn. The handles move when their
triangles deform. The correction fades out before the locks release,
so the compared free lengths, taper, root mask and material use the
same settings. Loose and fine fringe hairs keep their own guides.

The matched front should reveal whether the early shaft paths
contribute to the apparent stripe. Match the profile to detect any
visible change outside the front of the scalp. The stepped temple
transition and fine lower edge remain separate visual questions.
Higher sampling can help distinguish the effect of the geometry
from speckle. These stills cannot establish clearance during motion.

## Lock shape and edge lengths

The smoother profile at `8b5130e2` also lost some lock separation.
The inherited guide field averages nearby release points and ends.
This gives it broad continuity, but leaves little coherent
variation through the free lengths.

The next medium cut uses that same field and adds a shaped lock
where one surface guide carries more influence than its neighbours.
Each lock has a shared bend, offset and length variation. The roots
retain their positions and the followers keep some separation in
the hanging length. The extra shape eases at guide boundaries and
around the front part. The temple margin uses a graded shorter
length. The controls affect the actual shaft paths. Density, shaft
widths, material, lighting and cameras use the previous budgets.

Compare the matched front and profile with the scalp-pin revision
as it enters the same queue. Inspect whether the outline has a
natural taper at the temple and grouped lengths through the neck.
The crown view helps judge whether coherent changes in lift read
as locks. Use an additional higher-sample still if the ends are
too fine to separate visually. This remains a static authored
style with sampled centre clearance.

## Side-part drape

In the matched `3ca7aa81` and `fc63c589` frames, the scalp opening
is narrower than at `8b5130e2`. The lock shift at `fc63c589` is
subtle. Both profiles still show a rigid temple edge and the front
has similarly shaped masses beside both cheeks. Individual ends
are difficult to distinguish at 416 × 480 and three rays per axis.

The study now uses a stronger side-part drape. Crown and front
follicles on the broad side feed an outward bend and longer
face-framing lengths. Their counterparts on the narrow side tuck
toward the back. Occipital sections keep the underlying medium
cut. The finer margin follows part of each original shaft path,
stepping down to the short fringe near the hairline. This changes
the mesh silhouette and keeps the same root mask, optics, skin,
seed and cameras. Native callers can set `MediumSweep` to zero for
a more balanced shape, or set the part near the middle to keep a
two-sided drape.

Use the matched front and profile to compare the two visible side
masses against `fc63c589`, then the crown to judge the transition
from root flow into the longer sweep. A higher-sampled view can
help distinguish coherent ends from grain. These are authored
static routes with sampled clearance.

## Descending sweep

The `a5985f7c` front shows the broad-side sweep, but its outline
bulges outward by the ear, then curls back at the ends. The profile
shows fuller lengths and their uneven fall. The front roots still
read as a smooth cap. The former sweep displaced both its release
and distal handle sideways, which reinforced the ear-level bulge.

The next cut keeps the depth and length difference between sides
and reduces the lateral displacement. The sweep comes off the
scalp on a more downward tangent, and the last handle moves
toward the chart-vertical above its tip. Nearby frontal hairline
roots can anchor their first handle on the next patch of scalp
within the growing region. The part keeps its own attachment
rules. The roots, mask, widths and material use the same settings.

Compare front for cheek-side volume and hairline flow, and
profile for release and terminal direction. The crown can show if
the edge flow breaks up the cap impression. Chart vertical is part
of the authored shape. The curves follow the head when it tilts,
with clearance sampled along each shaft.

## Crown groups

The `12b45d6e` front and profile show a smaller cheek-side bulge
and a straighter fall than `a5985f7c`. The upper hair still reads
as a smooth cap with regular root streaks. The profile shows some
bright specks at the temple and noisy ends. Those views cannot
establish how much of the fine texture comes from under-sampling.

The `c1dfdcfb` study kept the descending sweep and grouped its main
candidate roots into pairs at the same expected shaft density.
In the dense crown above the hairline and to either side of the
part, nearby follicles followed a shared arch using two projected
handles. The after view shows pronounced pale fan streaks beside
the part. With small expected counts per triangle, pairing roots
reduces the number of occupied triangles while preserving the
expected strand count. Both the tighter root placement and more
aligned paths can contribute to a sparse appearance. The stills
do not isolate their individual effects.

[Medium fibre](medium-fibre.yaml) now restores the independent
root placement and descending curves of the `12b45d6e` cut.
[Layered medium fibre](medium-layered-fibre.yaml) uses those same
roots and adds a later bend to a subset of eligible crown
strands. Neighbours differ in bend distance and lift. Most
strands retain their base course to cover the scalp below the
optional guided lengths.

Compare the two crown cameras from the same binary and seed.
Match the front and profile if the crown looks promising. The
part, body shape, materials, lights and film settings are shared
between the two sets. Both use static curves and sampled clearance.

## Part termination at the crown

The matched `834467f5` crown and profile show little visible
change from the optional lift. Both crown views show a warm
diagonal stripe with repeated streaks. The density mask already
eases out near the crown. Farther back, the two guide regions
keep drawing roots in opposite directions as if the part were
still present. This is a candidate source of the wide stripe.

In this revision, strands in the posterior continuation of the
part gradually borrow a common rearward course. Their release
and hanging targets turn gently around the whorl, and the
side-specific guides lose influence in that patch. Front directions
and side lengths retain the descending cut. Both study sets use the
same independent roots, mask, shaft diameter and material as before.
The optional crown lift remains available for a separate comparison.

Surface guide shape and seed can be varied independently of root
distribution and part masks in production grooming tools. Here the
change is confined to the course after the part. The earlier paired
root experiment used a fixed count. Observational work reports a
range of follicle counts per unit, with differences by age and sex.
[Epic's guide interpolation](https://dev.epicgames.com/documentation/en-us/unreal-engine/groom-interpolation-in-unreal-engine),
[Epic's grooming controls](https://dev.epicgames.com/documentation/metahuman/mh-groom-tools?lang=en-US),
[Scalp hair loss across follicular units](https://pubmed.ncbi.nlm.nih.gov/36815266/).

Compare the same seed and material first at `crown` and
`part-detail`, using the new camera with both binary pins.
Look for a shorter visible seam at the rear, continuous coverage
and intact flow along the frontal part. Use `front` and
`profile` to compare the outer drape. With the scene frozen,
repeat the close view at higher sampling if the warm stripe or
speckle is difficult to judge. Subpixel shaft visibility and
directional fibre reflections can contribute to grain. A seam
that moves with the guide change at both sample counts supports
a coverage explanation. Colour alone cannot identify which
follicles are present.
[Phantom Ray-Hair Intersector](https://research.nvidia.com/sites/default/files/pubs/2018-08_Phantom-Ray-Hair-Intersector/Phantom-HPG%202018.pdf),
[Importance Sampling of Reflection from Hair Fibers](https://jcgt.org/published/0001/01/01/).

### Staggered turn into the whorl

The `6a3bd5bb` matched crown and part-detail views show a shorter
rear seam. Its end still opens into a pale, triangular fan with
regular radiating streaks. Both revisions use the same roots and
sampling. The change in the large shape follows the guide edit.
The images give limited evidence about individual exposed roots
or the source of the remaining fine grain.

The guides now keep more spacing across the rear course. At the
part end, their first handles continue to follow the scalp while
their turn and travel vary slightly with location and strand.
Near the frontal part both sides keep their original path rules.
Behind the fading part the handles can follow the shared course
across its former line. The roots, density mask and optics are
the same as the prior study.

Compare `crown` and `part-detail` to `6a3bd5bb` for fan width,
seam length and continuity. Match `front` and `profile` to judge
the descending cut. A high-sample close view can help separate
a repeatable pale pattern from grain. The curves remain static
with sampled clearance.

### Complete scalp direction field

The shared crown strip from the earlier gallery comparison
improved a narrow part crop at the cost of new bright bands and
a broad exposed wedge at its boundary. The [field study](medium-field-fibre.yaml)
uses one connected direction field on each side of the part
across the complete scalp. The trace uses the same field at
the crown, temples and nape. The opt-in releases a shaft from
its last accepted scalp tangent into an authored fall. Near-zero
directions, including the authored whorl core, remain inspectable
through `WeakVertices` and release locally.

The graph smooths vectors in tangent planes. The trace advances
by at most 3 mm per step and reprojects to chart faces. The
render mesh attaches samples to those faces at each pose. This
is a static approximation. The graph and the sampled paths give
no collision or projected-coverage guarantee.
[Sketching Hairstyles](https://hongbofu.people.ust.hk/doc/sbim07_hairstyle_sketching.pdf),
[Vector field design on surfaces](https://hhoppe.com/proj/vfdesign/),
[Vector field design with singularity constraints](https://web.engr.oregonstate.edu/~zhange/vecfld_design.html).

Compare the field and ordinary `part-detail` at the same seed,
camera and sampling. Use the flat shaft view to assess coverage
beside the part, the shape of the whorl and any channels at the
release. Match the ordinary fibre view to check whether those
shapes read as continuous hair. The default keeps the proven
descending cut.

### Scalp field with shared wisp shapes

The compact fall reduced rear bulk in profile. The lower edge
still read as a curtain and the close-up retained its pale
part-end wedge. The [wisp preset](medium-field-wisps.yaml)
keeps the scalp field as the root course and gives nearby,
orientation-compatible guides shared lift, shoulder and tip
offsets. The offsets follow the child's local frame. Each shaft
keeps its own root and path. A finite part boundary tapers into
the crown before the whorl, and the same field continues on both
sides behind that boundary.

The design borrows the separation of coarse volume from detail
in layered hair meshes, the use of coherent local orientation in
wisp capture, and the separation of coarse occupancy from visible
strands in reconstruction. The envelope is authored, with no
visible shell geometry.
[Hair Meshes](https://www.cemyuksel.com/research/hairmeshes/hairmeshes.pdf),
[Structure-Aware Hair Capture](https://gfx.cs.princeton.edu/pubs/Luo_2013_SHC/),
[Neural Haircut](https://openaccess.thecvf.com/content/ICCV2023/papers/Sklyarova_Neural_Haircut_Prior-Guided_Strand-Based_Hair_Reconstruction_ICCV_2023_paper.pdf).

Match the `3141bf5c` field's `part-detail` flat and optical
views at the same seed and sampling. Compare `profile` with its
existing profile control for the shoulder, overall volume and
staggered lower edge. Look for visible transitions between wisps
as well as cover at the part. These views describe the static
authored geometry.
