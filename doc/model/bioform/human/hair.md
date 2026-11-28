# Human hair

The native study grows a short swept haircut from its indexed skin.
The previous study used a 7–10 mm cut and a simple threshold in height.
The new default allows a guide scale up to 58 mm at the crown and shorter curves at the
temples and nape. The arcs lift away from the root and settle toward the
scalp at their tips. Nearby shafts converge slightly as they grow out.
The side part changes direction along its length and retains some roots.
The flow turns around the rear crown while continuing down the scalp. Finer edge hairs and a sparse
set of lifted tips vary the outline. A shallow slope in front of the
ear replaces the narrow temple drop. The material selection stays
with the study, including the existing CPU fibre material on the male
study. The bare-scalp detail views keep their original geometry.

`primates.Human.Groom` accepts a `HumanHair` with independent `Scalp`,
`Body`, `Vellus`, `Eyebrows` and `Eyelashes` populations. A scalp profile
controls shaft density, length, radius, waviness, the lateral position
of the part and whorl direction.
Its hairline controls move the forehead and nape heights, recess the
temples, extend a central peak, or change the two sides asymmetrically.
A body profile can include forearms and upper arms, legs, torso,
axillae and the anterior pubic mound at independent shares. Each uses a
separate seed and chosen hair material. Regional masks and variation
are authored. These controls apply to either adult anatomy. The study
leaves body hair off. Beard and moustache growth are separate work.

## Fine hair

`HumanHair.Vellus` is an optional groom with its own material, seed and
shaft settings. A nil population or zero density makes no fine fibres.
A positive `Density` specifies candidate roots per square metre, and
`Length` and `Radius` set their scales in metres. `FlowTurn` rotates the
local growth direction in radians. Each of `Cheeks`, `Forehead`, `Torso`,
`Arms`, `Legs`, `Fingers` and `Toes` sets a share from zero to one.

The cheek mask fades below the eye opening and leaves the lips clear.
Digit roots sample the proximal skin triangles on the dorsal side of
their own branches. Palms, soles, nail plates and forefoot pads stay
bare. The same barycentric attachments follow the skin through finger
and toe flexion. The short curves have static geometry.

