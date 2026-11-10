# A large segmented telescope

The `elt` kind adds a five-mirror scenery assembly with 798 closed primary
panels, a central opening, an open truss and a slit enclosure. It uses the
ordinary mesh route shared by CPU and GPU rendering.

ESO's [ELT facts](https://elt.eso.org/about/facts/) give the nominal 39 m
primary diameter, 798 segments and five-mirror design. The
[M1 description](https://elt.eso.org/mirror/M1/) explains the six identical
sectors of 133 installed segments.

Its 931 figure includes a spare sector;
the manufactured telescope uses 798, not 931.

The numeric aperture and installed count are pinned in
`research/research-derived-data.yml`. Both source pages are pinned in
`sources.yaml`. The positions are an authored sixfold lattice around a
central opening, not a surveyed map of the actual mirror.

## What is chosen

The 96 m enclosure diameter, 90 m height, observing pose, truss length,
panel gaps and shallow surface figures are scenery choices. So are the
secondary and relay dimensions, folded poses, coatings and supports. Their
values and provenance live in `manufacture/ground-choices.yaml`.

The model names all five reflective roles and provides a closed mesh for
each. It does not trace a calibrated five-mirror optical prescription.
The primary's 39 m extent measures the outermost panel corners in its
plane; it is not an effective collecting aperture.

## Use the recipe

Use `kind: elt` in either scene reader. The four detail levels retain
the same physical envelope and installed segment inventory while changing
the surrounding mesh detail.

```sh
game run catalogue -- render elt --out out/elt
```

For an exposed instrument study, set `cutaway: true` in a catalogue plan.
The catalogue labels that view and hides only the enclosure; a scene's
ordinary manufactured object remains complete.

## Merge checks

Independent geometry checks cover all five named mirrors, closed finite
meshes, the physical envelope, exactly 798 primary components, the nominal
aperture and the central opening. Those checks pass at all four details.

The earlier 216 appearance records are unchanged; eight ELT records are
kept separately with their producer hashes. Scoped checks of both scene
readers and the catalogue's pointing also pass.

The exposed assembly's earlier CPU/Metal discrepancy was resolved by the
[ordinary triangle precision change](triangle-precision.md). The original
cutaway view and RGB acceptance budget were retained. This is a bounded
comparison of that assembly, not a general watertightness guarantee.

The quick-detail exterior and labelled cutaway have inspected native PNGs,
with settings, actual native refinement counts and producer records.
The [OWL draft](owl.md) follows a different open-frame concept and awaits
its own checks; the ELT results do not validate that new geometry.
