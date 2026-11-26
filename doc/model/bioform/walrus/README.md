# Walrus

An adult Pacific walrus bull (Odobenus rosmarus divergens) lying on land,
about 3.09 m from nose to tail. It is one hide over a skeleton, bound to the
bones, so a pose turns joints. It is a study, not a specimen: the length and
the mass are held to published figures, and every other proportion says in
the code whether it is read or chosen.

## Poses, representations and sets

All five selections are static. None animates with scene time.

- `pose: lying`: at rest, the chest propped on the fore flippers and the
  belly flat on the ground.
- `pose: hauled-out`: the same, with the neck raised and the head
  lifted and held level, by turning joints.
- `pose: swimming`: the body in line, fore flippers laid back, hind
  flippers trailing and swept to one side.
- `representation: volumes`: the outline and the flippers alone, in plain
  grey, for seeing the form.
- `representation: bones`: the bones alone.

`sets/bioforms/walrus-outline.yaml` places the first on pale ground under the
air's sky and a typed sun, with cameras `side`, `quarter` and `top`.
`walrus-outline-raised.yaml` does the same for the hauled-out pose. The
`walrus-fresh-5*.yaml` sets beside them are studio studies of each selection;
the one for the rest pose adds a `skin` camera for the neck and head. The named
walrus study scenes default to a 38 degree lens.

## Built from

Nothing was imported: no mesh, rig or texture. Reference photographs were
looked at and not used as assets. What was read, and for what:

- Size and mass: Animal Diversity Web says a bull "can be as long as 320
  cm", "weighing up to 1200 to 1500 kg". The Handbook of the Mammals of the
  World, on Plazi, says "about 315 cm" and "about 1500 kg". The COSEWIC
  Atlantic walrus report quotes Garlich-Miller and Stewart (1998) for an
  asymptotic standard length of 315.2 cm in bulls. Fay (1985) gives 2.2 to
  3.5 m.
- Shape in words: US Fish and Wildlife Service (fws.gov, 228.pdf): a heavy
  rounded body, short flippers, a relatively small head, downward tusks and
  neck folds. Alaska Department of Fish and Game (adfg.alaska.gov): adult
  males reach 12 ft, with hundreds of short strong whiskers, and a greatest
  girth nearly the nose-to-tail length. A circular 3 m girth would make a
  diameter of about 0.95 m. That is a rough check, not a measured width.
- Photographs, for comparison only: the public-domain USGS "single walrus
  side view" and "walrus water close" (a frontal head).
- Vertebrae: Pierce, Clack and Hutchinson (2011), J. Anat. 219(4):502-514,
  give 14 thoracic and six lumbar vertebrae, one rib pair to each thoracic.
- Fore limb: a measured Pacific female, museum specimen UWBM 35479, on a
  virtual zooarchaeology page: scapula 255.882, humerus 253.959, ulna
  265.384, radius 210.408, read as millimetres and scaled by 1.15 (chosen)
  for a bull.
- Flippers and hide: the OBIS-SEAMAP species profile, for fore digits
  "shorter after the first", longer first and fifth hind digits, a hide
  "thick, rough, and heavily marked with creases and folds", yellowish brown
  that turns pink to red out of the water, and many lumps on the neck and
  chest of older males.
