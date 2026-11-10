# More than a row of differently sized domes

The next observatory recipes need different structures. The current dome
and alt-azimuth assembly is a useful starting point; scaling it cannot
produce every instrument that was asked for.

The [primary-source shortlist](../research/observatory-family-references.yaml)
records the references read for this work. Palomar, Keck, Subaru, Rubin, ELT
and Griffith source bytes are fetched and pinned.

Fourteen measured claims are verified against those bytes before a build.
Other families remain forthcoming.

The named templates are `palomar`, `keck-i`, `keck-ii`, `subaru`,
`rubin`, `elt` and `griffith`. They preserve `Optical` and `Observatory`
for existing callers.
Keck I and II share one geometry template for distinct placement at a site.

## Ground optical families

- Palomar: a long optical structure on an equatorial mounting, inside a large
  dome. Keep the source's nominal 200-inch primary in its original unit;
  a conversion must not imply extra measurement precision.
- Keck I and II: two instances of a segmented-primary family. The reference
  gives 36 segments per 10 m primary. A reusable pair belongs in a site
  composition, with distinct object names for cameras and guide-star launchers.
- Subaru: a cylindrical rotating enclosure and alt-azimuth telescope. Its
  published parameter page supports a different enclosure silhouette.
- Rubin: a compact survey telescope with its combined primary/tertiary blank.
  The physical 8.4 m mirror and mean effective 6.49 m aperture are different
  quantities; a visible mirror diameter cannot stand in for collecting area.
- ELT: a large hemispherical enclosure, open structure and dense segmented
  primary. Preserve the 798-panel reference while treating gaps, thickness
  and simplified supporting members as authored geometry.
- Griffith: a civic building with wings and domes, and a refractor in its
  east roof dome. Model this as a composition, not a giant version of the
  existing reflecting telescope.
- OWL: keep the historical concept. Label the selected design version;
  ESO's design page and overview use 3048 primary segments, while an indexed
  FAQ gives 3042. Do not silently resolve that difference as one timeless count.

New constructors should preserve `Optical` and `Observatory` for existing
callers. Published defaults and authored proportions belong in separate data.
All details need a stable physical envelope for clearances and camera paths.

## Using the named templates

Use any of their names wherever a scene accepts a manufactured kind. A named
recipe can be as small as `kind: palomar`; place it with the ordinary scene
thing or stand controls.

The reflecting templates and Griffith's refractor have a fixed 55-degree
optical elevation and an observing slit toward negative Z before scene
placement.

Site rotation, guide-star launchers and weather belong to the scene.

To inspect the templates without downloading terrain:

```sh
game run catalogue -- render palomar keck-i keck-ii subaru rubin elt griffith \
  --out out/ground-book
```

The default catalogue camera looks into their slits. A labelled cutaway
plan can hide the enclosure for inspection. All four complexity levels keep
the same physical envelope and primary dimensions.

Caltech gives the Hale's nominal [200-inch mirror and rounded 42 m by 41 m
enclosure](https://sites.astro.caltech.edu/palomar/about/telescopes/hale.html).
Keck gives [10 m primaries with 36 hexagonal
segments](https://keckobservatory.org/our-story/telescopes/).

NAOJ gives
Subaru's [8.2 m effective aperture and 40 m by 43 m
enclosure](https://subarutelescope.org/Telescope/Parameters/).

Runtime source entries are in `research/research-derived-data.yml`. Support
sizes, gaps, member counts, mirror figure, coatings and the Keck and ELT
enclosures are picked scenery choices in `manufacture/ground-choices.yaml`.

The Hale's horseshoe suggests its equatorial mounting without a complete
mechanical model.

These are visual recipes, with no calibrated optical simulation.

ELT's [construction note](large-observatory.md) records its 798-panel
inventory and authored sixfold layout. Native quick-detail exterior and
cutaway studies exist; its software-Metal cutaway comparison is still a
merge blocker.

Griffith's [construction note](civic-observatory.md) describes the visitor-map
layout and the guide's 12-inch objective. Its independent roof and objective
checks and one small CPU/software/native-Metal exterior comparison pass.
The west solar instrument is represented by a closed roof.

## Dishes, arrays and radomes

An ALMA-like dish and an SKA-Mid-like dish should be different assemblies.
The SKA source describes an offset sub-reflector and receiver; putting a
central feed on every paraboloid loses that distinction. Reflector curvature
and supporting geometry can remain visual approximations with clear labels.

Array layout belongs with placement. Reuse a small number of dish specimens,
then place their instances on terrain with explicit spacing and pointing.
Compact and spread-out layouts are useful scenery recipes; neither claims
to reproduce surveyed antenna pads or simulate an interferometer.

Radomes need their own shell and support recipes. Their visible panels are
scenery, with no claim about radio transmission through the enclosure.
Keep their meshes closed and finite for the existing CPU and GPU mesh paths.

## Proof beside the recipe

Each family needs independent silhouette and component probes, finite bounds,
actual native GPU images and a catalogue entry. Record sources beside defaults
and inspect the PNGs. An authored compatibility signature can hold appearance
steady, but cannot decide whether a structure is recognisable or correct.