Measurements of vellus hairs on the cheek, forehead, chest, shoulder
and back found higher facial densities in the observed sample. A
separate skin-biopsy study found differences in follicle density and
dimensions across body sites. Both motivate regional controls. The
native masks, render density and diameter are authored.
[Vellus hair physiology](https://pubmed.ncbi.nlm.nih.gov/1993141/),
[Body-site follicle measurements](https://pubmed.ncbi.nlm.nih.gov/14962084/)

The [vellus study](../../../../sets/bioforms/groom-study/vellus.yaml)
uses 15 candidates per square centimetre before masking, a 2.6 mm
length scale and a 32 µm geometric diameter scale for close views under
raking light.

## Eyebrows and eyelashes

Each brow grows from a feathered band over the orbit. `Eyebrows.Left` and
`Right` set each band's rise, arch and fullness. Individual fibres start
more upright toward the nose, lean toward the middle of the brow through
its body, and lie down toward the temple. The two brows share a material
and density setting. A seeded draw varies the shaft angles and lengths.
The positions and shapes are authored. A morphometric survey documents
variation in brow height and shape with age in the studied population.
[Eyebrow morphometry](https://doi.org/10.1016/j.bjps.2019.01.011)

`Eyelashes.Upper` and `Lower` each accept a `LashLine` with independent
density, length, radius and curl. An absent line leaves that lid bare.
Roots stay on the exterior skin strip adjoining the lid rim. Their
barycentric attachments and the local tangent frame carry through
`Groom.At` as they do on the scalp. Eye aim turns the globe separately.
The present rig moves the lids with the head and holds their open shape.

The upper lashes grow longer and denser, with short shafts and fewer
roots near the corners. Length, tilt and bend vary along each row. The
study uses 6.2 mm and 3.5 mm upper and lower length scales before
applying that variation. In a sample of 50
Japanese adults, the longest lash in the central 2 mm averaged
7.33–7.47 mm for upper lids and 4.98–5.56 mm for lower lids across
the study's two groups. The groom dimensions and areal root count are
render settings.
[Japanese eyelash measurements](https://www.oatext.com/A-study-of-normal-eyelashes-in-Japanese-individuals.php)

The position-dependent variation follows observations from
multi-view reconstructions of real lashes. Brow and lash shafts use
separate materials and seeds. The small skin clearance is applied
along each shaft against nearby skin, and lashes have static curves.
[3D eyelash reconstruction](https://doi.org/10.1111/cgf.15040)

The [face groom study](../../../../sets/bioforms/groom-study/facial-hair.yaml)
uses the `adult-human-facial-hair` set kind. The three cameras frame the
brows and lids in front, three-quarter and side views.

## Roots and reference

The anatomical map is the native human's canonical material chart,
with +Y up and +Z forward. It covers the full human mesh, including body
branches. A profile reads the chart at the geometric triangle's
barycentric root. The chart's triangle edges transfer the authored
growth direction to the measured rest surface. The same root weights
attach to the posed skin. This lets a hairline follow changes in cranial
breadth or stature, and keeps body follicles attached through limb
motion. The shaft's length stays in metres; bending the skin carries
its root frame. The profile follows the existing vertex order, so
changing the mesh topology requires growing the groom again.

These render settings are artistic. The default scalp samples 67
candidates per square centimetre before applying the mask, with a
second, sparser population at the margin. At their widest, the main
shafts have a geometric diameter of 138 micrometres. This trades count
for screen coverage; it does not prescribe biological density or
diameter. The body defaults use about 3.3 candidates per square
centimetre before masking and region-specific scale factors. The
reference measurements below describe observed populations or body
sites with different sampling methods.

- Healthy adult Thai samples measured about 134–163 hairs per square
  centimetre across four scalp sites. Published healthy US samples
  reported different densities by scalp site and sampled population.
  [Thai trichoscopy study](https://onlinelibrary.wiley.com/doi/10.1155/2020/2476890),
  [US trichoscopy study](https://pmc.ncbi.nlm.nih.gov/articles/PMC6219221/)
- Rakowska et al. measured mean shaft thickness of about 57–61
  micrometres and a greater share of single-hair units in the temporal
  areas of their sample. This motivates allowing shaft widths and
  density to vary across the cut.
  [Healthy female scalp trichoscopy](https://pubmed.ncbi.nlm.nih.gov/21886722/)
- Measured hairline patterns include round, linear, peaked and recessed
  contours. The part is a hairstyle choice and the crown flow is an
  authored field informed by observations of parietal hair whorls.
  [Hairline morphology study](https://pmc.ncbi.nlm.nih.gov/articles/PMC8367035/),
  [Hair-stream and whorl study](https://pubmed.ncbi.nlm.nih.gov/871150/)
- Otberg et al. measured regional differences in follicle density and
  size across the back, thorax, upper arm, forearm, thigh and calf.
  Follicle counts include fine hair and cannot be read as counts of
  visible terminal hair. The body zones choose a sparse visible
  population whose dimensions and coverage can be set independently.
  [Body-site follicle measurements](https://pubmed.ncbi.nlm.nih.gov/14962084/)

## Other bioforms

Supply `hair.Surface.Chart` when the rest skin has independent anatomy
or paint coordinates. A layer can read `hair.Site.Chart`, carry a
chart-space heading with `Site.ChartFlow`, and use a conservative
`Layer.FaceFilter` to skip whole triangles outside its region. Roots
remain attached to the actual surface. With no chart, the rest space
serves both purposes. The human masks and flow fields are specific to
its canonical chart; a seal coat or another bioform can supply its own.

The scalp uses curves in the follicle frame, and guides the free ends
toward seeded targets in the chart. The tangent field follows the
chart normal and turns without reversing near the part. The model grows a piecewise polygonal
shaft from each curve. The nearest oriented posed skin face supplies a
small clearance at each sampled ring centre. This is a local correction
for short hair. There can still be intersections between samples, with
other shafts or with clothing. The groom has no dynamics, full collision
handling, growth cycles or scattering between hairs.

The initial render at `5905580c` showed a wide scalp gap at the part,
pointed temple margins, and a fine speckled surface. The retained bare
controls show the underlying contour. At `7fb6379f` the guides made the outline rounder and shortened the
scalp gap to a patch with radial spokes. The new tangent field and
lock-specific lift await comparison from the same cameras. The
[sampling comparisons](../../../../sets/bioforms/groom-study/SCALP-FLOW.md)
separate low strand coverage and directional shading from changes to
the underlying groom.
[Disney production hair pipeline](https://media.disneyanimation.com/uploads/production/publication_asset/121/asset/hairPipeline.pdf),
[Procedural styling with hair meshes](https://www.cemyuksel.com/research/hairmesh_rendering/Real-Time_Hair_Rendering_with_Hair_Meshes-Supplemental.pdf).

The current short cut uses separate fine, intermediate and terminal
populations around a staggered edge. Shorter side guides follow the
curvature of the posed scalp. The seeded clusters share shaft lift
and bend, with blended joins between neighbouring clusters. These are
artistic profile controls. Their effects on the temples, early shaft
direction and crown still need the
[matched scalp views](../../../../sets/bioforms/groom-study/SCALP-FLOW.md).

`ScalpHair.Length`, `SideLength` and `NapeLength` control different
parts of the cut and blend on the canonical scalp. `Curl` and
`Wavelength` control a bounded wave along the free part of each guide.
Zero curl gives a swept guide without that oscillation. Wavy cuts
can use finer shafts or a different guide scale without changing the
root sampling method. These controls cover short static cuts within
the stated size limits. They leave long hanging styles, tight coils
and motion to later work. The study keeps its existing material,
root density and camera settings.

## Parted medium cut

Choose `primates.PartedMedium` in `ScalpHair.Style`, or select
`adult-human-groom-medium` as a set kind. The zero style keeps the
existing swept crop. The medium guide sweeps over the scalp and
releases into a second curve. Free lengths hang around the ear
and along the rear of the neck. A narrow, irregular part changes
both root density and the side each shaft follows. The forehead
remains exposed.

The same length, side and nape controls have separate defaults
for this style. Length settings shape the guide rather than
measuring its arc. Local displacements carry through the
canonical chart and the rooted pose frame. The tail shape and
variation are chosen. The current implementation has no
gravity solve or collision constraints between strands. View the
[matched short and medium study](../../../../sets/bioforms/groom-study/MEDIUM.md)
for the intended comparisons.

The medium cut uses a narrower, irregular strip of scalp at the
part, a rearward sweep before the shafts separate, and staggered
lengths with neighbouring locks. Main shafts taper late along the
cut. The free ends turn toward the rear neck in the authored shape.
These remain static geometry choices. The
[medium study notes](../../../../sets/bioforms/groom-study/MEDIUM.md)
record the baseline and planned comparison.

For the medium cut, sparse guide targets now interpolate along
connected scalp regions. Each follicle retains its own position and
uses the same barycentric weights to read nearby guide influence.
Guide propagation stops at the part. Heading and release targets
blend on each side, with more of the local field retained at the
nape. This improves control over shared locks as a layer of geometric
authoring. It does not supply hair dynamics or light transport
between strands.

Near the front of the medium part, two guide handles now follow
nearby scalp faces before the shafts turn out toward their locks.
The targets follow the authored part on each side, then separate
gradually. Each target resolves to a lifted point on its own posed
triangle. The correction fades before the locks release. The other
medium regions and the swept crop still use root-frame curves.

The medium cut shapes loose lengths around connected scalp guides.
A strongly leading guide gives its nearest strands a shared bend,
cut length and end direction. Each follicle's position across that
guide contributes some width through the hanging length. Between
guides the paths return to the blended field. This leaves the part
and roots visible while giving the lengths and crown different
volumes. The temple margin carries finer, shorter strands with a
gradual change in length near the hairline.

`ScalpHair.MediumSweep` sets the amount of front drape for a parted
medium cut. It accepts zero to one. Zero keeps the more balanced
cut. With a centered `Part`, the drape extends on both sides. With
an offset part, the broad side bows outward and falls forward, and
the narrow side falls back behind the ear. Moving `Part` to the other
side switches the sweep. The medium study uses a left-of-centre
part and a sweep of 0.94. The edge shafts follow their existing
curves for less distance as their roots approach the hairline.

For the swept lengths, the curve descends into its release and
its final handle sits closer to the chart-vertical above the tip.
The broad side carries more depth and length. It takes less lateral
width at the cheek. Away from the part, some frontal hairline
strands also draw their first guide handle closer to a nearby
posed skin face. The resulting drape is authored in the rest
chart and follows the groom when the head moves.

Grouped follicle units are available to bioform grooms. Human
scalp histology reports grouped follicles and regional differences
in a Thai sample. The medium cut seeds its candidates independently,
as the first paired-root study produced visibly sparse streaks at
the crown resolution.
[Variation of Hair Follicle Counts among Different Scalp Areas](https://pubmed.ncbi.nlm.nih.gov/35118125/).

`ScalpHair.CrownLift` is an optional medium control from zero to
one. It pulls the later guide handle for some crown strands toward
a nearby surface guide. The distance and amount vary by root, and
other strands keep their base path beneath them. Zero gives the
base cut. The separate `adult-human-groom-medium-layered` set kind
uses 0.90. Both set kinds retain the descending sweep and can be
compared with the same cameras and seed.

The part opens over the frontal scalp. Farther back, a local flow
turns both sets of strands toward the occiput and around the whorl.
The share of each side's guide fades in that patch, and the outside
locks keep their own course. Root sampling and the authored density
mask are separate from the transition.

At the crown transition, some guides continue to follow the skin
as they turn to the rear. Adjacent roots turn at slightly different
distances and keep more of their spacing as they join the rear course.
Projection across the line is permitted only near its end where the
frontal part is already fading.

`ScalpHair.ScalpField` is an opt-in medium cut with connected
tangent directions across the scalp. The part separates the two
domains. Part, hairline, temple, nape and whorl constraints share
the same field. Scalp rings remain attached to posed skin
triangles. Hanging lengths continue from the last tangent in
each released strand's local frame. The study uses the existing
root density, shaft widths and optics. Its sampled field defines
the course while the ordinary cut's curl controls shape its
separate cubic guides.

With `ScalpField` enabled, set `FieldWisps` to a
`ScalpWispShape` to author shared volume and tip variation.
`Spacing`, `TipSpread` and `PartReach` are in metres.
`Volume` scales contact lift and free shoulder offsets.
The wisp preset uses 36 mm spacing, 27 mm tip spread, a
27 mm rear part reach and a volume scale of one. The field
continues across the crown behind the part. Shapes blend from
adjacent guides in the same side of the groom. Each fibre retains
its own root and course. A nil `FieldWisps` uses the first field cut.

`ScalpWispShape.BlendPart` carries a shared rearward heading
across the end of the part. The courses near the line follow its
bend over the crown before drifting across the former seam. The
wider transition near the whorl eases this drift into the side
courses. The wisp study enables this option. It leaves the sampled
root positions, shaft widths and authored hanging envelope in place.