- Whiskers: "400 to 700 vibrissae in 13 to 15 rows" (Kastelein, Stevens and
  Mosterd 1990, as quoted on Wikipedia's Walrus page); "some 480-odd"
  (California Academy of Sciences, California Wild, fall 2002); about 8 cm
  long and 3 mm thick (Guinness World Records).
- Ankle: Riedman (1990, p. 64), as quoted on Wikipedia's Pinniped page:
  phocids cannot turn the hind flipper forward "because of their massive
  ankle bones and flatter heels"; otariids and the walrus can.
- Swimming: Gordon's behavioural study,
  doi.org/10.1111/j.1469-7998.1981.tb03470.x, says the hind limbs are the
  main swimming propulsor and the fore limbs steer and add to it. A
  comparative morphology paper, pmc.ncbi.nlm.nih.gov/articles/PMC3196755/,
  reports an intermediate, mixed morphology. Stroke amplitudes and periods
  are chosen.

## How it is made

The bones are laid out first, on the shared `bioforms/skeleton` package:
seven cervicals, 14 thoracics, six lumbars, three sacrals and nine caudals,
eight breastbone segments, nine rib pairs reaching the breastbone, and
flippers of five digits with two bones in the first and three in each
other.

The cervical, sacral, caudal and breastbone counts, the sternal
ribs, the digit counts, the hind limb and every bone length not named above
are chosen.

Head, neck and trunk are one surface, cut from a table of stations from the
nose to the tail (`walrusStations` in `manufacture/walrus_outline.go`). At
each: the half width, the height of the back and of the underside above the
ground, and how square the section is. The rows are eased into one smooth
profile.

The whole encloses about 1.4 cubic metres over the ground, about
1400 kg at a density of 1.03. The underside lies a little under the ground
from the shoulders back, and the ground flattens it.

The flippers' flesh is laid on their bones and joined to that surface. The
face is cut into it: a hollow round each eye, the nostrils, the groove
between the muzzle pads. Creases cross the neck and shoulders, with a ring
round each flipper's root. The whole is triangulated on a grid of about
thirteen millimetres.

The skull carries the tusks, the bristles and the eyes rigidly. The tusks
have growth rings and shallow ridges and narrow to a worn tip. The bristles
are 14 rows of 17 a side, 476 in all, each about 8 cm and 3 mm at the root,
stiff and straight; their spacing is chosen to fill the pad. The hide is
bound to the bones by nearness. Its colour is read at each point's place at
rest, so it stays on the body as it bends.

## Using it in a set

Select the `walrus` species from a checked pinniped profile book in a
recipe, then place that recipe as a thing with a seed, an east and a north.
For a set alongside the supplied bioform sets:

```yaml
recipes:
  profile-walrus-lying:
    animal:
      family: pinniped
      file: ../../manufacture/animals/vertebrates/mammals/pinnipeds/profiles.yaml
      species: walrus
      pose: lying
```

The file path is relative to the set. Skin is the default representation;
bones accept no pose, and volumes accept an omitted pose or `lying`. See
[animal profile recipes](../../../animal-profile-recipes.md) for the shared
rules. The rest and hauled-out poses lie on land, so a set gives them ground,
as the outline set does with its own sky and sun. Draw posed skins on the CPU
renderer.

## Known faults and unfinished parts

- Only the rest pose has been looked at closely. The swimming pose is not
  checked against a swimming animal. No whole-body swimming reference was
  found, so flipper shape and placement are not validated.
- No table of widths and heights along a walrus was found, so every row of
  the stations table is picked.
- The graphics processor does not yet read the rest place when it paints a
  posed hide. Draw posed skins on the CPU.
- Neck folds, lip texture and bristle roots are cartoon approximations of
  the photographs. The tusks taper more sharply than the photographed ivory.
- Deformation is not audited for self-intersection or local strain. In the
  fifth build a tusk's root-to-tip chord changed by 0.127 per cent, so it
  was not exactly rigid.

## Earlier builds

Five builds were made, each from a blank file after a written comparison
with the photographs. The fifth is the line kept, and its skin has since
been recut from the station table.

1. A lofted trunk with separate closed fin pieces: disconnected, a very
   narrow head, tiny fins, 48 whiskers where the sources say hundreds. The
   trunk had 1,186 vertices.
2. A loft with openings and fins sewn to the boundary loops: connected, but
   80 unused vertices and pointed fins.
3. Ellipsoid masses with a shared-edge tetrahedral extraction: no unused
   vertices, but the eyes were buried and the hind fins fused.
4. An axial radius profile with capsule flippers and field-rooted eyes:
   visible eyes and nostrils, separate hind fins, a toy-like face.
5. A network of volumetric kernels with a field-rooted face: a connected
   deformable skin and visible facial landmarks. Its skin had 17,868 used
   vertices and 35,732 faces in one closed component, and a signed volume of
   1.7841 m3 at rest and 1.7841 m3 in a 0.8 radian turn. A side frame at 720
   by 480, one ray, two CPU workers took 0.63 s and 97 MB at peak.

What carried forward:

- Check the connectivity of the vertices the faces use, and count unused
  vertices as well. A closed surface is not proof against intersections.
- Test extreme poses and local strain. Conserved total volume can hide a
  shear.
- Put face features against the actual skin in the same deformation frame.
- Smooth unions can merge anatomical parts silently; compare several views.
- Do not infer fin stroke or soft-tissue movement from a single still image.

Earlier uncalled walrus studies and their tests remain in Git history.
