package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The deployed layout follows the September 2012 NASA concept diagram,
// NTRS 20120016895, page 5. Lengths and finish are chosen for the visual
// model. This is an unflown configuration.
// https://ntrs.nasa.gov/citations/20120016895
type jimo struct{}

// madeAt is made at any detail: the craft has the one representation.
func (m jimo) madeAt(from seed, detail string) (good, error) {
	return m.made(from)
}

// made assembles the craft from its parts and materials. It refuses a seed
// that was not made.
func (jimo) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	a := &instrumentAssembly{fine: cut{around: 16}, object: good{dressed: looks{
		body:           thing.Polished("gold equipment blanket", maths.RGB(.50, .29, .071), .30, 1.7),
		leaf:           thing.Polished("blue-grey heat rejection faces", maths.RGB(.095, .15, .23), .36, 1.7),
		spaceStructure: thing.Polished("aluminium structure", maths.RGB(.48, .51, .56), .40, 1.8),
		spaceFoil:      thing.Diffuse("dark electronics", maths.Grey(.045)),
		spaceMirror:    thing.Diffuse("antenna reflecting surface", maths.Grey(.56)),
		spaceSecondary: thing.Polished("heat rejection tubing", maths.RGB(.21, .31, .41), .29, 1.8),
	}}}
	jimoBoom(a)
	jimoScienceBus(a)
	bounds := maths.SegmentBox(maths.V(-5.8, -4, -2.6), maths.V(5.8, 35, 3.9), 0)
	g, err := a.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *g, nil
}

// jimoBoom adds the boom: four radiator banks, each narrower than the one
// before, and the hinges between them.
func jimoBoom(a *instrumentAssembly) {
	// Four progressively narrower radiator banks and three boom hinges.
	for _, bank := range []struct {
		base, height, span float64
		bays               int
	}{
		{3.6, 7.9, 5.5, 13}, {12.1, 7.7, 4.4, 12},
		{20.35, 6.5, 3.2, 11}, {27.4, 3.15, 1.65, 6},
	} {
		for _, side := range []float64{-1, 1} {
			jimoRadiators(a, side, bank.base, bank.height, bank.span, bank.bays)
		}
	}
	for _, side := range []float64{-1, 1} {
		for _, z := range []float64{-.39, .39} {
			x := side * .40
			a.rod(maths.V(x, 2.3, z), maths.V(x, 31.2, z), .067, spaceStructure)
			for bay := range 19 {
				y := 2.35 + float64(bay)*1.48
				if side > 0 {
					a.rod(maths.V(x, y, z), maths.V(-x, y, z), .039, spaceStructure)
					a.rod(maths.V(x, y, z), maths.V(-x, y+1.45, z), .031, body)
				}
			}
		}
	}
	for _, y := range []float64{11.5, 19.8, 27.0} {
		a.rod(maths.V(0, y, -.62), maths.V(0, y, .62), .22, body)
		a.box(1.02, .36, .94, maths.V(-.51, y-.18, -.47), spaceFoil)
	}
	// Exterior silhouette of the power assembly, with no internals modeled.
	a.rod(maths.V(0, 30.8, 0), maths.V(0, 33.55, 0), .51, spaceStructure)
	a.rod(maths.V(0, 33.55, 0), maths.V(0, 33.90, 0), .69, spaceFoil)
	a.rod(maths.V(0, 33.9, 0), maths.V(0, 34.69, 0), .39, body)
	for _, x := range []float64{-.40, .40} {
		a.rod(maths.V(x, 31.05, .31), maths.V(x, 33.3, .31), .06, spaceSecondary)
	}
}

// jimoRadiators adds one bank of radiator bays at a base and height along
// the boom, span wide, to the side given by the sign of side.
func jimoRadiators(a *instrumentAssembly, side, base, height, span float64, bays int) {
	left := .57
	if side < 0 {
		left = -span
	}
	width := span - .57
	pitch := height / float64(bays)
	for bay := range bays {
		y := base + float64(bay)*pitch
		a.box(width, pitch-.025, .085, maths.V(left, y, -.045), leaf)
		a.box(width, .045, .11, maths.V(left, y, -.056), spaceStructure)
		// The radiator faces carry parallel heat-pipe runs and end clips.
		for i := range 7 {
			x := left + (float64(i)+.5)*width/7
			a.rod(maths.V(x, y+.062, .052), maths.V(x, y+pitch-.055, .052), .012, spaceSecondary)
			a.box(.043, .032, .038, maths.V(x-.022, y+.044, .037), spaceStructure)
		}
	}
	for _, x := range []float64{left + .018, left + width - .018} {
		a.rod(maths.V(x, base, 0), maths.V(x, base+height, 0), .037, spaceStructure)
	}
}

// jimoScienceBus adds the science bus: the box of the body, three rods
// with foil boxes, and a finned panel on each side.
func jimoScienceBus(a *instrumentAssembly) {
	a.box(2.3, 3.45, 1.85, maths.V(-1.15, -.95, -.93), body)
	for _, x := range []float64{-.74, .0, .74} {
		a.rod(maths.V(x, -.78, 1.04), maths.V(x, .92, 1.04), .24, spaceStructure)
		a.box(.48, .25, .20, maths.V(x-.24, 1.52, .92), spaceFoil)
	}
	for _, side := range []float64{-1, 1} {
		left := 1.10
		if side < 0 {
			left = -2.42
		}
		a.box(1.32, 1.62, .063, maths.V(left, .51, -.24), leaf)
		for fin := range 7 {
			x := left + .10 + float64(fin)*.174
			a.box(.032, 1.52, .041, maths.V(x, .56, -.174), spaceSecondary)
		}
		a.box(.92, 1.46, .069, maths.V(left, 15.1, -.25), spaceSecondary)
	}
	a.rod(maths.V(0, -2.65, 0), maths.V(0, -.87, 0), .25, spaceStructure)
	a.rod(maths.V(0, -2.89, 0), maths.V(0, -2.63, 0), .88, spaceFoil)
	for _, side := range []float64{-1, 1} {
		x := side * 2.54
		a.rod(maths.V(side*.88, -.49, 0), maths.V(x, -.18, 0), .088, spaceStructure)
		a.rod(maths.V(side*.91, .08, -.57), maths.V(x, -.10, -.30), .043, spaceStructure)
		a.box(.96, .83, .72, maths.V(x-.48, -.55, -.44), body)
		for _, dx := range []float64{-.23, .23} {
			for _, y := range []float64{-.33, .02} {
				a.rod(maths.V(x+dx, y, .23), maths.V(x+dx, y, .43), .135, spaceStructure)
				a.rod(maths.V(x+dx, y, .429), maths.V(x+dx, y, .454), .097, spaceFoil)
			}
		}
	}
	a.rod(maths.V(0, .55, .82), maths.V(0, .18, 2.24), .11, spaceStructure)
	dish := maths.NewSubspace(maths.V(0, .18, 2.24), maths.About(maths.XAxis, math.Pi/2), 1)
	a.fine.around = 40
	a.add(mirrorDish{radius: 1.15, depth: .18}, dish, spaceMirror)
	a.rod(maths.V(0, .18, 2.27), maths.V(0, .18, 2.98), .046, spaceFoil)
}
