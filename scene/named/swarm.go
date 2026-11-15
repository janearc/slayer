package named

import (
	"io"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// the swarm in units of the star's radius: a star of one, in a haze to
// fourteen. Picked when cmd/swarm was written, and moved here unchanged,
// as is every number in this file.
const (
	swarmStar = 1.0
	swarmHaze = 14.0
)

// the swarm's star: its surface in kelvin, its brightness in the green at
// its own radius, and how long a ring takes to go round
const (
	swarmKelvin = 4800
	swarmGreen  = 38
	swarmTurn   = time.Minute
)

// the swarm's sky: its darkness and the seed of its stars
const (
	swarmSkyRed, swarmSkyGreen, swarmSkyBlue = 0.002, 0.002, 0.004
	swarmSkySeed                             = 7
)

// the swarm's haze: how thick, the seed, scale and share of its wisps,
// and its colour
const (
	hazeThick                    = 0.05
	hazeSeed                     = 3
	hazeScale                    = 2.5
	hazeAmount                   = 0.45
	hazeRed, hazeGreen, hazeBlue = 0.9, 0.82, 0.72
)

// the swarm as cmd/swarm drew it: the way to the eye, a little above the
// plane, the lens's width in degrees, the picture's size and rays a side,
// and the tone's knee and top
const (
	swarmAboveY, swarmAboveZ          = 4, 26
	swarmWideDegrees                  = 45
	swarmWidth, swarmHeight, swarmRay = 1600, 900, 2
	swarmKnee, swarmTop               = 0.7, 8
)

// the swarm's parameters' defaults
const (
	swarmOrbits    = 60
	swarmPanels    = 120
	swarmBandWidth = 0.4
	swarmDistance  = 26
)

// swarmScene is the lantern: a star inside a dyson swarm, in a thin haze.
func swarmScene() Scene {
	return Scene{name: "swarm",
		about: "the lantern: a star inside a dyson swarm of mirrored " +
			"panels, its light coming out between them in shafts " +
			"through a thin haze",
		unit:       "the star's radius",
		renderers:  []Renderer{CPU, GPU},
		parameters: swarmParameters(),
		defaults: Defaults{width: swarmWidth, height: swarmHeight,
			rays: swarmRay, wide: swarmWideDegrees * maths.Degree,
			tone: colour.Tone{Knee: swarmKnee, Top: swarmTop}},
		build: buildSwarm}
}

// swarmParameters are what the swarm takes, as cmd/swarm's flags were.
func swarmParameters() []Parameter {
	return []Parameter{
		whole("orbits", swarmOrbits, unbounded,
			"how many tilted orbits the panels are on"),
		whole("panels", swarmPanels, unbounded,
			"how many panels to an orbit"),
		number("band-width", swarmBandWidth, unbounded,
			"how wide each band of panels is, in radii of the star"),
		number("wear", nothing, unbounded,
			"how battered the panels are, nought to one"),
		number("glass", nothing, unbounded, "panels of black glass outside "+
			"at this index, matte black inside; nought is warm metal"),
		number("distance", swarmDistance, unbounded,
			"how far out the eye is, in radii of the star"),
		number("spread", nothing, unbounded,
			"radii of the star between one ring and the next out"),
		number("narrowest", nothing, unbounded,
			"the outermost ring's width, narrowing from band-width"),
		span("fastest", nothing,
			"the outermost ring's turn, quickening from a minute"),
	}
}

// buildSwarm is the world: the lantern, a star of radius one in its
// swarm, and the haze its light shows in; and the eye a little above the
// swarm's plane.
func buildSwarm(held values, _ io.Writer) (*Built, error) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(swarmSkyRed, swarmSkyGreen,
		swarmSkyBlue), Seed: swarmSkySeed}
	glass := held.number("glass")
	_, err := lantern.Lantern{Radius: swarmStar,
		Orbits: held.whole("orbits"), Panels: held.whole("panels"),
		Kelvin: swarmKelvin, Green: swarmGreen,
		Wide: held.number("band-width"), Wear: held.number("wear"),
		Turn: swarmTurn, Skin: lantern.BlackGlass(glass),
		Lining: lantern.Matte(glass),
		Shells: lantern.Shells{Spread: held.number("spread"),
			Fastest:   held.span("fastest"),
			Narrowest: held.number("narrowest")},
	}.Place(world)
	if err != nil {
		return nil, err
	}
	world.AddVolume(hazeAround())
	from := maths.V(0, swarmAboveY, swarmAboveZ).Unit().
		Scale(held.number("distance"))
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	return &Built{world: world, eye: eye}, nil
}

// hazeAround is the haze the shafts show in: thin, ragged, scattering
// the lantern's light where no panel shades it
func hazeAround() thing.Volume {
	wisps := medium.Wispy{Of: medium.Puff{Radius: swarmHaze,
		Thick: hazeThick}, Seed: hazeSeed, Scale: hazeScale,
		Amount: hazeAmount}
	dust := &thing.Material{Name: "haze",
		Colour: maths.RGB(hazeRed, hazeGreen, hazeBlue)}
	return thing.Volume{Medium: wisps, Material: dust}
}
