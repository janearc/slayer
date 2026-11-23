package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// the matters a street is made of, each with its own look.
const (
	streetAsphalt slot = iota
	streetKerb
	streetPavement
	streetPaint
	streetGrate
	streetWater
)

// the numbers the street's markings and fittings are laid out with.
const (
	// streetDashLong and streetDashPeriod are the length of a painted
	// centre dash and the distance from the start of one to the next, in
	// metres. streetDashWide is how wide it is. picked.
	streetDashLong   = 3.0
	streetDashPeriod = 9.0
	streetDashWide   = 0.12
	// streetCrossingAt is how far along the street its crossing lies, as
	// a share of its length. streetCrossingLong is the crossing's length
	// along the street, and streetStripe the width of each of its
	// stripes and of the gaps between, in metres. picked.
	streetCrossingAt   = 0.45
	streetCrossingLong = 3.0
	streetStripe       = 0.5
	// streetStripes is how many stripes the crossing has. picked: seven
	// of half a metre with six gaps of half a metre is six and a half
	// metres, which the carriageway holds.
	streetStripes = 7
	// streetClear is how far the centre dashes and the gratings keep
	// from the crossing, in metres.
	streetClear = 1.0
	// streetGrateWide, streetGrateLong and streetGrateSpacing are the
	// gutter gratings' width across the gutter, their length along it, and
	// the distance between one and the next. picked.
	streetGrateWide    = 0.48
	streetGrateLong    = 0.6
	streetGrateSpacing = 12.0
	// streetGrateFirst is how far from the street's end the first grating
	// lies, in metres.
	streetGrateFirst = 4.0
	// streetPaintLift and streetGrateLift are how far the paint and the
	// gratings lie above the road, in metres: enough that the road does
	// not show through between the road's corners.
	streetPaintLift = 0.003
	streetGrateLift = 0.004
)

// streetSheet is a form that is faces already cut: the street's sheets of
// paint, water and gratings are laid on it, as the road is, in the form's
// own space. It is shaded flat.
type streetSheet struct{ faces surface }

// skin is the sheet's faces, however finely it is asked to be cut.
func (sheet streetSheet) skin(cut) surface { return sheet.faces }

// streetRoad is the carriageway: a sheet of faces over the ground, shaded
// smooth by the ground's own slope, so that the camber and the dishes
// mirror as one road and not as a field of flat cells.
type streetRoad struct {
	streetSheet
	ground streetGround
}

// normals is the road's normal at each corner of each face, from the
// slope of the ground there.
func (road streetRoad) normals(skin surface) [][3]maths.Vec {
	normals := make([][3]maths.Vec, len(skin))
	for index, one := range skin {
		for corner, point := range one {
			across, along := road.ground.slope(point.X, point.Z)
			normals[index][corner] = maths.V(-across, 1, -along).Unit()
		}
	}
	return normals
}

// streetRect is a rectangle of the street, from one distance to another
// across it and along it, in metres.
type streetRect struct{ across, along [2]float64 }

// drape is the sheet of faces that lies on the ground over a rectangle,
// lift above it, one cell at a time, wound up.
func (ground streetGround) drape(rect streetRect, lift float64) surface {
	columns := cells(rect.across, streetCellAcross)
	rows := cells(rect.along, streetCellAlong)
	point := func(column, row int) maths.Vec {
		x := lerp(rect.across[0], rect.across[1],
			float64(column)/float64(columns))
		z := lerp(rect.along[0], rect.along[1], float64(row)/float64(rows))
		return maths.V(x, ground.height(x, z)+lift, z)
	}
	var skin surface
	for column := range columns {
		for row := range rows {
			skin = quad(skin, point(column, row), point(column, row+1),
				point(column+1, row+1), point(column+1, row))
		}
	}
	return skin
}

// cells is how many cells of this side a span is cut into, at least one.
func cells(span [2]float64, side float64) int {
	return max(1, int(math.Ceil((span[1]-span[0])/side)))
}

