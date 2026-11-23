package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// SOHO's 4.3 m sun-axis length and 9.5 m array span are from ESA:
// https://www.esa.int/Enabling_Support/Operations/SOHO_operations
// Component placements, array subdivisions, colours and remaining dimensions
// are visual estimates from spacecraft images, not engineering dimensions.
const (
	sohoAperture slot = spaceSecondary + 1
	sohoGold          = sohoAperture + 1
)

type soho struct{}

// madeAt is made at any detail: the craft has the one representation.
func (s soho) madeAt(from seed, detail string) (good, error) {
	return s.made(from)
}

// made assembles the craft from its parts and materials. It refuses a seed
// that was not made.
func (soho) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	a := &instrumentAssembly{fine: cut{around: 28}, object: good{dressed: looks{
		body:           thing.Polished("black thermal blanket", maths.RGB(.035, .03, .028), .15, 1.7),
		leaf:           thing.Polished("blue-black solar cells", maths.RGB(.018, .026, .062), .36, 1.8),
		spaceStructure: thing.Polished("structural rails", maths.Grey(.56), .3, 2),
		spaceFoil:      thing.Diffuse("dark payload blanket", maths.RGB(.055, .045, .04)),
		sohoAperture:   thing.Diffuse("instrument apertures", maths.Grey(.035)),
		sohoGold:       thing.Polished("gold thermal foil", maths.RGB(.57, .37, .083), .52, 1.9),
	}}}
	// Service module and payload module, both closed bodies.
	a.box(2.05, 1.0, 1.9, maths.V(-1.025, -.65, -.95), body)
	a.box(1.75, 2.0, 1.6, maths.V(-.875, .35, -.8), spaceFoil)
	// A rail-framed pair of solar arrays; the outboard edges are 9.5 m
	// apart, measured to the outer face of the frame.
	for _, side := range []float64{-1, 1} {
		sohoWing(side).pulledIn().lay(a)
	}
	// ESA's front view shows a flat sunward deck, a large gold side barrel,
	// smaller cylindrical housings and a largely dark insulated payload.
	// The placements and proportions are chosen by eye from that image.
	a.box(1.9, .08, 1.8, maths.V(-.95, 2.35, -.9), spaceStructure)
	a.rod(maths.V(-.98, .35, .18), maths.V(-.98, 2.27, .18), .30, sohoGold)
	a.rod(maths.V(-.98, 2.27, .18), maths.V(-.98, 2.38, .18), .33, spaceStructure)
	for _, z := range []float64{-.43, .67} {
		a.rod(maths.V(-.64, 1.05, z), maths.V(-.64, 2.15, z), .12, sohoGold)
	}
	for _, x := range []float64{-.15, .45} {
		a.box(.40, .65, .12, maths.V(x, 1.13, .81), body)
		a.box(.33, .23, .035, maths.V(x+.03, 1.34, .935), spaceStructure)
	}
	for _, y := range []float64{.6, .85} {
		a.rod(maths.V(.72, y, .78), maths.V(.72, y, 1.16), .08, body)
		a.rod(maths.V(.72, y, 1.16), maths.V(.72, y, 1.19), .06, sohoAperture)
	}
	// Layered blankets, seams, hardware boxes and braces on the payload face.
	for _, y := range []float64{.48, .86, 1.24, 1.62, 2.0} {
		a.box(1.7, .018, .035, maths.V(-.85, y, .805), spaceStructure)
	}
	for _, x := range []float64{-.78, -.52, -.26, 0, .26, .52, .78} {
		a.box(.014, 1.85, .025, maths.V(x, .42, .81), body)
	}
	for i := 0; i < 10; i++ {
		x := -.72 + float64(i)*.16
		a.box(.04, .035, .035, maths.V(x, 2.39, .65), sohoGold)
	}
	for _, side := range []float64{-1, 1} {
		a.rod(maths.V(side*.92, .36, -.75), maths.V(side*.92, 2.30, -.75), .025, spaceStructure)
		a.rod(maths.V(side*.92, .36, .75), maths.V(side*.92, 2.30, .75), .025, spaceStructure)
		a.rod(maths.V(side*.92, .36, -.75), maths.V(side*.92, 2.30, .75), .018, spaceStructure)
	}
	// Paired equipment bay covers, radiators and rows of visible fasteners.
	for _, side := range []float64{-1, 1} {
		x := side * 1.025
		if side < 0 {
			x -= .035
		}
		a.box(.035, .9, 1.45, maths.V(x, -.45, -.725), body)
		for i := 0; i < 12; i++ {
			for j := 0; j < 4; j++ {
				y := -.4 + float64(j)*.24
				z := -.66 + float64(i)*.12
				a.box(.02, .035, .035, maths.V(x+side*.04, y, z), spaceStructure)
			}
		}
	}
	sohoFlightFittings(a)
	// Communications dish and support. Its pointing is a visual choice.
	a.rod(maths.V(0, -.65, 0), maths.V(0, -1.17, 0), .05, spaceStructure)
	a.add(mirrorDish{radius: .44, depth: .12}, maths.NewSubspace(maths.V(0, -1.18, 0), maths.About(maths.XAxis, math.Pi), 1), spaceFoil)
	a.rod(maths.V(0, -1.31, 0), maths.V(0, -1.55, 0), .052, spaceStructure)
	for arm := range 3 {
		angle := float64(arm) * 2 * math.Pi / 3
		rim := maths.V(.36*math.Cos(angle), -1.27, .36*math.Sin(angle))
		a.rod(rim, maths.V(0, -1.53, 0), .012, spaceStructure)
	}
	if err := a.shareSolar(leaf); err != nil {
		return good{}, err
	}
	bounds := maths.SegmentBox(maths.V(-4.75, -1.8, -2.15), maths.V(4.75, 2.9, 2.15), 0)
	object, err := a.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *object, nil
}

