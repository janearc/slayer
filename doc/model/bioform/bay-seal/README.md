# Bay seal

A harbor seal at the bottling pose of the original bay scene: head down,
continuous skin, glass-like eyes, whiskers, and a dark wet spotted coat. It
is a static specimen. It does not swim and it does not read a clock.

## Built from

The harbor-seal profile’s `bay` study preserves the pose and the materials
of two named scenes at base `935105fd`: `scene/named/seal_bay.go` and
`scene/named/seal_rig_study.go`. The pose and materials are copied from
them. The geometry is built again, with the same `SealRig`. No external
asset is imported, and the named scenes are unchanged.

Appearance context: [NOAA Fisheries, harbor seal profile][noaa]. It is
context for how the animal looks and nothing was measured from it.

[noaa]: https://www.fisheries.noaa.gov/species/harbor-seal

## How it is put together

- skin: one continuous surface from `SealRig` at the fixed pose
- eyes: two corneas, set on the head and following it
- whiskers: on the muzzle
- coat: a dark wet material with a spotted pattern, seeded from the
  harbor-seal coat generator

The numbers are the original scene's. This page does not claim that any of
them is measured from a seal. The result is an illustrative model of a seal,
not a measured individual.

## Using it in a set

`sets/bioforms/bay-seal.yaml` places one seal at the origin with seed 407,
under authored studio light, with two cameras: `side` and `face`. The film
uses a 39 degree lens. Each set supplies its own studio, cameras and light.

The recipe selects `family: pinniped`, the `harbor-seal` species, and
`study: bay` from the checked profile book. The study takes a seed and places
the animal where the set puts it. Its drawn rig and pose are fixed by the
study, with look settings from the profile; it does not use the profile's
loft skeleton counts. The earlier
`bay-nose-map` and `bay-face-map` comparisons are documented in the
[material study](../../../../study/harbor-seal-four-pass/BAY-MATERIAL-COMPARISON.md).

For a set alongside `sets/bioforms/bay-seal.yaml`, the recipe is:

```yaml
recipes:
  profile-harbor-bay:
    animal:
      family: pinniped
      file: ../../manufacture/animals/vertebrates/mammals/pinnipeds/profiles.yaml
      species: harbor-seal
      study: bay
```

The file path is relative to the set. A study cannot be combined with a pose
or representation; see [animal profile recipes](../../../animal-profile-recipes.md).

To draw it from the repository root:

```sh
bin/slayer draw sets/bioforms/bay-seal.yaml --camera side --out bay-seal.png
```

## Known faults and unfinished parts

- The head points downward and the eyes are not readable in the quarter
  view, so the set does not validate a face-to-face bottling composition.
- Neutral studio lighting does not show how dark the coat looks underwater.
- The eyes sit proud of the skin. A request to set them closer to flush was
  not done; this study keeps the earlier eyes.
- The seal is static. A swim needs the deforming-set support described in
  the [bioform index](../README.md).
