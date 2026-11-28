# Native adult human study

The studio sets use the procedural [adult human][human] in metres. The
neutral and reaching sets share a 1.70 m female reference with a short
scalp groom.
The neutral views hold the lens, subject distance and mid-body target
constant. A low-key background, broad fill and directional key let the
whole silhouette and the face, feet and hands read in the same frame.

| Set | Camera | Output |
| --- | --- | --- |
| [adult-human-neutral.yaml][neutral] | `front` | `front.png` |
| [adult-human-neutral.yaml][neutral] | `profile` | `profile.png` |
| [adult-human-neutral.yaml][neutral] | `three-quarter` | `three-quarter.png` |
| [adult-human-reaching.yaml][reaching] | `three-quarter` | `reaching.png` |
| [adult-human-male.yaml][male] | `front` | `front.png` |
| [adult-human-male.yaml][male] | `three-quarter` | `three-quarter.png` |

The reaching pose turns the thorax by 0.14 radians, counterturns the head
by 0.10, and turns the left clavicle, upper arm and forearm. The legs
keep the neutral stance. Follicles follow the final posed scalp triangles.
The hairline is picked for this reference from the crown, temple and
nape landmarks. Shaft curves follow the scalp as the head turns.

The skin retains its constructor's rest-chart pigment. The study adds
CPU lateral diffuse transport with RGB widths of 2.4, 1.08 and 0.52 mm
and a 0.24 share. Separate native eye globes follow the head frame with
a forward gaze. The skin mesh supplies the lid openings. The diffusion
parameters are illustrative. Sampling on mesh vertices can under-resolve
the shortest widths. The surface uses a single satin response and the
groom uses ordinary surface shading.
Detailed anatomical limits are recorded in the [constructor notes][human].

## Timed reach

The [reach clip][reach-clip] starts from neutral, reaches the static
study's endpoint at 1.4 s, holds through 2.0 s, and returns to neutral
at 3.2 s. The pelvis and legs keep their neutral turns throughout. The
same sampled pose moves the native skin and chest tissue, carries the
eye globes, and gives the groom its posed scalp triangles. The groom
keeps the same follicles and the skin keeps the same paint chart.
Edit the `reach` recipe's `pose-clip` keys directly in the set. Its small
aluminium marker follows the `left-hand` bone throughout the gesture.
The studio's original reach angles are retained in the editable keys.
See [authored pose clips][pose-clips] for controls and units.

Use the [film job][reach-film], or sample the set at a selected frame time.
For example, `--at 700ms` takes the midpoint of the outward reach and
`--at 1400ms` takes the first frame of the hold. Both use camera
`three-quarter` and the CPU renderer. This is an authored gesture with
fixed lower-body joint turns. Contact, inverse kinematics, balance and
walking cycles require separate authoring or solvers. Live deformation
currently supports one geometry sample per frame and refuses shutter
sampling.

For a shoulder comparison, draw `left-shoulder` at `0s` and `1400ms`
on both revisions with the same set file, CPU settings, and square
framing. Use 480 × 480 and three rays per axis. Compare the underside
of the arm where it meets the ribs and the outer shoulder silhouette.
Keep the original `three-quarter` view available to assess the gesture
and wrist attachment together.

The [shoulder support study][shoulder-support] adds the camera-side fill
and a six-degree left support turn. Compare `left-shoulder` at `1400ms`
and `2s` with its 480 × 480 CPU settings and three rays per axis. The
thorax, clavicle, arm and wrist marker stay in the same pose at these
two times. Inspect both the lower fold and the outer cap for a new ridge.
The support returns to zero with the rest of the pose at `3200ms`.

## Male anatomical reference

The male study uses the constructor's pectoral and external pelvic forms
with the same reference face and 1.70 m stature. The study picks a second
skin tone with melanin 0.57, vascular response 0.49, and gold and olive
undertones of 0.16 and 0.10. These controls and the existing regional
pigment are evaluated in the skin's rest chart. This set has no added
pigment marks.

The cropped groom uses the same authored scalp coverage and posed roots.
Its CPU material uses the [single-fibre model][fibre] with index 1.55,
longitudinal and azimuthal roughness 0.30 and 0.35, cuticle tilt 0.04
radians, and diameter optical depth of 0.72, 1.19 and 1.77 in linear RGB.
These are picked inputs for the portrait. The model resolves scattering
at each shaft hit under the studio lighting. It approximates circular
fibres and omits transport between strands. The groom has no collision
or dynamic simulation. The studio has one standing pose and uses the
same approximate lateral skin diffusion as the earlier scene.

## Face and extremity details

The [detail set][details] uses the same neutral female reference with a
bare scalp. Eye globes, body pigment and the illustrative skin-diffusion
profile come from the same studio material path. The cameras are
`head-front`, `head-three-quarter`, `head-profile`, `left-hand` and
`left-foot`. The three head cameras keep the same lens, aim and distance.
The extremity cameras show the outboard hand and the upper and lateral
foot, including the transition into the wrist or ankle.

Each detail frame is 480 × 640 at eight rays with a 31° horizontal
field of view under the existing studio lights. The constructor models
external surfaces and regional relief. The detail set has a relaxed
closed mouth, static eyelids and neutral limb pose. It omits internal
anatomy and dynamic contact. An opaque eye material approximates the
sclera, iris and corneal surface.

## Local backlight on the left hand

The [backlight set][hand-backlight] frames the neutral adult's left
hand and wrist from outboard. A local spot in the gap between the hand
and thigh sends light toward the inner surfaces of the digits. A weaker
front source makes the ordinary surface shading visible. The same
skin chart, pose and lateral diffusion are used as in the detail set.

