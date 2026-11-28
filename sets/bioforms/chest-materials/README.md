# Native adult chest material study

The nine plates compare diffuse chest pigmentation on the same neutral
adult reference. Each set contains one native posed skin mesh. The
front view spans the upper torso to show the surrounding skin on both
sides. Each plate uses the same camera, exposure, grey environment and
two equal neutral directional sources. All use the CPU renderer,
528 × 480 pixels, four rays per axis, a 31° horizontal lens and the same
illustrative lateral diffusion profile.

| Melanin | Low regional contrast | High regional contrast | High contrast with changed local chroma |
| --- | --- | --- | --- |
| Light, 0.15 | [light-low-contrast][ll] | [light-high-contrast][lh] | [light-high-chroma][lc] |
| Medium, 0.47 | [medium-low-contrast][ml] | [medium-high-contrast][mh] | [medium-high-chroma][mc] |
| Dark, 0.85 | [dark-low-contrast][dl] | [dark-high-contrast][dh] | [dark-high-chroma][dc] |

Low and high contrast change the local melanin adjustments at the areola
and its papillary centre. Each keeps a small difference between sides.
The third column keeps the high contrast parameters and changes
red and gold chroma locally. Overall vascular tone and undertone, local
vascular controls, geometry, pose and the lighting stay fixed. Compare
the first and second columns within a row for contrast. Compare the
second and third for local chroma. Compare down a column for the
authored melanin response. These are artistic material cases on one
anatomy, without calibrated spectra or population assignments.
[Areolar tone controls](../../../doc/model/bioform/human/areolar-pigment.md)

## Azazel recipe

Start with `light-low-contrast`, `medium-high-chroma` and
`dark-high-contrast`. These three plates sample the range for an initial
visual check. The matched comparisons are in the full matrix.

Plan on about one to three minutes of CPU execution per plate, or
three to nine minutes for the first three and nine to twenty-seven for
all nine if run serially. This estimate uses recent native detail
renders at eight rays and allows for fixture setup. It excludes queue
wait and will change with host load. The study itself has no measured
timings yet.

Copy the selected sets into a fresh shared gallery folder. Record the
integrated source revision, pinned Slayer binary path and SHA-256 in
that folder's `recipe.txt`. Submit one request per plate, replacing
`CASE` with its file stem, `render-submitter` with your submitter name,
and the absolute placeholders with the pinned locations. Use `pictures: 3` for the initial batch or `pictures: 9` when
commissioning the complete matrix. Use a fresh output and unique request
key per plate. [Azazel submission](../../../doc/azazel.md)

```yaml
agent: render-submitter
key: adult-chest-material-CASE-v1
command: /ABS/PINNED-SLAYER
directory: /ABS/SLAYER-CHECKOUT
lane: tests
timeout_seconds: 600
environment:
  SLAYER_CPU_WORKERS: "2"
inputs:
  - /ABS/GALLERY-FOLDER/CASE.yaml
args:
  - draw
  - /ABS/GALLERY-FOLDER/CASE.yaml
  - --renderer
  - cpu
  - --camera
  - chest-front
  - --fresh
  - --out
  - /ABS/GALLERY-FOLDER/CASE.png
pictures: 3
```

[ll]: light-low-contrast.yaml
[lh]: light-high-contrast.yaml
[lc]: light-high-chroma.yaml
[ml]: medium-low-contrast.yaml
[mh]: medium-high-contrast.yaml
[mc]: medium-high-chroma.yaml
[dl]: dark-low-contrast.yaml
[dh]: dark-high-contrast.yaml
[dc]: dark-high-chroma.yaml
