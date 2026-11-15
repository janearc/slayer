heard. - first still studies

320 x 180 pixels; two rays per axis on the traced scenes.
The title uses Slayer's titles over a generated black PFM.
Its only pixels before titles are zero: PF, 320 180, -1.0,
then 691200 zero bytes (little-endian RGB float32).

The summit comes from the selected 12 November low-Moon recipe:
renders/fable-team/sources/render-recipes/
2026-11-12-glass-hour-spooky/full/summit-low-set.yaml.
This study reduces the forest to 800 trees, all detail to quick,
and terrain sampling to 12 m. The original comments are retained,
preceded by the changes. Sources remain on the canonical shelf;
sources.yaml must accompany a copy outside the checkout.

The beam comparison translates four origins 20 m back along their
common boresight, preserving pointing, guide-star offsets and
display brightness. It is a static origin/occlusion study. It does
not establish attached launchers on articulated, moving instruments.
The close view deliberately crops the dome to inspect the slit.
The 0.5 m display radius is the lighting-reference value; the
narrower 0.15 m radius from the other reference is not used here.

The orbital is the existing far scene, at time zero, with the
renderer and musical gestures unchanged. It is not the final reveal.

Build the producer through Game:
  game run heard-build -- -ldflags "-X main.build=$(git rev-parse HEAD)" ./cmd/slayer

Copy the job and its set into a separate output folder per still,
with sources.yaml in their parent and the title's black plate present
as a folder named black beside the title's job.
Then, from the Slayer checkout:
  game lane gpu -- game run slayer -- draw /absolute/study/job.yaml

The jobs name their outputs with no folder. With SLAYER_RENDERS and
SLAYER_TEAM set, each still goes to its own dated folder under the
renders home's frames, and each film under its animations. Without
them the outputs are written beside the job.

The two films take their sound from b-m-a-s.opus one folder up, beside
shoot.yaml. The song is not kept in the repository.

Published studies: the heard studies of 2026-11-14, at 320 pixels,
under the shared renders' frames.

First producer: 55b9a23c. Its executable remained frozen throughout.
Each receipt records the actual recipe and image hashes separately.
The first summit invocation refused a missing source manifest; adding
it resolved the refusal. The first interior invocation refused an
unquoted colon in the scene name; the retained recipe corrects it.
No renderer code changed, and no broad test suite was run.

The first orbital framing clips the rings. Study 06 moves the eye to
5200 scene units (thousand kilometres) so the full structure fits.

720p motion studies

summit-animation-720.yaml and orbital-animation-720.yaml each ask for
48 actual frames at 1280 x 720, six fps, two rays along each pixel axis,
and eight seconds of original-speed stereo music from b-m-a-s.

The summit keeps its viewpoint and widens the lens from 60 to 62 degrees.
Forty seconds of wind-driven prescribed weather pass in eight seconds.
The instrument stays still; these are not yet searching mount animations.
At 720p, coarse bands in the mist shadows are visible. Keep this as a
diagnostic, not an accepted final-quality shot. It took 581.28 seconds.

The orbital has a gentle camera path and an authored sixty-second band
rotation. Its named scene writes day as the duration string 60s; the
ordinary job duration still uses {value, unit}. Its first staging attempt
refused the latter form for day, before drawing.

These jobs select gpu, the software traversal on the GPU. The summit's
original set selected gpu-native, which this job override replaces.
No fast-light switches are enabled. These timings are not a benchmark
of the native traversal or its cached-lighting path.

Omnipositor integration

summit-composite.yaml points to a fixed-camera version of the same summit
job. The held four-ray scenery plate stays at 720p while a one-ray 360p
pass computes the scene's scattered light and RGB transmission every frame.
This is an explicit preview approximation; surface lighting stays frozen.
Camera moves and zooms require fresh plates. See the compositor's SCENE.md.

The 320-pixel, one-ray, full-resolution decomposition control reconstructed
all 57,600 display pixels exactly. The cheaper 720p picture is a separate
visual acceptance decision. No broad test or general speed claim follows.
