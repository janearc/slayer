//go:build darwin

package metal

import _ "embed"

// GeodesicLibrary is celestial's Kerr-Schild Hamiltonian and its RK4
// step (hole.Step) in the Metal shading language, in float32. Vectors
// hold time first, then x, y and z, as celestial's Vec4 and
// ShaderParams do.
//
// A caller bounds the integration and checks its error against
// hole.Step: one single-precision step does not make an accurate
// picture of light. gpu's gravity tests hold it to celestial.
//
//go:embed geodesic.metal
var GeodesicLibrary string

// GeodesicTimeShare is the time a step of the march spans, as a share
// of the radius it is taken at. It is celestial's "time share" march
// choice (hole/choices.json at v0.0.9), which celestial does not export.
const GeodesicTimeShare = 0.03

// GeodesicHoverRatio is how many times its energy an arriving ray's
// time velocity is when the march gives it up. It is celestial's
// "hover ratio" march choice (hole/choices.json at v0.0.9), which
// celestial does not export.
const GeodesicHoverRatio = 1000
