# Orca

An adult cow orca, about six metres long, as one closed skin bound to a
skeleton. Turning the joints of the skeleton moves the skin. The same animal
is made three ways: swimming, as a skeleton alone, and spy-hopping.

It is an illustrative individual. Counts of bones come from specimen
records; shapes, sizes and markings are chosen, and the code says so in the
comment beside each number.

## Kinds and sets

- `orca-clean-05`: the animal in an authored three-second swim loop, driven
  by scene time. Set: `sets/bioforms/orca-clean-05.yaml`, cameras `side`,
  `quarter` and `close` (the head's skin).
- `orca-clean-05-bones`: the skeleton alone, at rest. Set:
  `sets/bioforms/orca-clean-05-bones.yaml`, cameras `side`, `quarter` and
  `hand` (the right flipper).
- `orca-clean-05-spy-hop`: the skin upright with the head raised and the
  flippers out. Set: `sets/bioforms/orca-clean-05-spy-hop.yaml`.
  `sets/bioforms/orca-clean-05-sea.yaml` places the same pose in an open sea,
  the animal standing 0.2 m down so the eye is about 0.95 m above the water.

The bones and the spy-hop are their own kinds because they do not change
with time, so a renderer that refuses deforming geometry can draw them.

## Built from

[SOURCES.md](SOURCES.md) lists each source with the value taken from it. The
code cites the skeleton sources by their numbers.

- Bone counts: a juvenile male specimen record (S1) gives seven cervicals,
  ten thoracics, twelve lumbars, ten ribs a side, eight sternal rib
  elements and the digits of the right manus. A vertebral study (S2) gives
  53 vertebrae in total for one specimen, from which the 24 caudals follow.
- Teeth and size (S3, S4, S5): 10 to 12 large recurved teeth in each half of
  each jaw, 10 to 14 per quadrant in another account, a blunt snout, adult
  females up to 8.5 m, dorsal fin to 0.9 m in females.
- Swimming (S6, S7): vertical motion concentrated in the rear third of the
  body, tailbeat 0.3 to 1.28 Hz.
- Appearance (S8, S9): black back, white underside, eye patch, pale saddle,
  blunt head, falcate dorsal, paddle flippers, notched flukes.

Reference images from NOAA were compared by eye and nothing was imported.
No mesh, rig or asset from outside is used.

## How it is put together

The skin is one surface cut from a field on a lattice: the trunk, the
falcate dorsal fin, the paddle flippers, the notched flukes, the melon, the
mouth and teeth, the blowhole, the eyes and the belly slits. Fin roots are
filleted into the body. Ridges, rake marks and scars are geometry. The
material gives a wet sheen.

- Trunk: fifteen cross-sections from the tail stock to the snout, each an
  ellipse. They are chosen for a cow of about six metres, with the greatest
  girth a third of the way back and a steep forehead over a short rostrum.
- Skeleton: skull, column in four regions (7 cervical, 10 thoracic, 12
  lumbar, 24 caudal), ten ribs a side with the first five reaching the
  sternum, the flippers' bones, and the pelvic vestiges. The sternal rib
  count is chosen from S1's eight elements. There is no bone in the dorsal
  fin or the flukes and there is no hind limb.
- Binding: each trunk point follows the skull or the nearest vertebrae, each
  by a share that falls from one at the vertebra's middle to zero a reach
  away. A flipper point follows the humerus, the forearm and hand being one
  stiff plate as in a cetacean.
- Swim: a wave of bending runs back along the lumbars and caudals, growing
  toward the flukes, with a three-second period. The skull does not move.
  The largest turn of a joint is 0.012 radian.
- Teeth: twelve a side on the skin, six to a jaw, which is fewer than the 10
  to 12 per half jaw of S3. The skeleton's rows have twelve a jaw. The skin's
  count is kept as drawn.
- Markings: black back, white belly with a hard edge, soft gray saddle and
  white eye patch, painted in rest space so the paint stays on the skin as
  it deforms. Reflectances are chosen: black skin reflects a few parts in a
  hundred, white about seven in ten.

## Using it in a set

Place the kind as a thing with a seed, an east and a north, as the sets above
do. Each set supplies its own studio, light and cameras. To draw one:

```sh
bin/slayer draw sets/bioforms/orca-clean-05.yaml --camera side --out orca.png
```

Deforming sets need CPU rendering and refuse shutters, lens rigs, cached
lighting, close-glow masks, and stand or ecosystem exclusions. The scene-clock
ocean path does not support regular terrain and things together. The swim
does not move the animal through the water: its placement is fixed.

## Known faults and unfinished parts

- Shapes, sizes and markings are chosen. No skull measurement, photograph or
  specimen of an adult female was reachable, and the counts are from one
  juvenile male and one other specimen.
- The swim is an authored cycle. It is not fitted to recorded kinematics, and
  fluke pitch is not separate from the body's bend.
- Deformation is checked by sampling, not guaranteed. The twelve-phase
  survey test logs edge-length ratios and signed volume; it does not prove
  the absence of self-intersection between frames.
- Initial bounds do not yet give a conservative envelope for every pose, so
  a coastal swim, with the animal travelling, is not supported.
- The skin has 6 teeth to a jaw against the 10 to 12 of the source.
- The adult female's source drawings show a curved flank blaze and rear flank
  lobes that the belly marking does not reproduce.

## Not settled

An evaluation of the first version of this design said it was not an
anatomically acceptable orca: a low dorsal hump, winglike flippers, a pointed
head, no mouth, and an eye that was a gray dot. That version was kept as
the original to build on.

The mouth, teeth, eyes, melon, falcate
dorsal and paddle flippers listed above were added afterwards, and no
written comparison of the present skin against the NOAA drawings has been
made. Read the page's anatomy as a study, not as a reference.

## Earlier builds

Two earlier lines of study, five builds each, were thrown away one after
another, each started from a blank file. Their code remains only as the named
scenes `orca-form-study` and `orca-bioform-study`. What they taught is carried
into the present animal:

- A body and fins that share one radial surface stay connected but are hard to
  shape one part at a time; fins need their own roots, filleted in.
- Paint evaluated in deformed coordinates slides. In one build, 244 of 45,016
  corresponding vertices changed colour between phase 0 and a quarter cycle.
  Carrying rest coordinates fixed it: at 241 matched ray hits the sampler saw
  the same rest coordinates and colour in both phases. Rest space alone does
  not stop a semantic mistake, such as belly paint landing on a fin displaced
  into the belly's height.
- Source cleanliness, connectivity, paint anchoring and anatomy are four
  separate checks. A build can pass the first three and fail the fourth.
- A ring-skin build of the swim, 12,194 points, measured a largest point
  displacement of 0.3354 m, edge-length ratios of 0.8493 to 1.1389 across
  twelve phases, a signed-volume change of 0.353 per cent, and a loop that
  returned within 5e-16 m. A fresh 720 by 480 one-ray CPU frame took 0.64 s
  and 88.7 MB at peak. These describe that build and not the present one.
- Earlier tail strokes were 0.42 m peak to peak in one study, and a 0.55 Hz
  review cycle within the published range was used for its eight-frame loop.
  Neither is a measured gait.

## Lineage

The model and its original motion tests were copied byte for byte from a
snapshot of that original, commit `4bc08724`. The kind,
paint, bones, face and poses were brought from a later skin study.
