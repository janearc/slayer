package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// demoDetails adds small photo-derived fittings at the highest catalogue
// detail. The feature names are documented in American Submariner (2023,
// pp. 22–23); their dimensions and precise seats are authored approximations
// because no dimensioned Project 705 drawings were found.
func (size alfaSizes) demoDetails(assembly *instrumentAssembly) {
	top := size.Beam + size.Sail.Sections[len(size.Sail.Sections)-1].Up
	aft := size.Sail.Front + size.Sail.Sections[len(size.Sail.Sections)-1].Lead +
		size.Sail.Sections[len(size.Sail.Sections)-1].Chord*spaceHalf
	// Periscope, radar and radio antenna forms are separate, simple rods.
	// Heights and lateral positions are chosen from surfaced photographs.
	masts := []struct {
		x, aft, height, radius float64
	}{{-.55, aft - .8, 2.8, .09}, {-.18, aft + .1, 3.8, .075},
		{.2, aft - .3, 3.2, .065}, {.52, aft + .5, 2.3, .06}}
	for _, mast := range masts {
		base := size.at(mast.x, top, mast.aft)
		assembly.rod(base, base.Add(maths.V(0, mast.height, 0)), mast.radius, alfaMast)
	}
	// Two low covers make the sail crown legible at close range. These are
	// surface plates, not openings cut through the sail.
	for _, offset := range []float64{-.95, .75} {
		centre := size.at(0, top+.035, aft+offset)
		assembly.box(.72, .07, 1.25,
			maths.V(centre.X-.36, centre.Y, centre.Z-.625), alfaFitting)
	}
	// The aft starboard condenser cooling scoop is feature-backed; this
	// shallow grille-like plate is only an authored exterior indication.
	scoopAft, scoopUp := size.Length*.21, 3.35
	hullRadius := size.hullRadius(scoopAft)
	fromAxis := scoopUp - size.radius()
	scoopSide := math.Sqrt((hullRadius - fromAxis) * (hullRadius + fromAxis))
	assembly.box(.12, .62, 1.35, maths.V(scoopSide-.05,
		scoopUp, scoopAft-size.Length*spaceHalf), alfaFitting)
}
