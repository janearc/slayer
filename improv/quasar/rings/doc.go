// Package rings follows light past a black hole that does not spin and
// says which image of its disc each ray brings back: the direct image,
// the one from light that went round once more before leaving, and so
// on. The thin bright ring at the shadow's edge is these later images.
//
// It is the arithmetic for the photon ring short. The march is
// celestial's hole, stepped as trace/field.go steps it: an eye held
// still, a ray set off with Arriving and carried by Step at PaceIn's
// size, black when it hovers or reaches the middle.
//
// Two differences from the tracer, both on purpose. The tracer lets an
// outgoing ray go straight past twelve masses; here the march goes on
// to a far radius, so a crossing of the disc's plane is placed by the
// geodesic. And the disc here is a plane with no thickness, where the
// tracer's has a thickness, so these bands are the thinnest a picture
// could show.
//
// A Lens is the hole and an eye at a distance, leaning above the disc's
// plane. Follow is one ray from the eye at an angle off the hole's
// centre, in the plane the eye and the disc's axis share: its Path,
// with every point for a diagram and every crossing of the disc's
// plane. Bands is, for each order of image, the angles it covers on
// the eye's sky, and Pixels what that is in a picture of a width.
package rings
