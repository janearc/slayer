# Humboldt squid

An authored adult Dosidicus gigas with a mantle length of 0.80 m. The length
is a design choice, not a maximum and not a representative individual. It is
built as a muscular hydrostat, with no vertebrate spine and no vertebrate
rig. Colour, scale and local deformation are choices, not a measured animal.

## What ships

`NewHumboldtSquid(phase)` in `manufacture/humboldt_squid.go` is the
squid. It is a Go function taking a phase of an authored pulse. The cornice
gargoyles use it at phase zero as the creature `squid`, a grotesque with the
mantle against the wall, the arms hanging over the corbel and the funnel as
the spout (`manufacture/cities/gargoyle-creatures.yaml`,
`manufacture/gargoyle_studies.go`).

Its length, proportions and coat are in the checked
[`squid-profiles.yaml`](../../../../manufacture/animals/molluscs/squid/squid-profiles.yaml) book. In Go,
`ReadSquidProfiles(path, bytes)` reads an independent book;
`book.NewHumboldtSquid(phase)` selects its named profile, and
`book.NewSquid(species, phase)` selects another entry. Additional entries
can change mantle length, mantle radius, head width, fin span, arm and
tentacle lengths, and coat. They share an illustrative body construction and
pulse; these controls do not establish the anatomy of another squid species.
Each checked profile owns its rest mesh. A missing species or invalid book is
refused.
Lengths are bounded to 0.01–100 m for stable construction; that is a numerical
limit, not a biological range.

The shared squid builder passes its posed surface to `NewMollusc`. A set can
use an `animal` recipe with `family: squid` and an optional phase in radians;
see [animal profile recipes](../../../animal-profile-recipes.md). It is not a
registered manufactory kind.

The earlier studies are preserved in Git history. They had no callers and
were removed with their tests to reduce the maintained code footprint.

Sources and what each gives are in [SOURCES.md](SOURCES.md).

## How the last study is put together

Positive x points toward the head. Zero is the mantle opening and the pointed
tail is at x = -0.80 m. The mantle's section is a smooth polynomial, chosen.
The skin is one closed, connected shell cut from a field with a head and a
sector-defined appendage field, and the details are separate:

- mantle with a circumferential pulse; broad terminal fins, with the FAO ratios
  as the target (about 0.45 m wide and 0.36 m long for the 0.80 m mantle);
- a head with a smaller eye than the earlier studies;
- eight arms and two longer feeding tentacles with elongated clubs;
- small pale sucker details, rooted on the nearest skin vertices, which are an
  illustration of limited resolution and not a species count;
- a separate hollow tube suggesting the funnel opening.

Rest coordinates are kept for the skin and the attached details, so painted
detail stays on the body. The skin was checked as one closed connected shell
of 28,688 vertices with rest-relative edge ratios of 0.81045 to 1.18955 over
eight phases, and a largest displacement of 0.04777 m for one tested pair of
phases. Those are geometric results and not biological tolerances.

The pulse is a stationary authored loop of mantle, fins and arms. The 2012
field study describes climb then fins-first glide. The loop does not copy it.

Tests: `TestNewHumboldtSquidCorrespondence`,
`TestNewHumboldtSquidRestTopology` and
`TestNewHumboldtSquidRestRelativeEdgeRange`.

## Known faults and unfinished parts

- No written comparison of the final study against FAO Fig. 293 and NOAA p.
  104 exists. The faults of the study before it, listed next, were being
  worked on and may or may not remain.
- The study before it had arms too uniformly thick and blunt, lumpy clubs,
  beads too small to resolve the paired arm suckers and the four-row club
  suckers, a hard mantle rim, a flat red colour, and no funnel. A dark fin
  seam might be a shading or sampling defect. A front camera foreshortens the
  arms, so their count cannot be seen there.
- No jet, funnel change, mouth, suction, muscle mechanics, photophore,
  collision, water contact or skin mechanics is modelled or tested.
- Whether eyes and appendages stay attached through all poses was checked by
  sampled correspondence only.
- Painted marks are not tissue folds. Teeth, beads and similar details need
  roots and an orientation frame, which the beads have only at their sampled
  vertex.

## Earlier builds

Five studies, each begun from a blank file. The third was found to have
rebuilt the first's fin and arm assembly from memory and does not count.

1. Parametric ring meshes: a mantle, two flat diamond fins, a ball head, a
   sphere funnel, ten pencil-like arms and tentacles with eight beads each.
   13 parametric meshes held their indices across phases, with 0.066 m largest
   displacement between phase 0 and a quarter cycle. It was separate pieces,
   with a hard mantle edge and cartoonishly large eyes.
2. One implicit field, 24,896 vertices and 49,796 faces, one connected
   component. Over eight phases 74,694 edge lengths ranged 0.59372 to 1.43910
   times rest, largest displacement 0.09565 m. Its eyes were black
   rectangles, fins short flat triangles, and the funnel a solid lump.
3. The uncounted experiment: 13 indexed parts, largest displacement 0.07929 m.
   The eye read better and the fins met the tail in top view, but arms were
   straight spikes and the funnel a nub.

3b. A voxel sculpture. Its first occupancy gave seven components; thicker
   tips gave one closed shell of 29,012 vertices and 58,020 faces. A first
   control lattice gave edge ratios of 0.00603 to 2.33332. At 35 per cent
   amplitude, 0.53530 to 1.46632 and 0.03506 m. Lumpy, with thick fused arms.
4. Bezier-path appendages in an implicit skin. A coarse field gave 41
   components until the sampling and thickness were raised. A pose with a
   branch cut gave edge ratios of 0.17322 to 15.33092. A continuous
   directional function gave 0.85762 to 1.18015 over 86,694 edges. The final
   shell had 29,214 vertices, 58,424 faces, one component, ratios 0.86297 to
   1.15317 over 87,636 edges and 0.03242 m.

   Its fin is a rounded superellipse.
   Its fin and head joins improved; its arms and clubs did not.

Lessons carried forward:

- Divide the work among mantle pulses, fins and flexible appendages, each
  controlled on its own.
- A connected shell can be had by thickening thin parts, but topology does
  not rescue the result.
- A branch cut in a pose function shows up as edge ratios beyond ten; use a
  continuous function.
- A detail that is invisible at the framing is not anatomy. Give the head and
  clubs their own view.
