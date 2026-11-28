# Native iris close-ups

Each set uses the same bare-scalp, neutral adult human pose and frames the
anatomical left eye. The fixed heading keeps the subject facing the camera
as the iris seed changes. The face, eye geometry, exposure and lights are
shared across the comparison. The world seed is also shared.

| Set | Iris palette | Left iris seed |
| --- | --- | --- |
| [Brown](brown-72.yaml) | Brown | 72 |
| [Hazel](hazel-72.yaml) | Hazel | 72 |
| [Blue](blue-72.yaml) | Blue | 72 |
| [Hazel variation](hazel-109.yaml) | Hazel | 109 |

The three seed-72 cases compare pigmentation with the same radial pattern.
The two hazel cases compare an independent pattern at the same pigmentation.
The fellow eye uses the same palette and a separately derived seed.

Both views aim at the front of the left globe from roughly 84 mm away.
`front` looks along the forward gaze. `oblique` moves the camera twenty
degrees toward the temple and shows the globe and lid from the side. The
24° horizontal field of view covers roughly 36 mm at the aim point. Each
image is 768 × 480 with four rays per axis, using a neutral directional key,
a weaker fill and ambient light. Lights, lens, placement and seeds are
chosen for comparison. Use the CPU renderer for the rest-space materials.

Use the ordinary `slayer draw` command through the
[Azazel queue](../../../doc/azazel.md). Pin one built Slayer binary for the
series and use one request per view. For each request, copy the corresponding
YAML beside its output and pass these arguments to that binary. Use a
separate output for every set and camera.

```yaml
args:
  - draw
  - /ABS/GALLERY/brown-72.yaml
  - --renderer
  - cpu
  - --camera
  - front
  - --out
  - /ABS/GALLERY/brown-72-front.png
```

The human cornea is an opaque exterior with scene reflections. The model
has fixed pupils and static lids. The shots can show how the procedural
markings and the exterior reflection read at close range. Corneal refraction,
a separate tear film, pupil dilation and blinking are outside this study.
See the [ocular model](../../../doc/model/bioform/ocular.md) for sources and
for the distinction between measured targets and chosen appearance values.

The original brown and blue front views showed conspicuous radial spokes,
isolated dark marks, a hard limbal ring and matching gray arcs in the
sclera. Varying the iris fields across the radius broke up the spokes in
the next views. Seating the lining against the globe shrank the dark
corner pocket. The front and oblique still showed a dark upper lid rim,
gray bands on the sclera, and fibres with a coarse, wavy appearance.
The revised lid clears the actual moving cap with a thinner roll. The
opaque globe uses a limited diffuse scattering approximation, and the
iris samples narrower, less wavy filaments. Compare the new front and
oblique under the original lights. Directional sources may still cast
hard shadows at the lid. The revised geometry and materials await
render inspection.
