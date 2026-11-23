# Purussaurus

An authored giant caiman after Purussaurus brasiliensis, built from one
skull and from living analogues. Only the skull is fossil evidence. The
body, its length, soft tissue, colour, teeth and every movement are
conjecture, and the page that follows says which numbers are read and which
are chosen.

## Evidence and what it does not give

- Skull UFAC 1403: Aguilera and others (2006), Table 1, give a maximal skull
  length of 1,400 mm, a maximal width of 820 mm and a snout length of 910 mm
  (the paper as a PDF on factual.afp.com). These are bone extrema, not skin.
- Aureliano and others (2015), PLoS ONE, doi.org/10.1371/journal.pone.0117944,
  Fig. 2: the dorsal and ventral skull of UFAC 1403 and a lateral view built
  with another mandible, UFAC 1118. The pterygoids and teeth of UFAC 1403 are
  missing (their Table 1). The figure shows a broad, flat snout and an
  unusually large external bony narial opening. The exact soft nostril shape
  is unknown.

  Their body length of 12.5 m is a regression estimate, and the
  use of 1.4 m in that regression is disputed in a published comment on the
  paper. The studies use an illustrative body of 9.5 m instead.
- Living analogues, for control ideas only: Fish (1984), five juvenile
  American alligators, travelling lateral tail waves starting near the pelvis
  (wcupa.edu, 1984CopeiaKinematics.pdf); Reilly and Elias (1998), varied
  alligator walking postures (doi.org/10.1242/jeb.201.18.2559). Neither gives
  adult Purussaurus kinematics, speed or gait.
- A living broad-snouted caiman photograph (nativealimentos.com.br), as an
  anatomical analogue, not a depiction of the extinct animal.

The reference figures were studied and not imported as geometry or texture.

## What ships

Five Go functions, each a different construction, with tests, in
`manufacture/purussaurus_fresh1.go` to `purussaurus_fresh5.go`:
`PurussaurusFresh1` and `PurussaurusFresh2` take a phase, a yaw and a ground
height, and the other three take a phase, a yaw and a posture. They are not
manufactory kinds, and no set under `sets/` and no gargoyle uses them. The
last is `PurussaurusFresh5`.

## How the last construction is put together

One skin from tapered skeletal segments: an anisotropic axial chain, a
manually controlled broad cranial outline, and four splayed multi-segment
limbs with feet and spaced digits. A joint-centred deformation field carries
a lateral travelling wave down the tail, with limb rotations weighted between
shoulder or hip and the extremities. The head moves as a rigid frame. Swimming
and a low ground pose are different poses.

The ground plane sits 1 mm below the
lowest posed skin vertex. That removes the penetration (the first render
reached -0.9204 m against a plane at -0.82 m) but is not a foot contact or
support solver.

Ten small teeth are authored, rigid, and rooted at the nearest
sampled skin vertices. The scute texture is an invented regular grid in
posed local space, so the marks may slide.

Measured on the skin: one face-connected closed component of 8,752 vertices
and 17,500 faces, no unused vertices, Euler characteristic 2; length 8.833 m
(x from -4.773 to 4.060), shorter than the 9.5 m planning figure and not a
specimen measurement; head-region box 1.540 by 0.930 m, which is not the
fossil's landmarks.

Volume is 7.8997 and 7.8781 m3 at rest and posed; largest
vertex movement 0.1929 m; loop error 6.2e-16 m; edge-length ratios 0.89163 to
1.09949; absolute triangle-edge change median 0, p95 4.044, p99 7.509 and
maximum 10.837 per cent.

These are for one invented pose and are not tissue
strain. The worst tooth root is 0.0458 m from its nearest skin vertex.

A 720 by 480 one-ray CPU render with two workers took 0.75 s and 84.9 MB at
peak. The preview film is 8 frames at 4 fps over 2 s, enough for broad
sampled motion and not for smoothness.

Tests: `TestPurussaurusFresh5`, `...ShapeAndStrain`, `...EdgeDistribution`,
`...NoTopologyHandle`, `...ToothAttachments` and `...BoundsAndGround`, in
`manufacture/purussaurus_fresh5_test.go`.

## Using it

Call a constructor from Go. The five named study scenes use a 39 degree
default lens, and their review jobs set 38 degrees explicitly, so recorded
framing is unchanged. The defaults are matched within the family and
distinct from the walrus family, which keeps 38.

## Known faults and unfinished parts

- The head is a rounded rectangle with bead eyes and guessed nostril marks.
  The fossil's posterior and dorsal structure and its large bony narial
  opening are not mapped.
- The four limbs are mitt-like at this resolution.
- The scute texture is visibly unlike a living caiman's organic armour and is
  no evidence of a Purussaurus layout.
- Teeth are authored illustrations. The fossil's own teeth are missing.
- The skeleton's proportions, joint angles, range of motion and body size
  are not verified. Whether the animal swam, and how, cannot be known from the
  skull or a low-frame-rate preview.
- Self-intersection, per-foot support and fluid forces are not tested, and the
  film has no water.

## Earlier builds

Five fresh constructions, each replacing the last. The first four remain in
the source tree and were not kept as the design.

1. Joined ellipsoids with a tail shear: a closed connected skin of 8,662
   vertices and 17,320 faces, volume 8.5172 and 8.5174 m3, edge ratios 0.83292
   to 1.18873. It looked like a salamander with featureless limbs and no feet,
   and the head-region box was 1.380 by 0.894 m.
2. A superellipsoid head, a changing-section tail and capsule limbs: 9,450
   vertices and 18,896 faces, volume 10.3274 and 10.3261 m3, ratios 0.83240 to
   1.18210. A blunter snout, but bead eyes, fused feet, and mouth cylinders
   that stuck out. The head box was 1.400 by 1.101 m, with the neck merged in.
3. Explicit axial stations with weighted limb displacement: 9,070 vertices and
   18,136 faces, ratios 0.83796 to 1.28292, the worst stretch. Mouth on the
   skin, bulbous eyes, fused digits. Head box 1.455 by 0.910 m.
4. Density-kernel masses and joint-centred rotations: 8,470 vertices and
   16,936 faces, volume 8.9036 and 8.8845 m3, ratios 0.89619 to 1.10050,
   absolute edge change p95 3.830 and p99 6.692, maximum 10.381 per cent, for
   a largest movement of 0.161 m against 0.262 m in the third, so it is no
   controlled comparison. No teeth. Head box 1.432 by 0.986 m.
5. The current build. While making it, a four-digit union gave Euler
   characteristic 0, an unintended handle that closure and connectivity both
   missed. Removing the digit stubs gave 2, and enlarging them restored 2
   with the stubs kept.

What carried forward:

- Keep apart bone, other-specimen analogues and invented soft features.
- Compare close anatomical views as well as whole-body scale.
- Check unused vertices and Euler characteristic, not only closure.
- Measure local distortion together with movement amplitude. A matching total
  volume is not material realism.
- Solve attachment and floor clearance explicitly, and label tests that are
  only nearest-vertex or lowest-point approximations.
