package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Mariner 10 represents the deployed Venus/Mercury spacecraft. The 1.39 m
// bus, 0.457 m bus depth, 1.37 m reflector, and 6.8 m deployed span come from
// NASA's spacecraft directory and Historical Data Book. Locations of small
// fittings and their surface finishes follow the flight diagram by eye.
//
// https://ntrs.nasa.gov/citations/19900020099
// https://ntrs.nasa.gov/citations/19880016046
// https://science.nasa.gov/resource/mariner-10-diagram/
type mariner10 struct{}

// madeAt is made at any detail: the craft has the one representation.
func (m mariner10) madeAt(from seed, detail string) (good, error) {
	return m.made(from)
}

// made assembles the deployed craft from its parts and materials. It
// refuses a seed that was not made.
func (mariner10) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	a := &instrumentAssembly{fine: cut{around: 8}, object: good{dressed: looks{
		body:           thing.Polished("gold blanket", maths.RGB(.51, .29, .065), .27, 1.7),
		leaf:           thing.Polished("blue-black solar cell glass", maths.RGB(.023, .044, .094), .38, 1.75),
		spaceStructure: thing.Polished("aluminium frame", maths.Grey(.55), .3, 1.7),
		spaceFoil:      thing.Diffuse("black equipment housing", maths.Grey(.045)),
		spaceMirror:    thing.Diffuse("silvered reflector face", maths.Grey(.58)),
	}}}
	a.rod(maths.V(0, 0, 0), maths.V(0, .457, 0), .695, body)
	a.rod(maths.V(0, .455, 0), maths.V(0, .481, 0), .703, spaceStructure)
	a.rod(maths.V(0, -.022, 0), maths.V(0, .008, 0), .702, spaceStructure)
	a.fine.around = 24
	marinerWings(a)
	marinerEquipment(a)
	if err := a.shareSolar(leaf); err != nil {
		return good{}, err
	}
	bounds := maths.SegmentBox(maths.V(-3.4, -.65, -3.5), maths.V(3.4, 3.5, 2.3), 0)
	g, err := a.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *g, nil
}

// The wings of Mariner 10, in metres, as the flight diagram shows them
// by eye: where each wing's outer edge is, the panel's size and its
// underside, the yoke's arms, and the cells' size and pitch. The cells
// are 31 along the wing and 9 across it.
const (
	marinerWingEdge     = 3.4
	marinerPanelLength  = 2.6924
	marinerPanelWidth   = .97536
	marinerPanelThick   = .045
	marinerPanelFloor   = .26
	marinerYokeRoot     = .48
	marinerYokeLength   = .36
	marinerYokeRadius   = .026
	marinerYokeAcross   = .34
	marinerYokeRootRise = .19
	marinerYokeTipRise  = .28
	marinerCellColumns  = 31
	marinerCellRows     = 9
	marinerCellWidth    = .081
	marinerCellDepth    = .095
	marinerCellHeight   = .006
	marinerCellPitchX   = .085
	marinerCellPitchZ   = .102
	marinerBarShare     = .5
)

// marinerWings puts an array on each side of the bus, and the small
// fittings at the wing tips.
func marinerWings(a *instrumentAssembly) {
	for _, side := range []float64{-1, 1} {
		marinerWing(side).pulledIn().lay(a)
		x := side*3.30 - .035
		a.box(.07, .065, .11, maths.V(x, .309, -.38), spaceFoil)
		a.box(.05, .012, .07, maths.V(x+.01, .375, -.36), spaceMirror)
	}
}

// marinerWing is one wing as an array: a single panel out to the wing's
// edge on two yoke arms to the bus, the cells facing up. The yoke's
// arms slope up from the bus to the panel's mid-plane.
func marinerWing(side float64) solarArray {
	plane := marinerPanelFloor + marinerPanelThick*spaceHalf
	panels := solarPanels{along: spaceOne, across: spaceOne,
		width: marinerPanelLength, depth: marinerPanelWidth,
		thickness: marinerPanelThick,
		reach:     marinerWingEdge - marinerPanelLength - marinerYokeRoot}
	return solarArray{
		frame: solarFrame(maths.V(side*marinerYokeRoot, plane, spaceZero),
			maths.V(side, spaceZero, spaceZero),
			maths.V(spaceZero, spaceOne, spaceZero)),
		panels: panels,
		cells: solarCells{columns: marinerCellColumns,
			rows: marinerCellRows, width: marinerCellWidth,
			depth: marinerCellDepth, height: marinerCellHeight,
			pitchX: marinerCellPitchX, pitchZ: marinerCellPitchZ,
			barShare: marinerBarShare},
		yoke: solarYoke{length: marinerYokeLength,
			radius:     marinerYokeRadius,
			rootOffset: plane - marinerYokeRootRise,
			tipOffset:  plane - marinerYokeTipRise,
			across:     []float64{-marinerYokeAcross, marinerYokeAcross}},
		looks: solarLooks{cell: leaf, bar: spaceMirror,
			substrate: spaceStructure, frame: spaceFoil,
			hardware: spaceStructure},
	}
}

