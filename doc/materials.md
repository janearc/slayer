# Materials and particle-bearing gases

materials carry composition and, where available, bulk density,
optical readings and attenuation coefficients. a surface finish
uses those readings to make the rendered surface. for mass and volume
accounting of solids, see [porous specimens](model/material/specimens.md).

## What a material carries

- A material has a name and a formula. It may also have a bulk density,
  with the table cell it was read from (`AtDensity`, `DensityFrom`).
- It may have optics: a complex refractive index, n and k, for each of the
  tracer’s red, green and blue bands. entries include their source,
  such as the row in refractiveindex.info. examples include gold,
  silver, aluminium and titanium.
- It may have x-ray attenuation: a mass attenuation coefficient at an
  energy, each with its row in NIST's tables (SRD 126). Lead, aluminium and
  liquid water have a row at 60 keV. `Passes` is the share of a beam that
  comes through a thickness by the exponential law. it requires
  a coefficient at that energy and a bulk density.
- `materials.ByName` looks up a material from the catalogues. use
  the returned source metadata to distinguish measured data from
  derived values.
- `thing.Made` combines a material with a finish. paint supplies
  colour directly. metal finishes need n and k for all three RGB bands.
  `thing.Stopping` uses the attenuation data for bands outside the
  visible. bands without data are stopped.

[Nickel-aluminium bronze](model/material/nickel-aluminium-bronze.md) adds a selected C95800
composition and cited density alongside titanium. Its optical readings are
explicit estimates derived from a different NAB specimen's reflectance;
`Source.Estimated()` keeps that distinction available to callers. The generic
`ConductorFromReflectance` constructor documents the chosen effective fit.

the tracer uses the stored RGB readings. spectral interpolation
between source rows is outside this path.

## Particle-bearing gas design

validated constructors already describe elemental proportions, material
fractions and particles. the proposed model puts particles in a gas by
mass share and size, and derives wavelength-dependent scattering
from their composition. a varying mixture would validate the fractions
returned at each position. the tracer would consume its usual density,
scattering, absorption and emission properties.

## Proposed set-file syntax

this design sketch uses proposed keys. current set files use their
individual gas and density models.

```yaml
nebula:
  gas: {hydrogen: 0.7, helium: 0.28, neon: 0.02}
  thick: 1.8e-3
  particles: {by_mass: 0.01, size_um: 0.1}
  made_of: {ice: 0.6, tholin: 0.3, iron: 0.1}
```
