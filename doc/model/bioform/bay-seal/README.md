# Bay seal

A harbor seal at the bottling pose of the original bay scene: head down,
continuous skin, glass-like eyes, whiskers, and a dark wet spotted coat. It
is a static specimen. It does not swim and it does not read a clock.

## Built from

The kind `bay-seal` preserves the pose and the materials of two named scenes
at base `935105fd`: `scene/named/seal_bay.go` and
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

The kind takes a seed and places the animal where the set puts it. The
`bay-nose-map` and `bay-face-map` coats stay available as opt-in comparisons
beside the original coat.

To draw it from the repository root:

```sh
bin/slayer draw sets/bioforms/bay-seal.yaml --camera side --out bay-seal.png
```

## Known faults and unfinished parts

- The head points downward and the eyes are not readable in the quarter
  view, so the set does not validate a face-to-face bottling composition.
- Neutral studio lighting does not show how dark the coat looks underwater.
- The eyes sit proud of the skin. A request to set them closer to flush was
  not done; this kind keeps the earlier eyes.
- The seal is static. A swim needs the deforming-set support described in
  the [bioform index](../README.md).
