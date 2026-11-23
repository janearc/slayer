// Package airlight turns a planet's air into the light the renderer
// takes: the sky as a scene/sky Sky, the direct sun as a distant
// lamp's light, and the exposure a camera's incident meter would pick.
//
// It works the air's radiance field with package skylight at each of a
// span of wavelengths, from the set's ground, and reads it back through
// the CIE 1931 observer by moonlight's route from a spectrum to the
// tracer's three channels, moonlight.Spectrum's Picture.
//
// Nothing in it is painted: the sky's colour and brightness, the sun's
// strength and the exposure are all the air's.
// doc/model/sky/sky-from-the-air.md is the design; this is the fourth step
// of its order.
//
// An air is named by a set file: Earth's or Venus's. Named gives the air
// of a name, lit by the sun's spectrum the caller read from the shelf,
// with its photons and its bins; NewPlan works its columns from the
// set's ground; Plan.Solve walks them into Fields, which a cache keeps
// under Plan.Keyed; NewSky, Sun and Meter read the fields.
package airlight
