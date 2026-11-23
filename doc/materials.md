# Materials and particle-bearing gases

For the implemented mass and volume accounting of porous solids, read
[Porous material specimens](model/material/specimens.md). It keeps current solid phases,
pore water and empty pores separate, without assigning appearance.

This design describes composition as data: hydrogen
with ice and tholin grains, an iron hull, or an invented material. Optical
behaviour should follow the declared composition. The implementation may
cover only part of this plan; the package APIs define current support.

## Motivation

A hand-picked gas colour or surface gloss cannot explain why dust reddens
a nebula, ice brightens it, or ionized gas emits particular lines.
Composition supplies those reasons, just as temperature supplies a star's
spectrum instead of an arbitrary RGB colour.

## Layers of the model

- **Element:** an atomic symbol such as H, He, Fe, or U, with mass and
  emission lines represented using the `lines` package.
- **Formula:** elemental proportions. Water is H2O; a tholin is a mixture
  involving carbon, hydrogen, and nitrogen. One element is a valid formula.
- **Material:** a formula plus bulk density and a complex refractive index
  across bands, with published tables cited. Example densities are about
  0.92 g/cm^3 for ice and 7.87 g/cm^3 for iron.
- **Fractions:** material shares summing to one. Construction rejects
  invalid totals rather than requiring downstream rechecks.
- **Particle:** size and composition. Mie calculations derive band albedo
  and scattering asymmetry once for reuse.
- **Gas:** the existing gas model augmented with elemental fractions.
- **Compound gas:** gas carrying particles, described by their mass share,
  size, and composition. It implements the same medium contract as plain gas.

## Construction and interfaces

Private values are created through validating constructors and inspected
through methods. Errors explain invalid inputs.

The tracer needs density at a point, band-dependent scattering and
absorption, scattering directionality, and emission. Plain and compound
gases should supply that same interface so composition does not introduce
a special tracer path.

## What a material carries

This part is not design. It says what the `materials` package holds today
for solid things, so a reader need not infer it from the plan above.

- A material has a name and a formula. It may also have a bulk density,
  with the table cell it was read from (`AtDensity`, `DensityFrom`).
- It may have optics: a complex refractive index, n and k, for each of the
  tracer's red, green and blue, each reading with the row it was read from
  in the refractiveindex.info database. Gold, silver, aluminium, beryllium,
  titanium, steel and ochre have them.
- It may have x-ray attenuation: a mass attenuation coefficient at an
  energy, each with its row in NIST's tables (SRD 126). Lead, aluminium and
  liquid water have a row at 60 keV. `Passes` is the share of a beam that
  comes through a thickness, by the exponential law, and refuses a material
  with no row at that energy or no density.
- `ByName` gives a material by its name. A row is quoted in the code and its
  page is not: NIST's pages are copyright and stay with celestial-data's
  adapter, which holds each to its checksum.
- A look is made of a material and a finish by `thing.Made`: a metal mirrors
  by its own n and k, and the finish says how polished it is.
  `thing.Stopping` is what a material takes from a band beyond the visible,
  from its attenuation; a band with no row is stopped whole.

[Nickel-aluminium bronze](model/material/nickel-aluminium-bronze.md) adds a selected C95800
composition and cited density alongside titanium. Its optical readings are
explicit estimates derived from a different NAB specimen's reflectance;
`Source.Estimated()` keeps that distinction available to callers. The generic
`ConductorFromReflectance` constructor documents the chosen effective fit.

Nothing reads a material's formula back yet, so a wrong one could not be
seen; that is a known gap. No material's optics are interpolated between
rows: the readings are the grid points nearest the tracer's three colours.

## Spatially varying mixtures

A cloud may contain more ice far from a star and more iron nearby. A
composition callback can return validated fractions for a position instead
of one fixed mixture.

The callback must construct those fractions through the same validation
boundary as constant input. The compound-gas constructor can probe a few
positions to catch obvious failures early; such probes do not establish
that every possible position is valid.

## Proposed set-file syntax

This is a design example, not a claim of current set-reader support:

```yaml
nebula:
  gas: {hydrogen: 0.7, helium: 0.28, neon: 0.02}
  thick: 1.8e-3
  particles: {by_mass: 0.01, size_um: 0.1}
  made_of: {ice: 0.6, tholin: 0.3, iron: 0.1}
```