For a matched comparison, stage the set once as written with
`adult-human-thin-skin`. Make a second copy using
`adult-human-details` as the only changed value. Use the
`left-hand` camera and CPU renderer for both. Keep the hand pose,
seed, framing, lamps and development fixed. Compare the digit pads
and the thicker wrist. The reserved share can darken diffuse
reflection independently of the light transmitted through the pads.

The example uses a three-layer optical template with a total depth
cutoff of 22 mm and a 0.20 reserve. These are chosen study inputs in
`manufacture/adult_human_study.go`. The result depends on mesh
thickness, the paired faces and whether the spot reaches the inner
surface.

## Local backlight on the left ear

The [ear backlight set][ear-backlight] uses the adult's bare scalp and
marks with an opt-in pinna. A spot behind the left ear is aimed at its
thin rear wall. For a matched pair, use camera `left-ear` once with
`adult-human-auricle-thin-skin` and once with `adult-human-auricle`.
Keep the CPU renderer, dimensions, lighting, seed and exposure fixed.
The illustrative transmission template uses a 9 mm thickness cutoff
and a diffuse reserve of 0.20. Compare the light on the free portion
of the pinna with the thicker head attachment.

For shape, compare `adult-human-auricle` to `adult-human-details` with
`head-front`, `head-profile` and `head-three-quarter` in the [detail
set][details]. Keep the same cameras, seed and studio lights. Inspect
the outline, rear gap and skin transition at the attachment.

## Female pelvic surface

The [pelvic set][pelvis] frames the lower abdomen, hips, and upper thighs
of the same adult reference in two static poses. The neutral specimen
reuses the detail figure. The second specimen turns each thigh outward
by five degrees about the forward axis at its own hip. The rig carries
the rest of each leg with the thigh. These poses show the external skin
and its deformation under moderate abduction. Contact and planted-foot
constraints are outside this protocol.

Both pairs of cameras aim at 0.80 m above the floor, with a horizontal
standoff of 0.92 m and a camera height of 0.74 m. The oblique views are
30 degrees left of front. The neutral-colour directional key, weaker
fill, and room light match between poses. Light levels, lens, and camera
placement are authored choices. Each frame is 640 × 600 at eight rays
per axis with a 31° horizontal field of view.

Submit four separate CPU draws using the queue recipe below. Use the
camera name as the output stem: `neutral-front`, `neutral-oblique`,
`wider-front`, and `wider-oblique`. Every request uses the same copied
`adult-human-pelvis.yaml` and `pictures: 4`. Record the integrated Slayer
revision, binary digest, and measured times in the gallery recipe.

```sh
"$PELVIS_STUDY_BINARY" draw "$PELVIS_STUDY_FOLDER/adult-human-pelvis.yaml" \
  --renderer cpu --camera "$PELVIS_STUDY_CAMERA" --fresh \
  --out "$PELVIS_STUDY_FOLDER/$PELVIS_STUDY_CAMERA.png"
```

## Put the stills in the gallery

Use the operator's shared [Azazel queue][queue]. Prepare a fresh folder
under `frames/` in the shared renders root. Copy the exact set files
there and include both the pinned Slayer build revision and the render
settings in `recipe.txt`. Use one folder per set of commissioned views.
Choose a date and folder name for the actual run and add a queued note
to the owner's gallery notes.

Submit one request per still. This is the request for the neutral
three-quarter view. Set the agent and absolute paths, and use separate keys and
output names for the other views. Each invocation draws one picture.
`pictures` is the intended PNG total in the folder for the submitted
series. Set it to 1 if only one still is being commissioned.

```yaml
agent: YOUR-QUEUE-AGENT
key: adult-human-study-neutral-three-quarter-v1
command: /ABS/PINNED-SLAYER
directory: /ABS/SLAYER-CHECKOUT
lane: tests
timeout_seconds: 600
inputs:
  - /ABS/RENDERS/frames/DATE-adult-human-study/adult-human-neutral.yaml
args:
  - draw
  - /ABS/RENDERS/frames/DATE-adult-human-study/adult-human-neutral.yaml
  - --renderer
  - cpu
  - --camera
  - three-quarter
  - --out
  - /ABS/RENDERS/frames/DATE-adult-human-study/three-quarter.png
pictures: 4
```

The neutral and reaching sets request 600 × 800 at eight rays. The male
set requests 480 × 640 at six rays. Both use a 31° horizontal field of
view. These scenes need the CPU backend for rest-space skin and fibres.
Keep the copied set files and versioned binary unchanged
while jobs wait. The shared publisher reads the job's output path to
index the folder and put completed pictures in `latest.html`.

[fibre]: ../materials/fibre/README.md
[ear-backlight]: ../sets/bioforms/adult-human-ear-backlight.yaml
[hand-backlight]: ../sets/bioforms/adult-human-hand-backlight.yaml
[details]: ../sets/bioforms/adult-human-details.yaml
[human]: model/bioform/human/README.md
[male]: ../sets/bioforms/adult-human-male.yaml
[neutral]: ../sets/bioforms/adult-human-neutral.yaml
[pelvis]: ../sets/bioforms/adult-human-pelvis.yaml
[queue]: azazel.md
[reaching]: ../sets/bioforms/adult-human-reaching.yaml
[reach-clip]: ../sets/bioforms/adult-human-reach-clip.yaml
[reach-film]: ../jobs/bioforms/adult-human-reach-clip.yaml
[shoulder-support]: ../sets/bioforms/adult-human-shoulder-support.yaml
[pose-clips]: model/bioform/pose-clips.md
