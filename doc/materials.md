# Materials and particle-bearing gases

This design, recorded on 2026-10-04, describes composition as data: hydrogen
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

## Implementation sequence

Start with data types and constructors. Test unknown symbols, negative
shares, totals such as 0.999, zero particle size, and missing optical data.
Then compare the Mie calculation with published ice and silicate albedos.
Finally connect compound gas through the same tracer interface as plain gas.

## Unresolved choices

- Select and cite optical-index tables for ice, silicate, iron, graphite,
  and tholin.
- Begin with a single grain size; add size distributions when a scene
  requires them.
