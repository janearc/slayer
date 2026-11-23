// Package skyglow works out the light a flat layer of air gathers from
// what shines in a city and sends back: the dim, coloured, uneven glow
// over a street, brightest low over the brightest blocks.
//
// Nothing in it is painted. The glow is the city's own light, scattered
// by the air, and every number in it is the air's or an emitter's.
//
// # The model
//
// Light going through air of extinction k and single-scattering albedo
// w loses the share exp(-tau) along a path of optical depth tau, and a
// bit of air scatters w times what reaches it. The mean radiance J at a
// point is the light from every direction, summed over the sphere and
// divided by four pi; a point source of intensity I, a distance r away,
// adds I exp(-tau) / (4 pi r^2) to it.
//
// Take the air in a grid of cells of volume V. A cell scatters, alike in
// every direction, an intensity w k J V: it is a point source of that
// intensity. So the order of scattering after the next is the order
// before it, spread by the same sum, and the glow is the series
//
//	J = J1 + J2 + J3 + ...
//
// J1 is the light that reaches the air straight from the emitters, and
// each next order is the last one scattered once more. Each order is
// about w tau of the one before it, which is why it is worth having:
// over a city a few hundred metres across in a haze of a few kilometres
// visibility, w tau is about a third, so the second order is a third of
// the first and the third a ninth.
//
// Solve stops when an order adds less than the tolerance, and says what
// the orders past it are likely to add, as the share the geometric tail
// would hold.
//
// The camera's march reads J at each step through the air and adds the
// albedo times J as the step's source, which is the isotropic-single
// source of doc/model/sky/volume-environment-lighting.md.
//
// # Error
//
// What the solution leaves out, each stated so it can be judged:
//
//   - Emitters are gathered into bins of a fixed size and each bin shines
//     from its middle. A node nearer than a bin is smeared; the glow is
//     right at a distance of a few bins and more, and a lamp's own halo
//     is the tracer's, not this package's.
//   - The cells are points to their neighbours. The nearest neighbours
//     are the worst, by some tenth of their share; their share of the
//     second order is a few tenths of it.
//   - Scattering past the first is taken as isotropic. Real haze throws
//     light forward, which keeps it nearer where it began; the second
//     order is spread wider here than it would be, by a factor not worse
//     than the order itself is small, so about a third of a third.
//   - Nothing hides an emitter or the air from one: no wall, no ground.
//     Light going down is lost where in life the ground would give some
//     back. A panel's back face, against a wall, shines into the wall
//     here, which in life it does not.
//   - The air is grey: the glow is worked out at the reference
//     wavelength. The tint a haze has across the eye's three channels,
//     medium.Aerosol's, reaches the picture twice, in what each channel
//     of direct light loses and in how much each channel of the glow is
//     scattered toward the eye; it does not reach the glow's own order
//     of scattering, which stays grey.
//   - Nothing shining gives no glow: a dark field, which is what air with
//     no light in it is.
//
// The GPU takes the field's cells by Cells and Cell and reads them with
// the same trilinear weights as Mean; see doc/model/sky/skyglow.md.
package skyglow
