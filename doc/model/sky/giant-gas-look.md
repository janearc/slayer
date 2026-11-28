# Authored giant gas

Generated giants may optionally say `giant_gas: {depth: 0.08, opacity: 40,
made_up: true}`. Depth is a positive share of the body's radius, at most one;
opacity is a finite, nonnegative dimensionless control up to 100000. Omission
preserves the existing 8% and 40 defaults. This adjusts the existing gas
volume and keeps its band appearance; it does not remove the giant or add a
cloud generator. The draw reports the authored values as made up.

The existing medium uses a smooth density ramp from zero at the outer radius
to `opacity / (depth * radius)` at the inner depth. Rays that meet the inner
boundary are still clipped there under the existing opaque-core assumption,
even at small opacity. Consequently changing this control is not a general
transparent-planet model and cannot alone predict a natural atmosphere.
It is not a separate spectral, Rayleigh/Mie or multiple-scattering model.

The distinction between a density profile and optical effects along a ray is
informed by Bruneton and Neyret, *Precomputed Atmospheric Scattering* (2008),
DOI 10.1111/j.1467-8659.2008.01245.x, on the source shelf as
`bruneton-neyret-2008-atmosphere`. This implementation is not their model;
the profile, parameters and limits here are Slayer's existing artistic choices.
For an image comparison, hold camera, geometry, lights, local wisps, print and
seed fixed, then compare default opacity 40 with an explicitly authored lower
positive value. An orbit view and a local view together reveal whether the
change helps a shot and what it costs the global appearance. Image evidence
does not establish physical validity.
