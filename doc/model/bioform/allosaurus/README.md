# Allosaurus

An authored walking Allosaurus fragilis about 7.4 m long, built in
`manufacture/allosaurus_bioform.go` as `AllosaurusStudy(seconds, at)`. The
motion is kinematic, not physical. It is a study of published bone
proportions with neutral invented flesh, not a likeness and not a
reconstruction of an individual.

## Where it is used

It is a Go function, not a manufactory kind, and no set under `sets/` draws
it directly. The cornice gargoyles use it: `manufacture/gargoyle_studies.go`
holds the study at time zero, and `manufacture/cities/gargoyle-creatures.yaml`
lists it as the creature `allosaurus`, a snarling head and neck out of the
wall with the lower jaw as the spout.

The function takes a time in seconds. It moves the root forward at about
0.47 m per second (0.56 m in 1.2 s) and returns the same animal at that
moment. It rejects a time that is not finite.

## Built from

- Target: the neotype USNM 4734, designated by the ICZN in 2023
  (iczn.org, case 3506). The 7.4 m length is an estimated full-body
  reconstruction, not a measurement of a complete fossil
  (doi.org/10.1007/s41513-021-00172-1).
- Not the neotype: the Smithsonian record `allosaurus-fragilis-marsh-1877`
  is a different specimen, USNM V8423, a pelvis, caudal vertebrae and
  hindlimb elements. Do not confuse the two.
- Osteology: Madsen (1976), Utah Geological Survey Bulletin 109
  (ugspub.nr.utah.gov, B-109.pdf), which draws several disarticulated or
  composite animals.
- Limb proportions: a different specimen, UUVP 6000, with a 0.880 m femur,
  0.730 m tibia and 0.375 m central metatarsal, from a comparative table
  (pmc.ncbi.nlm.nih.gov/articles/PMC3722220/). These lengths do not give
  stance height or soft-tissue outline.
- Forelimb: a range-of-motion study based on specimen HS0387 and compared
  with USNM 4734 (researchonline.ljmu.ac.uk, id 24278) supports three
  grasping digits with the palms facing each other, not turned as human hands
  are. The paper's abstract says Early Jurassic; its introduction and the
  Smithsonian record say the Late Jurassic Morrison Formation, which is
  what is meant here.
- Locomotion: an analysis of Allosaurus locomotion (J-STAGE, Biomechanisms
  10, KJ00004275222) estimates a horizontal trunk with a counterbalancing
  tail. It is an analytical reconstruction, not a recorded gait.
- Comparison image: the Smithsonian photograph of the USNM 4734 mount. It
  shows a crouching mount, so its angles are not evidence of a crouched walk.

Skin cover, colour and muscle profile are deliberately neutral inventions;
the fossil sources do not establish them. No mesh or texture was imported.

## How it is put together

- Body: one closed skin, tail tip through torso, neck and skull. It is cut
  from a rest-space lattice of cross-sections. The tail bends by locally
  rotated cross-sections; the neck and head follow the same map as the eyes,
  brow and lower jaw, so attachments follow the head.
- Legs: separately meshed three-segment limbs with a visible metatarsal and a
  digitigrade three-toed foot with dark claws. Hip height is near 1.65 m.
  Two-link inverse kinematics sets each leg.
- Forelimbs: two much smaller bent arms with three medial fingers.
- Gait: stance and swing alternate on the two feet. During stance a foot's
  offset cancels the root's travel, so the planted toe stays fixed in world
  space at a ground height of 0.095 m, and lifts only in swing.
- Parts: the body, jaw, limbs, toes and claws are each closed, and they
  overlap. The animal as a whole is not one watertight surface.

Neutral hide, claws and eye materials are named speculative in the code.
Gait cycle, posture, bend amounts and mass placement are authored.

## Using it

Call `AllosaurusStudy` from Go, or reach it as the gargoyle creature
`allosaurus`. The `scale`, `roll`, `pitch`, `yaw` and the held point of the
gargoyle row place it on a cornice.

## Known faults and unfinished parts

- The head is a smooth bulb. The skull's height, lacrimal ridges, long toothed
  jaw and nasal and orbit shapes are not represented, and the jaw reads as a
  detached stick.
- Shoulder and hip transitions are visibly separate, the torso is fishlike
  from above, and the toes, hands and orbital ridges need more work.
- It is not a likeness of Allosaurus fragilis. Lips, teeth and the eye's
  outside look are uncertain.
- Balance is not shown. Body volume is 3.168164 m3 at 0 s, 3.166612 m3 at
  0.5 s and 3.173193 m3 at 1.5 s, under 0.2 per cent change. With a
  uniform-density body alone, the centroid at 0.5 s is about (-0.062, 1.933,
  +0.012) m in local space, while the planted right foot is centred near
  z = -0.36 m, so the centroid lies outside the footprint of those toes.

  That says a static body-only proxy fails. It does not prove the animal
  would fall, since limbs, angular momentum and contact forces are not
  modelled. Do not present the gait as a solved walk.
- A planted foot's point contact does not show that every toe surface stays
  clear of terrain, and the stills do not show clearance from
  self-intersection.
- Cycle speed, foot trajectory and skin compression are unknown from the
  fossil. The motion is a coarse authored study. In one review film of 16
  frames at 480 by 320 and 8 fps (2.0 s) the side camera hid some foot
  separation.
- One 720 by 480, one-ray CPU still took 0.40 s and 78 MB at peak, in one
  run on one machine. It is not a benchmark.

Tests: `TestAllosaurusPlantedFootWorldContact`,
`TestAllosaurusRejectsNonfiniteTime`,
`TestAllosaurusClosedBodyVolumeAndDeformation` and
`TestAllosaurusSeparateMeshPiecesClosed` in
`manufacture/allosaurus_bioform_test.go`.

## Earlier builds

Five builds were made, each from a blank file, and the fifth is the one
kept. What each taught:

1. A lofted body with separate head and legs: foot contact slid because the
   scene stayed fixed while stance feet moved in x.
2. Separate jaw and brows, and a translating root with planted-foot offsets
   cancelling travel: the collar seam between neck and head remained.
3. Neck, skull and trunk as one loft: the skull and tail did not move
   relative to the torso, so it was not a spine deformation study.
4. A weighted spine with the head map carrying eyes, brow and jaw: a single
   centreline tail displacement sheared rings instead of rotating them.
5. Locally rotated cross-sections along the tail: the kept build.

What carried forward: a deforming spine with an attachment map keeps head
parts in place; foot trajectories are to be measured in world coordinates;
checking that separate parts are closed does not show a continuous skin,
believable anatomy or balanced locomotion.