// sheet adds a sheet of faces to the street as a matter, if it has any.
func (assembly *instrumentAssembly) sheet(skin surface, of slot) {
	if len(skin) > 0 {
		assembly.add(streetSheet{faces: skin}, maths.Local(), of)
	}
}

// build lays the street into the assembly: the road, the stones along its
// sides, its markings and gratings, and the water on it.
func (ground streetGround) build(assembly *instrumentAssembly) {
	whole := streetRect{across: [2]float64{-streetRoadHalf, streetRoadHalf},
		along: [2]float64{-ground.length / 2, ground.length / 2}}
	assembly.add(streetRoad{streetSheet{ground.drape(whole, 0)}, ground},
		maths.Local(), streetAsphalt)
	ground.stones(assembly)
	assembly.sheet(ground.markings(), streetPaint)
	assembly.sheet(ground.gratings(), streetGrate)
	assembly.sheet(ground.puddles(), streetWater)
	assembly.sheet(ground.flood(), streetWater)
}

// stones lays the kerb and the footway on each side of the road: closed
// boxes standing on a footing, their tops level.
func (ground streetGround) stones(assembly *instrumentAssembly) {
	floor := streetKerbUp - streetFooting
	for _, side := range []float64{-1, 1} {
		kerb := streetRoadHalf
		if side < 0 {
			kerb = -streetRoadHalf - streetKerbWide
		}
		assembly.boxAt(streetKerbWide, streetFooting, ground.length,
			maths.V(kerb, floor, -ground.length/2), maths.Local(), streetKerb)
		footway := streetRoadHalf + streetKerbWide
		if side < 0 {
			footway = -footway - streetFootway
		}
		assembly.boxAt(streetFootway, streetFooting, ground.length,
			maths.V(footway, floor, -ground.length/2), maths.Local(),
			streetPavement)
	}
}

// crossing is where the crossing lies along the street: its near and far
// edges.
func (ground streetGround) crossing() [2]float64 {
	middle := ground.length * (streetCrossingAt - 0.5)
	return [2]float64{middle - streetCrossingLong/2,
		middle + streetCrossingLong/2}
}

// markings is the street's paint: the dashes of its centre line, and the
// stripes of its crossing, laid on the ground.
func (ground streetGround) markings() surface {
	var paint surface
	crossing := ground.crossing()
	for start := -ground.length/2 + 2; start+streetDashLong <
		ground.length/2-0.5; start += streetDashPeriod {
		if start+streetDashLong > crossing[0]-streetClear &&
			start < crossing[1]+streetClear {
			continue
		}
		paint = append(paint, ground.drape(streetRect{
			across: [2]float64{-streetDashWide / 2, streetDashWide / 2},
			along:  [2]float64{start, start + streetDashLong}},
			streetPaintLift)...)
	}
	span := float64(2*streetStripes-1) * streetStripe
	for stripe := range streetStripes {
		left := -span/2 + float64(2*stripe)*streetStripe
		paint = append(paint, ground.drape(streetRect{
			across: [2]float64{left, left + streetStripe}, along: crossing},
			streetPaintLift)...)
	}
	return paint
}

// gratings is the gully gratings in the gutter of each side, spaced down
// the street and clear of its crossing.
func (ground streetGround) gratings() surface {
	var iron surface
	crossing := ground.crossing()
	for start := -ground.length/2 + streetGrateFirst; start+streetGrateLong <
		ground.length/2-streetClear; start += streetGrateSpacing {
		if start+streetGrateLong > crossing[0]-streetClear &&
			start < crossing[1]+streetClear {
			continue
		}
		for _, side := range []float64{-1, 1} {
			edge := side * streetRoadHalf
			iron = append(iron, ground.drape(streetRect{
				across: [2]float64{math.Min(edge, edge-side*streetGrateWide),
					math.Max(edge, edge-side*streetGrateWide)},
				along: [2]float64{start, start + streetGrateLong}},
				streetGrateLift)...)
		}
	}
	return iron
}
