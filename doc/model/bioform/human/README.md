# Native adult human

The standing reference is authored in metres, with +Y up and +Z forward.
The body has one indexed exterior. Arms grow from open shoulder rims, thighs
share a perineal bridge, and digits branch from their palms and forefeet.
A single chart through the torso continues into the neck and head.

Each construction station describes an exterior contour. The chin and jaw,
orbital region, nose and lips are deformations of the head chart. The chest
uses the existing thoracic wall with a new mobile tissue field, independent
left and right footprints, and relief evaluated in the same chart. A short
inferior transition represents the attached lower margin while a longer
superior transition spreads over the supporting wall. The tissue moves
relative to that wall under approximate gravity and shoulder pull.

The [NASA anthropometric sourcebook](https://ntrs.nasa.gov/citations/19790003563)
and [CDC adult body measurements](https://stacks.cdc.gov/view/cdc/100478)
describe measurement conventions and population variation. The stations are
sculptural choices for an adult reference, not a fitted sample or a composite
of population percentiles. The chest approximation follows the supporting
layers and regional transitions described by the [superficial fascia
dissections](https://pubmed.ncbi.nlm.nih.gov/30511967/) and the [inferior
attachment dissections](https://pubmed.ncbi.nlm.nih.gov/34998682/). Geometry
and motion coefficients are authored approximations.

The constructor exposes rest points and indexed faces for surface-bound
grooms. Named skeleton frames carry attachments. A posed result exposes the
final skin points after the soft-tissue correction so a groom and the visible
mesh can use the same deformation. Material choice is independent of posing.

## Skin tone and marks

`AdultHuman()` supplies the reference tone and an empty mark stack.
`HumanSpec.WithSkinTone` sets the melanin and vascular controls from zero
to one and a `Chroma` undertone whose red, gold and olive axes run from
−1 to 1. Melanin drives the main change in diffuse lightness. The undertone
shift preserves the luminance of the base skin before face and nail colour
is applied. The RGB coefficients are authored approximations.

Broad regional tones follow the material chart. Cheeks, nose and ears
carry more vascular colour. Palms and soles have reduced melanin
absorption. Pigment also changes gently over the forearms, knuckles,
elbows and knees. Measurements show variation among anatomical sites
and individuals, including across the face, palm and sole.
[Regional skin colour measurements](https://doi.org/10.1159/000211297).

Use [areolar tone controls](areolar-pigment.md) to vary the areola and
papillary centre independently on each side while retaining overall skin
tone and the native chest landmarks.

Use `HumanSpec.WithPaint` with the [bioform pigment package][pigment].
`Human.PaintPoints` exposes the material chart, in the same index order
as `Human.RestPoints` and `Human.Faces`. Author marks against those chart
coordinates, which are in canonical metres. The coordinates are shared by
every pose, so a mask follows the posed triangle even where the exterior
moves away from its neutral position. If selecting a point inside a
triangle, use the same barycentric weights on its three chart points.

The adult references also use gentle seeded mottling. Set its size and
strength with `WithSkinMottle(pigment.Mottle{Seed: 347, Scale: .062,
Strength: .13})`. Scale is in chart metres. `WithSkinMottleSeed(347)`
changes the seed alone. An empty mottle clears that variation and keeps
the anatomical tones. The finer scales fade at wider pixel footprints.

The default surface composites authored marks over the regional colour.
For example, use a seeded scatter field restricted to the front of the
face. The adult detail set uses separate fields for freckles and
blemishes so that changes in pose preserve their relative placement.

```go
cheeks := pigment.MaskFunc(func(p maths.Vec, _ float64) float64 {
    height := maths.Smooth(1.510, 1.525, p.Y) *
        (1 - maths.Smooth(1.544, 1.553, p.Y))
    front := maths.Smooth(0.035, 0.063, p.Z)
    return height * front
})
freckles, err := pigment.NewScatter(pigment.Scatter{
    Seed: 347, Spacing: 0.0036, Radius: 0.00125,
    Density: 0.72, Feather: 0.40, Region: cheeks,
})
if err != nil {
    return err
}
marks, err := pigment.NewPaint([]pigment.Layer{{
    Style: pigment.Freckle, Mask: freckles,
    Colour: maths.RGB(0.80, 0.65, 0.54), Opacity: 0.55,
}})
if err != nil {
    return err
}
spec := primates.AdultHuman().WithSkinTone(primates.SkinTone{
    Melanin: 0.48, Vascular: 0.52,
    Undertone: primates.Chroma{Gold: 0.17},
}).WithPaint(marks)
```

Pass the specification to `NewHuman`, pose it, and call
`HumanPose.Surface(nil)`. Use `pigment.NewPatch` for a blemish and
`pigment.NewStroke` or a custom mask for a scar or tattoo. Mark layers
affect diffuse colour only. Scar height and changes to the surface
roughness require separate modelling. Patch and stroke filtering uses
the posed footprint as an approximation to the chart footprint.
Supplying a custom surface material uses its own texture with the
same chart.

On CPU, opt into lateral diffuse scattering by passing
`primates.AdultSkinDiffusion()` to `WithSkinDiffusion` before building the
human. The studio and chest material studies use this profile. Edit the
three RGB widths or the diffuse share independently of complexion.
The [diffusion guide][pigment] describes the mesh sampling and backend
limits.

To add CPU transmission through thin features, construct a profile with
`thing.NewThinTissue(layers, share)` and pass it to
`HumanSpec.WithSkinTransmission` before `NewHuman`. Author the layer
template and reserved diffuse share independently of complexion, marks
and lateral diffusion. The zero profile keeps the default skin response.
`HumanPose.Surface(nil)` carries the selected profile to the posed
surface. A caller-supplied material uses its own optics.

The tracer uses the thickness of the posed mesh. It can admit a chord
across closely opposing walls, such as the two sides of a digit pad or
the free portion of an ear constructed with `WithAuricles`.
The share is reserved from ordinary diffuse reflection across the
material, including at hits too thick for the transmission path. The study
coefficients are illustrative. See [finite-layer optics](../tissue-layers.md)
for the transport limits and [the ear study]
(../../../native-human-study.md#local-backlight-on-the-left-ear) for a
matched scene using the native material.

[pigment]: ../../../../bioforms/README.md

## Proportions and anatomy

The authored exterior approximates two adult anatomical families.
`AdultHuman()` keeps the original female reference and `AdultMaleHuman()`
selects a male reference. `HumanSpec.Anatomy` chooses the chest and pelvic
form. The choice describes external physical features independently of
gender, stature, face, scalp hair, skin tone, pigment and pose.

`HumanSpec` controls the following measures relative to the authored model.
These are continuous sculpting bounds. A zero `AdultProportions` field uses
the reference ratio of 1.0.

| Control | Range | Meaning |
| --- | --- | --- |
| Stature | 1.45–2.05 m | Crown height in the standing reference |
| LegLength | 0.90–1.10 | Ground-to-pelvis height relative to stature |
| ShoulderBreadth | 0.91–1.10 | Width at the upper-arm pivots, with a transition into the trunk |
| HipBreadth | 0.94–1.12 | Transverse pelvis width and upper-thigh placement |
| CranialBreadth | 0.93–1.10 | Width of the cranial vault above the eye region |
| NasalBreadth | 0.88–1.15 | Width of the cartilaginous nose, alar wings, and nostril openings |
| NasalProjection | 0.83–1.16 | Forward projection of the cartilaginous nose relative to the adjacent face |
| MandibularBreadth | 0.90–1.12 | Width of the lateral lower jaw ahead of the ears |
| ArmDrop | 0.88–1.12 | Standing vertical distance from shoulder to wrist, with hand length preserved |
| ArmGirth | 0.88–1.16 | Cross-section scale through the upper arm and forearm |
| LegGirth | 0.90–1.13 | Cross-section scale through the thigh and calf |

Torso width and depth also scale the body in the global left–right and
front–back axes. Breast contour parameters remain local to each side. The
new local controls use the same regions and authored joint landmarks for
skin and skeleton. Changing leg proportion no longer stretches the arms
and hands. Vertical stature and pelvis placement still affect where the
shoulder and arm hang. Index order and neutral material coordinates stay
paired with the original topology across these controls.

These ratios were chosen to bound the model. They are neither demographic
percentiles nor estimates of joint distributions. NHANES measurements use
landmark and posture protocols, so a rendered silhouette cannot be directly
assigned a percentile. The [NHANES procedures manual]
(https://wwwn.cdc.gov/nchs/data/nhanes/public/2021/manuals/2021-Anthropometry-Procedures-Manual-508.pdf)
describes those protocols. The [NASA Human Integration Design Handbook]
(https://www.nasa.gov/wp-content/uploads/2023/03/human-integration-design-handbook-revision-1.pdf)
explains why different segment percentiles cannot be combined to represent
one person. Variation here does not reproduce population covariance or
predict mass or circumference. The allowed extremes and their combinations
await inspection for contact, silhouette and pose.

## External anatomy

The female chest has separate left and right attachments with a rounded
inferior return, upper fascia and axillary tails. The loaded meridian
controls the thickness through the breast and the inferior fold across
its contour. Set the width, projection, attachment height, areolar radius
and papilla dimensions with `HumanSpec.Torso.Left` and `Right`. Each side of
`HumanSpec.Breasts` adds the controls below. A zero setting uses the
reference. Positive outward values move away from the sternum.

| Control | Range | Effect |
| --- | --- | --- |
| RootOutward | −.024 to .024 m | Spacing of the tissue footprint along the rib cage |
| UpperFullness, LowerFullness | −1 to 1 | Relative projection above and below the centre |
| LowerPoleLength | −1 to 1 | Vertical extent of the tissue below the centre |
| AreolaOutward, AreolaUp | ±.015 and ±.035 m | Position of the areola relative to its reference root |
| PapillaOutwardTilt, PapillaUpTilt | ±.44 radians each | Lean from the local surface normal, bounded by papilla size |
| WithoutAreola | Boolean | Absence of areolar relief and pigment |

For example, combine independent size and shape controls into a subtly
asymmetric specimen.

```go
spec := primates.AdultHuman()
spec.Torso.Left.Projection = .069
spec.Torso.Right.Projection = .051
spec.Breasts.Left = primates.BreastSideForm{
    RootOutward: -.007, UpperFullness: -.30, LowerFullness: .35,
    LowerPoleLength: .26, AreolaUp: -.009,
}
spec.Breasts.Right = primates.BreastSideForm{
    RootOutward: .010, UpperFullness: .19, AreolaOutward: .006,
    PapillaOutwardTilt: .18,
}
```

The native female projection range is 0–.103 m per side. Zero gives a
flat tissue contour over the rib cage and pectoral wall. Use
`WithoutAreola` independently for absent areolar tissue. To depict a
healed surface mark, author a `pigment.Scar` stroke in the rest chart and
apply it through `WithPaint`. Draw the stroke on the intended surface,
such as the lower chest, the areolar margin or the axilla, using the
geometry and chart of that specimen. Scar relief and surgical outcomes
are outside this surface model.

These ranges describe the model's sculpting controls. They do not
encode shape classes or statistical norms. The mesh and pigment share
the same canonical areolar coordinates as the skin moves with the pose.

The [chest form study](../../../../sets/bioforms/chest-forms.yaml) uses three
authored specimens with matched lighting and front and three-quarter views.

The female proximal thighs grow out of a common curved saddle under the
mons. The anterior pad rounds below the hip plane and tapers into the
inguinal skin before the thigh branches. Closer samples follow the lower
pubic return. The two outer folds meet through a shallow blended contour
in the standing reference. Inner fold visibility varies with both sets of
fullness controls. The labia majora and minora, anterior hood and glans,
urethral and introital recesses, and perineum are continuous with that skin. The
vestibule and recesses are closed surface approximations. Internal canals,
contact and independent labial motion remain unmodeled. `HumanSpec.Vulvar`
varies outer and inner fullness separately, with a side-to-side asymmetry
control. The authored ranges are .72–1.32 of reference for outer fullness,
.65–1.50 for inner fullness, and −.25 to .25 for asymmetry. The [adult
measurements](https://pubmed.ncbi.nlm.nih.gov/15842291/) describe substantial
variation in individual structures. The [standard anatomical terms]
(https://pmc.ncbi.nlm.nih.gov/articles/PMC4440541/) inform the landmarks.
The dimensions here are sculpting parameters.

The male chest shapes the sternocostal and clavicular regions over the
same thoracic wall. The lateral fold tapers toward the upper-arm root and
moves with the shoulder. The nipples and areolae use the same geometric and
pigment coordinates as the main surface. For this family, `Torso.Left` and
`Torso.Right` control the individual pectoral and areolar forms through
their width, projection, height and papilla dimensions. The shape is
informed by [pectoral dissection](https://pubmed.ncbi.nlm.nih.gov/19291757/).

The male pubic skin branches into a shaft with an exposed glans and a
scrotal envelope with two lobes. The branches share a boundary with each
other and the pelvis. The corona, sulcus, closed meatal depression,
scrotal and perineal raphe, and the posterior perineal depression are
authored surface landmarks. The [external morphology nomenclature]
(https://pmc.ncbi.nlm.nih.gov/articles/PMC4440541/) informs the landmarks.
Authored dimensions are an example of neutral external anatomy. A
[prospective measurement study](https://pmc.ncbi.nlm.nih.gov/articles/PMC9809495/)
used the pubic bone and retracted foreskin as measurement conditions,
which differ from this visible skin surface.

The male pelvis uses the existing near-hip and pelvis frames. Independent
soft-tissue movement or collision under leg motion, foreskin variants,
testicular internals and the urethral and anal canals remain unmodeled.
Default colour reuses the shared adult skin stack. Add regional genital
pigment against `PaintPoints()` through the existing paint interface.
Each family has its own face index set near the pelvis. Keep attachments
with the mesh instance that supplied their indices.

## Face and extremities

The lower jaw sweeps from the chin toward the ramus ahead of each ear, with
a thicker masseter over its lateral body. The jaw grades into the neck on
the shared skin chart. The philtrum and lips share the contour of the upper
jaw. The lips meet along a relaxed closure line. The landmarks follow this
[3D facial anthropometry study](https://pmc.ncbi.nlm.nih.gov/articles/PMC10882018/).

Nasal stations define the sagittal sequence from the columella through the
tip, dorsum and root, with the alar sidewalls joined across each section.
The nostril margins roll into short vestibules that angle up and back. Each
vestibule ends in a closed pocket within the shared skin mesh. Projection
and base dimensions are authored with reference to [3D nasal morphometry]
(https://www.sciencedirect.com/science/article/pii/S0379073810003762)
and this [adult nasal anthropometry study](https://pubmed.ncbi.nlm.nih.gov/33710043/).

The midface rises within the default crown-to-chin height. The eye globes
move with their sockets, leaving a 114 mm pupil-to-crown height at neutral
gaze. This is an authored dimension informed by the [RAF head measurement
series](https://asma.kglmeridian.com/downloadpdf/view/journals/asem/62/1/article-p67.pdf).
Paint coordinates keep their canonical heights across the proportion change.
Additional rows resolve the lip closure and the underside of the nose.

The cranial stations put the greatest occipital projection above the ear,
with the lower occipital surface curving into the nape. The upper forehead
curves into a slightly anterior crown. Fore–aft and transverse dimensions
are informed by the head-length and head-breadth measurements in the
[ANSUR II report](https://dacowits.defense.gov/LinkClick.aspx?fileticket=EbsKcm6A10U%3D&portalid=48).
The sagittal profile is authored with reference to this [adult cranial
morphometry study](https://pmc.ncbi.nlm.nih.gov/articles/PMC8742967/).

The eyelids and surrounding orbital skin share one depth field, so the
surface across the brow, cheek and canthi follows the same curvature.
Curved upper and lower margins meet at the medial and lateral canthi. The
aperture follows the landmark approach of this [upper-lid contour study]
(https://pmc.ncbi.nlm.nih.gov/articles/PMC12132956/). The dimensions are
authored for this reference. `HumanPose.Eyes` supplies the separate posed
globes. A zero `Gaze` looks forward. Each eye can turn independently and
accepts an iris colour. The nominal globe diameter follows [adult CT measurements]
(https://pubmed.ncbi.nlm.nih.gov/25431659/) and the fissure width and
interpupillary spacing are informed by [adult periocular measurements]
(https://pmc.ncbi.nlm.nih.gov/articles/PMC9803744/).

The eye surface combines an outer corneal bulge, iris, pupil and sclera in
one opaque material with approximate colour and reflection. Blinking,
eyelid tracking under gaze, transparent corneal refraction, lashes, the
lacrimal apparatus and internal orbital structures require further work.
The default ears and the nail plates use local surface relief. Call
`spec.WithAuricles(primates.AuricleForm{})` to build external ears with
a rolled rim and a thin rear wall. The new surface shares head vertices at
the attachment and moves with the head joint. Each front surface includes
a conchal depression, antihelix and tragus, with a softer lobe below.
The authored dimensions and back tilt follow the scale of [adult external
ear measurements](https://pmc.ncbi.nlm.nih.gov/articles/PMC5550315/).
`AuricleForm` can vary height, width, lateral projection, back tilt and
left–right size. The canal is represented by a shallow, blind depression.
The skin uses the same rest chart and scalp follicles skip the ear faces.

Nasal passages beyond the vestibules, mouth interiors, facial expressions
and nail free edges require further geometry. The weighted rig has no
contact or collision solver, muscles, or fitted pose correctives.

## Studio observations

The first front, profile and reaching renders showed dark triangular
creases where the upper arm meets the chest. A similar triangle appeared
in the male study. The profile showed an abrupt inferior breast turn
and the front view showed a hard medial transition. Angular edges around
the male pubic join remained visible under three-quarter light.

The shoulder rim now rounds off in the torso chart. The proximal arm
starts in the direction of the adjoining skin and joins the arm profile
with a matched tangent. Shoulder skin influence and soft-tissue motion
cross the rim gradually. The breast profile spreads the inferior return,
moderates downward travel and follows the local surface at the areola.
The male pubic chart has an arched boundary and the shaft and scrotal
branches start along the shared skin tangents.

[Shoulder-elevation measurements](https://pmc.ncbi.nlm.nih.gov/articles/PMC8434297/)
reported localized extension near the posterior axillary fold in a small
group. The changes above use that observation to motivate a continuous
skin transition. They do not fit a motion or strain model.

The second comparison showed smaller axilla shadows in the front and oblique
views. A high outer shoulder bulge and a ridge along its join remained.
The profile still converged onto a pointed breast contour, and the front
showed dark, distinct inferior lobes. The male pubic root still had corners.
The next geometry used smoothly overlapping projection fields on the thorax,
a narrower deltoid and continuous radial shoulder tangents, and a common
pubic transition before the two distal branches. The anterior chest follows
the rib cage farther into its lower attachment under rotation. The shoulder
weight ramp meets the arm's ramp at the shared boundary.

`left-shoulder-support` and `right-shoulder-support` are optional helper
bones for the skinned shoulder cap and its adjacent folds. They are
children of their respective clavicles. The existing middle weight now
follows the helper, which has the clavicle's motion until it has a turn
of its own. Set a small extra rotation in `pose-clip.turns` to move the
skin through that weight. The upper arm remains a separate child of the
clavicle, so a helper turn leaves its wrist socket in place. The helper
pivot and weights are artist controls. The shoulder-motion measurements
motivate separate articulation, and supply no angle for this control.
[Braman et al., 2009](https://pmc.ncbi.nlm.nih.gov/articles/PMC2841046/)

During an arm raise, the shoulder cap now follows a separate clavicle
share. The lower rim follows more of the upper arm, with a longer fade
into the lateral ribs. The proximal arm starts with the rim's three
weights and eases into the upper-arm bone. The rest surface, its paint
chart and the named bone frames used by attachments keep their existing
coordinates. The new weights need matched renders of the shoulder in
neutral and reach.

The third comparison retained profile projection but lost the round return
and weakened the distinction between the two masses from the front. A
broad, dark wedge extended below the mons in neutral and reaching views.
The current geometry restores projection with a rounded meridian, a
narrower medial bed and a shorter inferior transition. The paired outer
folds now sit on a continuous saddle that turns into each proximal thigh.
Current changes await a new front, profile and reaching comparison and
close neutral views at more than one stance and tissue shape.

The first close neutral view showed a high ridge at the trunk-to-thigh
join and a deep V below it. The earlier recesses sat mostly behind the
anterior part of the saddle. Moving the relief forward made more anatomy
visible. The next front and oblique close-ups exposed angular, overlapping
silhouettes at the proximal thigh joins, where the same spatial relief
also affected the independent thigh curves.

The lower pubic skin now descends as an ordered shell before the thigh
split. The broad inguinal V, paired lower contours and narrow cleft are
separate features of that shell. Its anterior relief follows the pubic
skin. The medial-thigh adjustment is on the thigh surface alone. The early
femoral loft controls preserve downward order and keep each side of the
thigh join on its own side of the sagittal plane. This addresses potential
skin fold-over and normal changes seen as shading boundaries. The earlier
images alone cannot establish which triangles intersected.

The model approximates neutral near-contact through the skin shape.
Tissue contact, deformation under thigh separation and new close-up
acceptance remain unresolved.

The following close views reduced the hanging shapes but showed a shelf
and fan-like angular shadows. The pubic pad matched the trunk position
while adding a nonzero depth slope at the join. Its contribution now
reaches both neighboring surfaces with zero slope. The proximal thighs
use fixed angular meridians through one transverse chart, with separately
ordered height and radial depth. This avoids the independent sideways
turns of the earlier join. The new shape still needs matched front and
oblique renders and a view that distinguishes cast shadow from silhouette.

The ordered-thigh comparison changed the lower fan but left a shelf
and large angular patches above the split. The c3 camera-axis key
control changed shadow placement while those boundaries remained.
The shared-patch construction fits the lower trunk, pelvic rows, and
proximal thighs together over their indexed edges. An edge-weighted
membrane solve rounds the joins against fixed boundaries above and below
the patch and a rest-position term. The broad inguinal V, lower saddle
and hip volume come from the authored starting surface.

The fitted version still showed a raised block over a horizontal
undercut. The anterior relief ended in the last cuff rows, while the
underside relief started farther back. A measured centreline from the
mons into the perineum let the relief turn gradually from forward to
downward. That revision mapped each vertex onto the centreline by
spatial proximity.
The authored labial volume, hood, glans and recesses are fitted on the
same mesh edges, with the narrow central features held close to their
targets. The labial profile varies independently in outer and inner
fullness and asymmetry.

The model approximates standing near-contact with a continuous envelope
and closed recesses. Tissue contact and motion under thigh separation
remain open. New matched front and oblique views are pending.

The continuous-arc views removed the transverse undercut. The outer
pads still turned downward before losing much thickness, leaving
rounded ends with a sharp transition into the thigh shadow.
The outer envelope now converges from a wider mons and groin attachment
and retains forward projection through the lower turn. The inner folds
and vestibular features retain the full underside trajectory. The
descending inguinal attachment is part of the same skin, and its position
is separate from the narrow midline cleft. The outer and inner fullness
controls still set their own relief. Visual acceptance of this revision
remains pending.

In the subsequent matched views the broader upper contour introduced a
new horizontal shelf. The upper relief rose at the same arc position
across the widened skin, and a separate lateral ridge added to it.
The outer tissue now uses one oblique footprint per side. Each footprint
rises higher toward the groin, converges toward the lower mons and
reaches the supporting skin with zero slope. The transition and the raised
region are part of the same field. The authored peak thickness, fullness
and asymmetry controls remain. The inner folds and recesses follow their
own underside trajectory. Physical contact and the next visual comparison
remain open.

The oblique attachment study still showed a lower pad with an abrupt
edge near the thigh. The spatial projection could assign the same
course to surfaces on either side of the groin because it ignored the
skin path between them. The constructor now assigns course along the
indexed skin from landmarks on the anterior pubic rows and sagittal
bridge. It interpolates between those landmarks along mesh edges.
Outer relief tapers with skin distance from the pubic rows and inner
relief follows the narrower strip around the sagittal bridge. The
oblique footprint, forward outer projection, standing cleft envelope
and independent variation continue to define the authored shape.

The edge distances depend on the mesh's sampling. The constructor
approximates soft-tissue support and standing near-contact. It has no
collision or contact solve, and the recesses have closed floors. Matched
front, oblique and separated poses still need comparison. If the
boundary persists, a separate diagnostic view can color the two
support fields and show face normals under matched lighting to
distinguish the authored transition, mesh faceting and cast shadow.

The connected-skin study softened the upper attachment. The oblique
view still showed a block-like lower edge and pinching where the pad
meets the thigh. Relief at the same course shared a single sagittal
direction across the fold. At the thigh-facing side this direction
could slide skin along its attachment and create an undercut.

The outer and inner fields now displace along a normal derived from
the underlying fitted skin and averaged over its edges. The outer
layer retains an anterior bias. The inner direction follows the
local normal and the curved sagittal course into the perineum.
The model keeps the shared midline fixed and bounds inward
displacement for the standing cleft. The separate thickness profiles
and their variation remain in the geometry. This uses the authored
rest surface to approximate tissue direction. The bound cannot establish
contact or exclude intersections during posing. Visual comparison of
this change is pending.
