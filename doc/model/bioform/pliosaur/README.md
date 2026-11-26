# Pliosaur

An authored pliosaur after Pliosaurus kevani, about 8.5 to 9 m long, with
four flippers. It is a composite. Only the skull of the one known specimen is
a source; the body, tail, flippers, eyes, teeth, colour and every motion are
authored.

## Evidence and what it does not give

The target is DORCM G.13,675, the only known specimen of the species, a skull
with no associated postcranium.

- Benson and others (2013), PLoS ONE, doi.org/10.1371/journal.pone.0065989,
  Figs. 2 and 3: a dorsal midline skull length of 1.995 m, a preorbital
  length of 1.130 m, a temporal width of 0.730 m, and a premaxillary end
  expanded then narrowing behind it, with a greatest width of 0.215 m. The
  anterior symphysis is partly reconstructed.
- Foffa and others (pmc.ncbi.nlm.nih.gov/articles/PMC4111928/): about 70 to
  80 cm between the jaw joints. The 0.730 m temporal width and the 0.80 m jaw
  joint width are different landmarks and must not be treated as one.
- Muscutt and others (2017), physical flipper experiments
  (pmc.ncbi.nlm.nih.gov/articles/PMC5577481/), Fig. 1: phase-dependent thrust
  enhancement is possible. The flipper form comes from a rhomaleosaurid, the
  Collard plesiosaur, and the joint information from the cryptoclidid
  Muraenosaurus. Neither is this individual, and the paper does not give its
  stroke.
- Frey and others (2017), the soft-tissue-preserved polycotylid
  Mauriciosaurus, Fig. 3 (scielo.org.mx): a distant plesiosaur example only.

The figures were looked at and not imported. A specimen's bone measurements
are not comparable to the bounding box of a chosen outer skin. The skin's
head-region box is therefore compared with the skull figures only as a
sanity check, not as a fit.

## What ships

- `PliosaurStudy(phase, at)` in `manufacture/pliosaur_study.go`: the first
  baseline. A closed body loft with four detached flipper pieces, 24 simple
  cone teeth, eyes and a jaw line in neutral clay. The cornice gargoyles use
  it at phase zero as the creature `pliosaur`, a long-jawed head reaching out of
  the wall with the tip of the jaw as the spout
  (`manufacture/cities/gargoyle-creatures.yaml`).
- `PliosaurContinuousSkin`: the last of four further constructions, formerly
  named `PliosaurFresh5`. Its family builder lives in
  `manufacture/animals/vertebrates/reptiles/pliosaurs/continuous.go`. The
  reptile profile book selects it with the `continuous` study. Earlier
  studies and their tests remain in Git history.
- `sets/gargoyle-cornice-rain.yaml` uses the detached-fin gargoyle form.

## How the last construction is put together

`PliosaurContinuousSkin` is one skin from an axial and head field with four
asymmetric wing fields and smoothly combined fore and hind controls. A
three-second cycle with a fore and hind offset beats the flippers. The cycle
is illustrative, and no fluid is simulated. Sixteen separately made teeth are
rigid points; their roots are the nearest sampled skin vertices.

Measured on the skin: one face-connected closed component of 23,220 used
vertices and 46,436 faces, no orphans, Euler characteristic 2; length 8.520 m
(x from -4.530 to 3.990), shorter than the planned 9 m; head-region box
2.010 by 0.822 m; volume 9.848 m3 at both tested phases; largest movement 0.844
m; loop closure error 5.3e-16 m.

Per-edge absolute length change: p95 5.72 per
cent, p99 11.45 per cent, maximum 18.30 per cent (44.59 to 52.75 mm). All four
flippers move; fore maxima are 0.843 and 0.844 m and hind 0.278 m. The worst
tooth root is 0.0436 m from its nearest skin vertex, which does not prove
exact attachment.

Tests: `TestPliosaurContinuousSkin`, `TestPliosaurContinuousSkinWorstEdge`,
`TestPliosaurContinuousSkinAttachmentsAndSeam`, `TestPliosaurContinuousSkinBoundsAndFourPaddles`.
A 720 by 480, one-ray side frame with two CPU workers took 1.34 s and 112 MB
at peak. A compression check of an eight-frame preview film gave 47.17 dB
average PSNR, which says nothing about smooth motion.

## Using it

Call the constructors from Go, or use the gargoyle creature. The five named
study scenes use a 40 degree default lens so their defaults stay distinct
from other scene families, and their review jobs set 38 degrees.

## Known faults and unfinished parts

- The head is a narrow uniform tube without the fossil's broad temporal
  region and changing bone contours. The eye is a simple bead, with one
  straight mouth mark, eight even upper teeth a side, no modelled lower jaw,
  and none of the source's more numerous, variable, interlocking teeth.
- The flippers are triangular with jagged edges, less curved than the Collard
  analogue, and not proven for P. kevani. Body, tail, thickness, colour and
  flipper size are inferred.
- Kinematic edge ratios are not tissue strain. Self-intersection, joint limits,
  water flow, a whole-cycle strain audit and a collision audit were not done.
- `PliosaurStudy`, the shipped gargoyle form, has detached fins and is not a
  single connected skin. Its volume was 9.368 m3, and the largest displacement
  between two phases was 0.780 m.
- The studio frames have no water.

## Earlier builds

Five constructions, the first four each replaced by the next:

1. A body loft with four detached flipper lofts: each closed, not one skin,
   a featureless taper for a head. Head-region box 1.997 by 1.051 m.
2. A connected implicit torso with tapered wings: one shell of 8,548 vertices,
   17,092 faces, but triangular fins; absolute edge change p95 6.38 per cent,
   p99 12.79 per cent, maximum 36.02 per cent while volume stayed at 10.069 m3.
3. A finer grid with axial masses and curved flipper sections: 24,718 vertices
   and 49,432 faces. The maximum edge change was 97.92 per cent, where the code
   switched abruptly between fore and hind flipper controls at normalized x of
   -0.27: a discontinuous weighting that total volume (10.016 m3) hid.
4. An explicit indexed body with four sewn boundary branches: 5,618 vertices,
   11,232 faces. It removed the switch, with the 96 body vertices there at zero
   displacement. The roots looked blocky and pinched; p99 rose to 16.14 per cent
   and the maximum was 28.97 per cent.
5. The current build, above. Its former switch region, 1,462 sampled edges,
   changes by at most 0.674 per cent.

What carried forward:

- Measure locally and not only by total volume. Look at the worst edge and
  where it lies, to find weight discontinuities.
- Compare motion at comparable displacement; less deformation is not better
  by itself.
- A closed connected skin can still pinch or intersect. Test orphan vertices,
  Euler characteristic and root attachment separately.
- Show the close head, the side and the top, as well as the whole animal.
- Teeth anchored to a field are not shown collision-free or attached across all
  poses.
