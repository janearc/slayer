# Native groom studies

Compare the fibre groom with the same unadorned adult under neutral
raking light. Each comparison shares the body, pose, skin, eyes, seed,
heading, camera and exposure. The scalp series uses a swept part and
posterior whorl. The body series adds a separate, optional layer on the
arms and legs. It uses the same bare-scalp reference so the only change
within each body comparison is the limb fibres.

## Scalp

Draw [scalp-fibre.yaml](scalp-fibre.yaml) and
[scalp-bare.yaml](scalp-bare.yaml) for each camera. Both sets use a
34° horizontal field of view and 384 × 384 pixels at three rays per
axis. These are the intended readings for each crop.

| Camera | Inspect | Output stems |
| --- | --- | --- |
| `hairline` | Roots at the forehead, edge wisps and temple recession | `scalp-fibre-hairline`, `scalp-bare-hairline` |
| `part` | Separation along the elevated side part and lift near the roots | `scalp-fibre-part`, `scalp-bare-part` |
| `crown` | Whorl, rear directional lay and exposed skin | `scalp-fibre-crown`, `scalp-bare-crown` |
| `profile` | Cranial silhouette, ear clearance, nape and stray tips | `scalp-fibre-profile`, `scalp-bare-profile` |

The source, fill and rim stay fixed in world coordinates. Bright edges
and exposed roots can be compared against the ungroomed skin. Check
whether the boundary reads as individual strands from the skin and
whether crown and temple shapes remain distinct. Watch for disconnected
tips, clumping, visible skin intersections and sampling speckle.

## Optional body comparison

Draw [body-fibre.yaml](body-fibre.yaml) and [body-bare.yaml](body-bare.yaml)
with `forearm` and `shin`. Their stems are `body-fibre-forearm`,
`body-bare-forearm`, `body-fibre-shin` and `body-bare-shin`. The
29° horizontal field of view covers roughly 160 mm at either focus
point. These crops use 384 × 512 pixels and three rays per axis.
Raking light shows short fibres crossing the skin's highlight and
their softer coverage toward the wrist and distal shin. The crop and
bare control leave the limb geometry easy to compare.

The body recipe picks 6.5 candidates per square centimetre before its
regional mask and variation. It uses up to 14 mm shaft length and a
142 µm geometric diameter before the local profile scales both.
The scalp budget is 67 candidates per square centimetre before
masking, with a sparser fringe population, a small number of lifted tips, and
up to 138 µm maximum diameter. These are authored display budgets. The [model notes](../../../doc/model/bioform/human/hair.md)
give the biological references and limits.

## Queue recipe

Use the operator's [Azazel queue](../../../doc/azazel.md) and a pinned
Slayer binary from the revision containing these sets and kinds.
Use separate gallery folders for scalp and body. Copy the exact set
files beside their outputs and retain their names. Record the source
revision, binary digest, settings, and queue acknowledgement for
each image in the gallery receipt.

Submit one CPU request per set and camera, using the film settings
from the set. Replace the submitter name and absolute paths in
[request.example.yaml](request.example.yaml). Choose a unique key
and output path per image. Set `pictures` to eight for the full scalp
comparison or four for the full body comparison. Use the actual
number planned if commissioning a subset. Start with the matched
hairline and crown pairs to check framing and strand coverage before
scheduling the remaining scalp and optional body views. The template's
timeout is a ceiling for a first request, with timing still unmeasured.

Full-frame coverage at this pixel size is a visual diagnostic.
Individual shafts can be below a pixel. If the initial images show
sampling artefacts, keep the camera and lights fixed for any later
higher-sample comparison. The profiles generate static shafts with
rooted positions and use direct single-fibre CPU transport. Strand
dynamics, collision handling and scattering between fibres remain
outside the study. The scenes have been authored for queueing and
are awaiting their first renders.

The [scalp flow comparison](SCALP-FLOW.md) records the two earlier
prototypes and lists controls for separating the guide shape from
sampling and the fibre material.

The [parted medium study](MEDIUM.md) uses a wider matched head and neck
crop to compare the medium part and hanging locks with the short cut.

## Eyebrows and eyelashes

[facial-hair.yaml](facial-hair.yaml) supplies `front`, `three-quarter`
and `profile` cameras. For a bare control, copy the set and replace
`adult-human-facial-hair` with `adult-human-details` at the same seed.
The front view shows the shape and fibre coverage of both brows. Use the
front quarter to examine lash spacing and the side to check root
attachment, curl and clearance around the globe. The study uses the
ordinary surface materials and covers 640 × 360 pixels at four rays per
axis. Submit one Azazel request per view using the queue recipe above.

## Fine hair

[vellus.yaml](vellus.yaml) isolates the fine population with `cheek`,
`fingers` and `toes` cameras. For bare controls, copy the set and replace
`adult-human-vellus-study` with `adult-human-details`. Keep seed 27,
both lights and the three crops fixed. Submit matched CPU pairs through
Azazel at 640 × 480 and four rays per axis. Inspect fine shafts in the
raking highlights and their fade toward the lower eye, finger pads and
nails.
