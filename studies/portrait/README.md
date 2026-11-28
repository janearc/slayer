# SMPL-X portrait study

The checked-in geometry is the unmodified **SMPLX-female.obj** sample from
[Meshcapade's SMPL texture samples](https://github.com/Meshcapade/SMPL_texture_samples/tree/c87d871c7498fdc98f3cabb7341b5c85c38debaa/Textured_Mesh_samples),
revision c87d871c7498fdc98f3cabb7341b5c85c38debaa, SHA-256
c8b1d2a7aec2e0c7ad0a310d4af7d19e0c79f325fce0c8ae70321e525a12f640.
Meshcapade distributes the samples under the
[SMPL-Body CC BY 4.0 licence](https://smpl.is.tue.mpg.de/bodylicense.html).
The accompanying upstream declaration is in Meshcapade-LICENSE.md. Credit
Meshcapade and the SMPL-X authors (Pavlakos et al., *Expressive Body Capture:
3D Hands, Face, and Body from a Single Image*, CVPR 2019).
The portrait triangulates this sample, poses its neck and head, and adds
original pigment, iris paint, hair geometry, materials and studio lighting.

From the repository root, with Game configured as described in AGENTS.md:

    game5 lane tests -- game5 run portrait-check
    mkdir -p out/portrait
    game5 lane tests -- game5 run portrait -- -hairstyle legacy-bob -width 560 -samples 2 -out out/portrait/portrait.png

The surface-only frame is written alongside the frame with diffusion.
With -sss=false the command writes just the surface frame. The native CPU
tracer resolves eye geometry, hair occlusion, cast shadows and reflections.
The photograph uses a linear-to-sRGB conversion after shading.

The fixed sample supplies a face and two eyeballs. Its OBJ indices identify
the eyes in this particular topology. No licensed parametric model weights
are required. The two-bone portrait rig reuses bioforms/skeleton's joint
hierarchy, pose frames and blended skinning. The head rotates −0.095 rad around the vertical neck axis. The camera has a
0.303 rad horizontal field of view and a 4:5 crop. The key subtends 0.72 rad.
Pigment, brows, lips and irises follow rest coordinates returned by Slayer's
mesh. The bob has a swept cap and 720 deterministic tapered lock ribbons. Lash roots follow the aperture
where the original eyelids occlude each eye.

bioforms/pigment supplies a seeded, metric spot field shared by the cheek
paint and a tested bending fin fixture. Species-specific masks choose the
anatomical placement. bioforms/diffusion operates on visible tissue pixels,
with authored red/green/blue radii of 3.1/1.4/0.7 mm and a 47% diffuse share.
It filters unit-albedo diffuse lighting, gates by tissue visibility and
first-hit depth, and adds the lighting difference back through rest-space
albedo. This screen-space approximation cannot transport from hidden skin
or through the back of an ear. Its widths are artistic settings.

## Standing figure in a service alley

    mkdir -p out/alley
    game5 lane tests -- game5 run portrait -- -alley -camera alley -hairstyle legacy-bob -width 640 -samples 2 -out out/alley/alley.png

This full-length figure uses the same attributed adult body, eyes, scalp
and rest-space face paint. Separate materials describe torso and limb
colour variation in the same skin callback. The pelvis, spine, head,
upper arms, forearms, hands, thighs, shins and feet carry the figure.
Each foot's lowest posed sole vertex is grounded on the street plane.
This is an authored standing stance, without a balance simulation.

The passage contains steel-framed doors, utility conduits, hanging cables,
caged practical lamps, a drain, grime on running-bond brick, workshop
windows and irregular reflecting water. A warm practical, cool rim and
bounced fill make the silhouette legible. The camera sees a horizontal
field of 0.630 radians. Four primary samples per pixel feed the beauty pass,
and a separate one-sample irradiance pass drives visible-surface diffusion.

Independently adjustable figure inputs are -height-scale, -shoulder-scale,
-hip-scale, -mark-strength and -hair. For example, add
`-shoulder-scale 1.06 -hip-scale 0.96 -hair=false` to the alley command.
These are local breadth and height adjustments to the sample. The model
does not load the SMPL-X parametric weights.

The figure constructor accepts a skin callback and additional
bioforms/assembly.Part values. Each part carries its own rest vertices,
triangles, material and optional pose callback. A fitted garment can use
the body's pose callback. A rigid attachment can use the standing rig's
head, pelvis, left-wrist or right-wrist anchor. The constructor applies
the proportion fit to these attached parts too. Tests check the head and
wrist pivots, rigid lengths, both sole contacts, independent proportion
changes and paint interpolation on a posed attachment. The connected
skin mesh includes all four limbs. Omission currently works for separate
parts; an absent limb requires a validated replacement body topology,
closed cut surface and revised binding/attachment tests.

## Walk-between integration note

scene/thing.Keyed.At already interpolates root translation from position
keyframes at absolute time. A straight walk can use two endpoints, a
duration and Hold for the final pose. Heading must follow the horizontal
path tangent in the figure's root orientation. A translating rigid mesh
would slide its feet.

manufacture.MakeAtTime and session/retainedDeformation.thingsAt already
build fresh local geometry for an absolute timestamp and place the parts
together. A human gait can build one skeleton.Pose per sampled time and
pass its bone frames to the body, hair, garments and attachments. The
existing CPU deformation path accepts absolute-time sampling and excludes
GPU and shuttered jobs. The portrait study has no walk command today.

The theropod reference already implements a planted/swing cycle and a
two-link leg solve in manufacture/animals/vertebrates/dinosaurs/theropods/
allosaurus.go. Its TestAllosaurusPlantedFootWorldContact samples multiple
times and checks that the world-space stance foot stays fixed while the
root moves, and that the swing foot lifts. A human version should derive
phase from cumulative path distance and stride length, lock the world-space
sole through stance and solve thigh and shin angles back to the moving
hips. Verify both contacts against street height, no foot sliding, no
ground penetration, continuous transitions, exact endpoints, and matching
attachment positions when frames are sampled out of order.

## Proportions, gaze and figure cameras

The profile inputs shape this adult sample. Their authored dimensions are
figure-study variations, without fitting SMPL-X coefficients or classifying
measured people. Reference leaves the original proportions in place.
Compact shortens the leg segment 9.5% and the trunk 1.4%, widens the hip
band 8.5%, and increases local waist, thigh and trunk depth. Long-legged
lengthens the leg segment 11.5% and the trunk 2%, broadens the shoulders
9.5%, and narrows the hip band 3.5%. Head length and sole height stay fixed.
The width bands and depth offsets taper continuously in rest space. The
shared comparison uses the same source mesh, street, pose, lens and camera.

The default swept hairstyle combs 176 overlapping tapered locks from an
off-centre scalp part. Each has longitudinal texture coordinates. A fitted
underlayer is sampled by casting rays against the head. The legacy bob
remains available to reproduce the original portrait. The optional curls
study uses surface-seeded fibre groups. These models are fixed groom
geometry. They do not yet implement strand dynamics or hair scattering.

The prototype accepts a finite world-space point in metres with -look-at
x,y,z. It transforms the point into the actor's placed frame, solves head
yaw and pitch relative to the posed head, then directs each eye separately.
Limits are authored at 0.90/0.48 radians for head yaw/pitch and 0.49/0.36
radians for residual eye yaw/pitch. Distant targets behind the actor reach
those limits. Headwear can attach to the named head joint and follows its
rigid gaze rotation, including geometry extending below the neck.

Placing a figure adds its parts to the world and returns a suggested
camera named after the instance. The camera frames posed part bounds after
both part motion and world placement. Heading is measured in degrees about
scene up. Explicit study cameras take precedence. The placement result uses
a native Slayer camera path, and the scene-facing CLI prints which camera
it selected. The general set-YAML human/camera authoring path remains open.
The current solver frames the subject geometrically; it does not search for
a shot around walls or other occluders.

Run from the checkout root. The PNGs under out/variations are ignored outputs.

    mkdir -p out/variations
    game5 lane tests -- game5 run portrait -- -alley -camera alley -profile compact -width 520 -samples 2 -look-at=-.8,1.4,1.8 -out out/variations/compact-final.png
    game5 lane tests -- game5 run portrait -- -alley -camera alley -profile long-legged -width 520 -samples 2 -look-at=-.8,1.4,1.8 -out out/variations/long-legged-final.png
    game5 lane tests -- game5 run portrait -- -width 460 -samples 2 -look-at=-.46,1.57,1.02 -out out/variations/swept-gaze.png
    game5 lane tests -- game5 run portrait -- -alley -figure-name mara -at=.12,0,-.36 -heading=-21 -look-at=-.74,1.64,1.12 -width 440 -samples 2 -out out/variations/mara-camera.png

## Anatomy, skin and hair study

The control images in `out/variations` use the attributed, unmodified adult SMPL-X OBJ. The additional study keeps the same rest coordinates for painter and attachment lookup. Its stance settles the shoulders, flexes the elbows, rotates the palms toward the thighs and moves the pelvis toward the supporting heel. A smooth curl along each finger preserves segment length to first order. Each sole has a ground contact. These artistic adjustments share one skeleton across both proportions. Full SMPL-X has separately articulated hands and learned shape and pose corrections. SCAPE likewise distinguishes rigid articulation from pose-dependent changes in surface shape. This example uses the published static surface and Slayer's own rig. Shoulder and armpit muscle deformation remains an open modeling task. [SMPL-X](https://smpl-x.is.tue.mpg.de/) · [SCAPE](https://robots.stanford.edu/papers/anguelov.shapecomp.pdf)

One Catmull–Clark step on the original polygons rounds the planar transitions on the cheek, nostrils and eyes. It changes no distributed asset bytes. [Catmull and Clark](https://doi.org/10.1016/0010-4485(78)90110-0)

Weyrich et al. measured spatially varying surface reflection, albedo and subsurface scattering on human faces. Donner and Jensen model epidermal melanin, dermal hemoglobin and the surface oil separately. Here the face uses two independently traced surface responses mixed in rest coordinates, patchy superficial freckles and a coarser field of color across the cheeks, nose and ears. Wavelength-ordered visible-surface diffusion moves incident light farther in red than blue. Occlusion and same-tissue checks limit filtering. The radii, pigment weights and oil masks are artistic controls. Screen-space diffusion cannot recover light entering through the back of an ear or under an occluder. Jensen et al.'s single-scattering and diffusion model describes the physical transport omitted here. [Weyrich et al.](https://cgl.ethz.ch/Downloads/Publications/Papers/2006/Wey06/Wey06.pdf) · [Donner and Jensen](https://graphics.ucsd.edu/~henrik/papers/skin_bssrdf/skin_bssrdf.pdf) · [Jensen et al.](https://graphics.stanford.edu/papers/bssrdf/)

The layered groom uses side-dependent guide curves, overlapping locks, a narrow interrupted scalp line, short root hairs, tapered flyaways and color variation along the shaft. Marschner et al. measured directional highlights and internal scattering in hair. Longitudinal markings and rough dielectric shaft surfaces supply a limited visual cue here. A fiber scattering material remains future work. [Marschner et al.](https://www.cs.cornell.edu/~srm/publications/SG03-hair-abstract.html)

### Reproducing the comparisons

Use the Game tests lane and the portrait command above with these flags. Pass 0 regenerates the previous control. Pass 1 changes the stance. Pass 2 adds refined surfaces, complexion and visible-surface diffusion. Hair remains an independent input.

- Close-up control and current face share `-width 460 -samples 2 -look-at=-.46,1.57,1.02`. Add `-study-pass 2 -hairstyle layered` for the new face. Use `-skin-sheen original` or `-skin-sheen matte` to isolate the regional sheen treatment, and `-sss=false` to remove visible-surface diffusion.
- Alley figures share `-alley -camera alley -width 520 -samples 2 -look-at=-.8,1.4,1.8 -study-pass 2 -hairstyle layered`. Select `-profile compact` or `-profile long-legged`. The same explicit camera and pavement give a common physical scale.
- Supply a distinct `-out out/research/name.png` for each run. With diffusion enabled, the sibling `name-surface.png` records the native surface response under the same lights.

## Larson camp scale figure

To regenerate the bundled static mesh from the pinned study:

    game5 run portrait -- -camp-mesh ferruccio/examples/larson-camp-visitor.glb

The exporter triangulates the sample, applies the study's standing pose,
and generates smooth normals. Larson camp binds a native bronze material
and uses Oakland to put the lowest point on the pad and set the geometric
height to 2 m. The GLB and the adjacent notice carry attribution, licence,
and changes. The native scale mannequin remains available under its
existing constructor.