// SOHO's arrays, in metres, estimated from images like the rest of the
// craft: where each yoke starts, how long it is, how far the first panel
// is from its root, a panel's size and the gap between two, and the
// cells, 10 along the wing and 9 across it.
const (
	sohoYokeRoot    = 1.0
	sohoYokeLength  = .2
	sohoYokeRadius  = .045
	sohoPanelReach  = .15
	sohoPanelLength = 1.78
	sohoPanelWidth  = 1.65
	sohoPanelThick  = .07
	sohoPanelGap    = .04
	sohoPanels      = 2
	sohoCellColumns = 10
	sohoCellRows    = 9
	sohoCellWidth   = .164
	sohoCellDepth   = .165
	sohoCellHeight  = .012
	sohoCellPitch   = .175
	sohoBarShare    = .5
)

// sohoWing is one wing as an array: two panels in a row out from a short
// yoke at the bus, hinged to each other, the cells facing up.
func sohoWing(side float64) solarArray {
	return solarArray{
		frame: solarFrame(maths.V(side*sohoYokeRoot, 0, 0),
			maths.V(side, 0, 0), maths.V(0, 1, 0)),
		panels: solarPanels{along: sohoPanels, across: 1,
			width: sohoPanelLength, depth: sohoPanelWidth,
			thickness: sohoPanelThick, gap: sohoPanelGap,
			reach: sohoPanelReach},
		cells: solarCells{columns: sohoCellColumns, rows: sohoCellRows,
			width: sohoCellWidth, depth: sohoCellDepth,
			height: sohoCellHeight, pitchX: sohoCellPitch,
			pitchZ: sohoCellPitch, barShare: sohoBarShare},
		yoke: solarYoke{length: sohoYokeLength, radius: sohoYokeRadius,
			across: []float64{0}},
		looks: solarLooks{cell: leaf, bar: sohoGold,
			substrate: spaceStructure, frame: spaceFoil,
			hardware: spaceStructure},
	}
}

// SOHO assembly photos inform these fittings. Their dimensions and routing
// are visual estimates.
func sohoFlightFittings(a *instrumentAssembly) {
	for row := range 3 {
		for col := range 4 {
			x := -.92 + float64(col)*.46
			z := -.86 + float64(row)*.57
			a.box(.448, .035, .558, maths.V(x, -.683, z), sohoGold)
			a.box(.032, .032, .038, maths.V(x+.017, -.714, z+.017), spaceStructure)
			a.box(.032, .032, .038, maths.V(x+.399, -.714, z+.506), spaceStructure)
		}
	}
	for _, side := range []float64{-1, 1} {
		for _, z := range []float64{-.62, .62} {
			a.rod(maths.V(side*.88, -.31, z), maths.V(side*1.20, -.055, z), .033, sohoGold)
			a.rod(maths.V(side*2.88, -.045, z), maths.V(side*3.02, -.045, z), .027, spaceStructure)
		}
	}
	for i := range 5 {
		y := .47 + float64(i)*.33
		x := -.64 + float64(i%2)*.21
		widths := [...]float64{.48, .36, .53, .41, .48}
		width := widths[i]
		a.box(width, .16, .14, maths.V(x, y, -.929), spaceStructure)
		a.box(width-.13, .105, .025, maths.V(x+.060, y+.027, -.954), body)
		for port := range 2 + i%2 {
			px := x + .092 + float64(port)*.111
			a.rod(maths.V(px, y+.08, -.944), maths.V(px, y+.08, -.978), .024, sohoGold)
		}
		a.rod(maths.V(x+width-.028, y+.10, -.97), maths.V(.40, y+.19, -.97), .012, body)
	}
	for i := range 4 {
		x := .46 + float64(i)*.062
		a.rod(maths.V(x, .47, -.994), maths.V(x, 2.20-float64(i)*.12, -.994), .010, body)
	}
	a.box(.22, .61, .13, maths.V(.55, .64, -.946), sohoGold)
	a.box(.19, .38, .11, maths.V(.58, 1.43, -.932), spaceStructure)
	for rib := range 12 {
		y := .665 + float64(rib)*.047
		a.box(.195, .012, .016, maths.V(.563, y, -.958), body)
	}
	for _, y := range []float64{.72, 1.18, 1.58} {
		a.rod(maths.V(.77, y, -.906), maths.V(.89, y+.12, -.73), .017, sohoGold)
	}
	for _, head := range []struct{ x, z, radius, height float64 }{
		{-.40, .44, .14, .17}, {.37, .35, .20, .25}, {.51, -.40, .105, .11},
	} {
		top := 2.43 + head.height
		a.rod(maths.V(head.x, 2.429, head.z), maths.V(head.x, top, head.z), head.radius, body)
		a.rod(maths.V(head.x, top, head.z), maths.V(head.x, top+.024, head.z), head.radius*1.06, spaceStructure)
		a.rod(maths.V(head.x, top+.024, head.z), maths.V(head.x, top+.026, head.z), head.radius*.83, sohoAperture)
	}
}
