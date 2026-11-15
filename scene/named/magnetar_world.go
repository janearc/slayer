package named

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/thing"
)

// A world beside the real magnetar: the Earth, as its template has it,
// without its Moon or its cities, at a distance from the star inside the
// light cylinder, where the dipole law holds; its aurora pushed as smoke
// 33 pushes it.
//
// How far the aurora reaches and how bright it is are
// smoke 33's picks, an illustration of the effect and not its measured
// extent: nobody has seen a planet beside a magnetar. What is computed
// is the star's field at the planet against the Earth's own, and what
// the 2004 flare would bring there.

// how far from the star the world is, in km, by default: well inside
// the light cylinder and far enough that the star's field does not
// reach the strongest lab magnet's there. Picked by the author of this
// scene, 2026-11-12.
const worldAt = 1e5

// the pushed aurora, as smoke 33 draws it beside a magnetar: its shell
// in km above the ground, the oval's latitude, width and tilt in
// degrees, its strength and curtains as shares, its seed, picked for
// smoke 33 and taken unchanged; and the detail its ground is tiled at,
// raised from smoke 33's 8 by the author of this scene, 2026-11-12, so
// its edge is smooth across a full frame
const (
	auroraLow, auroraHigh              = 300, 25000
	auroraLat, auroraWidth, auroraTilt = 20, 30, 30
	auroraStrength, auroraCurtains     = 0.8, 1
	auroraSeed                         = 33
	fringeRed, fringeGreen, fringeBlue = 1, 0.3, 0.9
	bottomRed, bottomGreen, bottomBlue = 0.3, 1, 0.4
	topRed, topGreen, topBlue          = 1, 0.35, 0.15
	worldDetail                        = 32
)

// the eye on the world: from beyond the world, away from the star, in
// the world's radii, outside the pushed aurora, so the world and its
// aurora fill one side and the star is at the middle. Picked by the author of this scene, 2026-11-12.
const (
	worldEyeOut, worldEyeUp, worldEyeSide = 12, 1, 4
)

// placeWorld puts the world at its distance along x, refusing one at or
// past the light cylinder, where the field law the scene draws by does
// not hold.
func placeWorld(world *thing.World, body star, out float64) error {
	if !(out > body.radius) || out >= body.lightCylinder() {
		return fmt.Errorf("%w: world-at %g km; a world must stand "+
			"outside the star and inside the light cylinder, %g km, "+
			"where the dipole holds", ErrBounds, out,
			body.lightCylinder())
	}
	earth, err := pushedEarth()
	if err != nil {
		return err
	}
	return earth.Place(world, maths.V(out, 0, 0), single, worldDetail)
}

// pushedEarth is the Earth with smoke 33's aurora in place of its own.
func pushedEarth() (bodies.Body, error) {
	earth, err := template.Earth()
	if err != nil {
		return bodies.Body{}, err
	}
	storm, err := bodies.NewAurora(bodies.NewShell(
		bodies.Kilometres(auroraLow), bodies.Kilometres(auroraHigh)),
		measure.Degrees(auroraLat), measure.Degrees(auroraWidth),
		measure.Degrees(auroraTilt), maths.Share(auroraStrength),
		maths.Share(auroraCurtains), auroraSeed,
		bodies.NewAuroraColours(maths.RGB(fringeRed, fringeGreen,
			fringeBlue), maths.RGB(bottomRed, bottomGreen, bottomBlue),
			maths.RGB(topRed, topGreen, topBlue)))
	if err != nil {
		return bodies.Body{}, err
	}
	return earth.WithoutMoons().WithoutCities().WithoutAurora().
		With(storm)
}

// worldEye looks from beyond the world back toward the star.
func worldEye(out float64) maths.Subspace {
	radius := earthKm() / diameter
	from := maths.V(out+worldEyeOut*radius, worldEyeUp*radius,
		worldEyeSide*radius)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 0, 0), maths.Up)}
}
