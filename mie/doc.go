// Package mie is the scattering of light by a sphere, worked out from
// Maxwell's equations as Mie did in 1908, and the averages of that
// scattering over a population of spheres of different sizes.
//
// For one sphere of one size and one refractive index, the package
// finds the efficiencies of extinction and scattering and the asymmetry
// parameter: the share of the light the sphere takes out of the beam,
// the share it scatters rather than absorbs, and how far forward it
// scatters.
//
// The method is Bohren and Huffman's (Absorption and Scattering of
// Light by Small Particles, 1983, appendix A): the logarithmic
// derivative by downward recurrence, the Riccati-Bessel functions by
// upward recurrence, and a series stopped by Wiscombe's rule.
//
// Population is a log-normal distribution of radii, the shape cloud
// droplets are measured in. Its Average turns one into the numbers a
// radiative transfer solver wants for a cloud: the cross section of the
// average particle, the albedo of a single scattering and the
// asymmetry, all at one wavelength.
//
// The sphere is homogeneous and the medium round it is nothing that
// absorbs: the index is the sphere's relative to the medium.
// Polarisation is not here, and neither is a particle that is not a
// sphere.
package mie