// marinerEquipment adds the off-axis reflector with its feed and tripod,
// the low-gain mast, the magnetometer boom and the imaging telescopes.
func marinerEquipment(a *instrumentAssembly) {
	// Off-axis reflector, feed and feed tripod.
	pose := maths.NewSubspace(maths.V(.34, .77, 1.08), maths.About(maths.XAxis, .48), 1)
	a.fine.around = 64
	a.add(marinerReflector(), pose, spaceMirror)
	a.fine.around = 24
	a.rod(maths.V(.35, .21, .50), maths.V(.34, .77, 1.08), .043, spaceStructure)
	a.rod(maths.V(.34, .78, 1.08), maths.V(.34, 1.21, 1.34), .024, spaceFoil)
	for arm := range 3 {
		angle := float64(arm) * 2 * math.Pi / 3
		anchor := maths.V(.34+.54*math.Cos(angle), .88, 1.08+.48*math.Sin(angle))
		a.rod(anchor, maths.V(.34, 1.21, 1.34), .009, spaceStructure)
	}

	// Low-gain mast, magnetometer boom, and the two imaging telescopes.
	a.rod(maths.V(0, .45, -.15), maths.V(0, 2.94, -.15), .028, spaceStructure)
	a.rod(maths.V(0, 2.94, -.15), maths.V(0, 3.30, -.15), .013, spaceStructure)
	a.rod(maths.V(0, 3.28, -.15), maths.V(0, 3.30, -.15), .085, spaceMirror)
	a.rod(maths.V(0, 3.32, -.15), maths.V(0, 3.34, -.15), .071, spaceMirror)
	boomRoot := maths.V(.28, .27, -.49)
	boomTip := maths.V(1.38, .57, -3.22)
	a.rod(boomRoot, boomTip, .021, spaceStructure)
	for _, fraction := range []float64{.39, .96} {
		at := boomRoot.Add(boomTip.Sub(boomRoot).Scale(fraction))
		a.box(.15, .09, .10, at.Add(maths.V(-.075, .012, -.05)), spaceFoil)
		a.box(.17, .02, .11, at.Add(maths.V(-.085, -.004, -.055)), spaceStructure)
	}
	for _, x := range []float64{-.40, -.15} {
		a.box(.18, .19, .21, maths.V(x-.09, .24, -.66), spaceStructure)
		a.rod(maths.V(x, .33, -.64), maths.V(x, .33, -.96), .077, spaceFoil)
		a.rod(maths.V(x, .33, -.959), maths.V(x, .33, -.985), .081, spaceStructure)
		a.rod(maths.V(x, .33, -.984), maths.V(x, .33, -.989), .062, spaceFoil)
	}
	// Discrete instrument mounts following the NASA flight diagram.
	a.box(.89, .035, 1.29, maths.V(-.445, -.059, -.645), spaceMirror)
	a.box(.25, .21, .16, maths.V(.23, .21, -.64), spaceStructure)
	a.rod(maths.V(.32, .33, -.75), maths.V(.32, .33, -.93), .052, spaceFoil)
	a.rod(maths.V(.51, .27, .13), maths.V(.77, .27, .13), .058, spaceStructure)
	a.rod(maths.V(.76, .27, .13), maths.V(.80, .27, .13), .051, spaceFoil)
	a.box(.12, .16, .19, maths.V(-.65, .16, .25), spaceMirror)
	// Sunshade straps, individual equipment bay covers, and launch fittings.
	for bay := range 8 {
		angle := (float64(bay) + .5) * math.Pi / 4
		frame := maths.NewSubspace(maths.V(.57*math.Sin(angle), 0, .57*math.Cos(angle)),
			maths.About(maths.YAxis, angle), 1)
		a.boxAt(.33, .28, .074, maths.V(-.165, .075, 0), frame, spaceFoil)
		a.boxAt(.25, .19, .026, maths.V(-.125, .115, .072), frame, body)
		for _, x := range []float64{-.14, .11} {
			a.boxAt(.03, .032, .018, maths.V(x, .314, .071), frame, spaceStructure)
		}
	}
}

// The 0.55 m focal length and 1.37 m diameter describe the NASA
// reflector. The thickness is chosen for the display model.
func marinerReflector() roundProfile {
	const radius, focalLength, thickness = .685, .55, .018
	const rings = 12
	profile := make([][2]float64, 0, 2*(rings+1))
	for ring := 0; ring <= rings; ring++ {
		r := radius * float64(ring) / rings
		profile = append(profile, [2]float64{r, r*r/(4*focalLength) - thickness})
	}
	for ring := rings; ring >= 0; ring-- {
		r := radius * float64(ring) / rings
		profile = append(profile, [2]float64{r, r * r / (4 * focalLength)})
	}
	return roundProfile{turnedProfile{profile}}
}
