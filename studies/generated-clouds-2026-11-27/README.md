# Authored generated-planet clouds

Three matched quarter-phase Caprica views use the same seed, camera,
exposure, shell base (2 km), top (9 km) and extinction (0.5/km). Only
the authored cover changes: zero, 0.3 and 0.85. The original source was
`set/testdata/caprica-lava-drop.yaml`; the cloud settings are made up for
this comparison. Jobs 190–192 were submitted to Azazel with Slayer
integration tip d7a0b207f. Their submitted copies are in `/private/tmp`;
these copies use relative set paths for later reuse. Camera location is
[-22826400, 7411200, 0], looking at [0, 0, 0], in metres. The images
use the same 800 × 800 film, exposure 0.003, and one ray per axis.

These are comparison frames, not a visual acceptance of the cloud model.
The 0.3 and 0.85 frames both read mostly overcast; the bright limb loses
texture and contains dark flecks. The sparse 0.06 view probes whether a
lower setting makes the intended difference clearer. The seed-2 view holds
the 0.3 settings, the dry pair holds everything but cloud cover, and the
pole views share a close horizon camera at [0, 0, 6434000] looking at
[-893770, -436100, 6538530]. The sparse view still reads as mostly
cloud-covered; the dry-world override changes the view and the second
seed changes the pattern. Neither pole view shows readable cloud relief.
The oblique pair changes that camera's target to
[-893770, -436100, 6200000] and uses two rays per axis to seek a clearer
view of the deck; both are smooth gradients with no readable relief.
The nadir pair at [5200000, -3000000, 2600000], looking at the centre,
is also nearly featureless. The camera is nominally 120 km above the
generated planet's 6422 km radius, above the 2–9 km shell.

The `bands-*` views compare the authored latitude-weather option at 0.06,
0.3 and 0.85 against the earlier locked-weather quarter views, using the
same camera and seed. They use integration tip 47ed02b4d and await visual
grading. Locked weather adds a 1.2 blanket at the substellar point before
the input coverage and storm hearts; its designed appearance, therefore,
is not controlled by the cover value alone.

The `turn-zero` view uses the real authored `storm_turn: 0` option at
integration 6798d528d. It removes the polar rings in this view, but also
removes the visible vortices; it is an artistic option, not a storm fix.
The matched 4-ray `bands-middle` and `soft-bands` views used binaries
47ed02b4d and ee5748502 respectively. The latter reads the fine billows
before storm rotation and visibly softens fine scratches, though broad
rings remain. The experimental `soft-bands-noarm` job used 06ea08ac7;
removing the periodic arm-cover term gave only modest change, and that
candidate was not integrated. The `soft-bands-noturn` job used 7a67ab094;
it zeroed storm turn for measurement and removed the rings and vortices.

The `weather-giant-*` reference is the existing Melancholia-derived gas
giant, not a matched comparison with the rocky world. Its still sets
weather moment to 1e7 seconds and omits film-only pace and turning; the
long-shutter copy changes the physical rig from 0.004 to 1 second. The long-shutter frame in the November 27 render archive includes
`light.pfm`.
The normal PNG remained visually black. A separate `slayer develop` print
with knee 0.8, top 4 and an extreme +16 stops reveals a dark globe with
blue-gray streaks and bright flecks. It is a diagnostic, and cannot support
a direct comparison of cloud appearance. The draw with the same print
specified in its job is byte-identical to developing its own raw with that
job: both PNGs have SHA-256
`df0bc3c74a11dcf0b17b2352ad98c8ab7818a554af2ea27a0dcba4521d1aff35`.
