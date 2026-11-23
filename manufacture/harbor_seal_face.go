package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A lofted seal's face. The eyes are large, dark and set forward; the
// nostrils are "small and terminal, forming a "V" that converges at the
// bottom" (Duke OBIS-SEAMAP, harbor seal species page). Places and sizes
// in metres at rest, picked.
const (
	// An eye's ball: the station it sits at, how far round from the top
	// of the head, its radius, and how deep its centre lies in the skin.
	sealFaceEyeX, sealFaceEyeTurn     = .756, .88
	sealFaceEyeRadius, sealFaceEyeSet = .021, .011
	// The dark skin round an eye: whole within the first distance of the
	// eye's place on the skin and gone by the second; its strength.
	sealFaceRimFrom, sealFaceRimTo, sealFaceRim = .021, .036, .6
	// An ear is a small dark opening, with no flap, "set slightly behind
	// and below the eyes" (the same page): its station, its turn from
	// the top, its radius and its strength.
	sealFaceEarX, sealFaceEarTurn    = .716, 1.12
	sealFaceEarWide, sealFaceEarDark = .006, .75
	// A nostril is a slit on the nose's front from its upper, outer end
	// to its lower, inner end, as heights over the nose's centre and
	// distances from the middle; its half width and strength.
	sealFaceSlitFrom                   = .800
	sealFaceSlitHigh, sealFaceSlitOut  = .012, .016
	sealFaceSlitLow, sealFaceSlitIn    = -.005, .005
	sealFaceSlitWide, sealFaceSlitDark = .0032, .9
	// The mouth's line runs back along the muzzle from under the nose,
	// this share of the section's half height under its centre.
	sealFaceMouthFrom, sealFaceMouthTo  = .765, .826
	sealFaceMouthLow, sealFaceMouthWide = .55, .0022
	sealFaceMouthDark                   = .6
)

// eye is where an eye's ball meets the outline's skin at rest, and the
// way out of the head there; side is one for the right eye.
func (outline sealOutline) eye(side float64) (onSkin, out maths.Vec) {
	return outline.radial(sealFaceEyeX, side*sealFaceEyeTurn)
}

// sealFaceInk is how far the face's dark lines blacken the coat at a
// rest point of the outline's skin, from none to one: the skin round
// the eyes, the ears' openings, the nostrils' slits and the line of
// the mouth.
func sealFaceInk(outline sealOutline, point maths.Vec,
	section sealSection,
) float64 {
	if point.X < sealFaceEarX-sealFaceRimTo {
		return 0
	}
	side := 1.
	if point.Z < 0 {
		side = -1
	}
	eye, _ := outline.eye(side)
	ink := sealFaceRim * (1 - maths.Smooth(sealFaceRimFrom, sealFaceRimTo,
		point.Sub(eye).Len()))
	ear, _ := outline.radial(sealFaceEarX, side*sealFaceEarTurn)
	ink = math.Max(ink, sealFaceEarDark*(1-maths.Smooth(sealFaceEarWide/2,
		sealFaceEarWide, point.Sub(ear).Len())))
	return math.Max(ink, math.Max(sealFaceSlit(point, section),
		sealFaceMouth(point, section)))
}

// sealFaceSlit is how far a nostril's slit blackens a rest point.
func sealFaceSlit(point maths.Vec, section sealSection) float64 {
	if point.X < sealFaceSlitFrom {
		return 0
	}
	height, out := point.Y-section.y, math.Abs(point.Z)
	alongY := sealFaceSlitLow - sealFaceSlitHigh
	alongZ := sealFaceSlitIn - sealFaceSlitOut
	share := ((height-sealFaceSlitHigh)*alongY +
		(out-sealFaceSlitOut)*alongZ) / (alongY*alongY + alongZ*alongZ)
	share = math.Min(math.Max(share, 0), 1)
	gap := math.Hypot(height-sealFaceSlitHigh-share*alongY,
		out-sealFaceSlitOut-share*alongZ)
	return sealFaceSlitDark *
		(1 - maths.Smooth(sealFaceSlitWide/2, sealFaceSlitWide, gap))
}

// sealFaceMouth is how far the mouth's line blackens a rest point.
func sealFaceMouth(point maths.Vec, section sealSection) float64 {
	if point.X < sealFaceMouthFrom || point.X > sealFaceMouthTo {
		return 0
	}
	line := section.y - sealFaceMouthLow*section.height
	return sealFaceMouthDark * (1 - maths.Smooth(sealFaceMouthWide/2,
		sealFaceMouthWide, math.Abs(point.Y-line)))
}

// The whiskers of the muzzle: "~44 whiskers in seven rows" (Graff and
// others 2024, Journal of Experimental Biology 227(22), Europe PMC
// PMC11634035), read as one side's. Each row's count, the places,
// lengths, radii and the droop are picked.
var sealBristleRows = [...]int{7, 8, 7, 7, 6, 5, 4}

const (
	// The pad: the rearmost column's station and the gap to the next,
	// the lowest row's turn from the top of the muzzle and the turn up
	// to the next row.
	sealBristleX, sealBristleStep    = .768, .0052
	sealBristleTurn, sealBristleRise = 1.98, .115
	// The longest strand, at the rear of the lowest row, and how much
	// shorter each column forward and each row up is.
	sealBristleLong                = .072
	sealBristleLess, sealBristleUp = .0050, .0040
	sealBristleLeast               = .018
	// A strand leaves the skin along its normal, leaning along the body
	// by sealBristleAft at the rearmost column and by sealBristleFore
	// eight columns on, so the pad's strands fan; it droops this share
	// of its length, and is drawn as this many rods of this radius.
	sealBristleAft, sealBristleFore = -.50, .25
	sealBristleDroop                = .12
	sealBristleRods                 = 4
	sealBristleRadius               = .00025
	sealBristleColumns              = 8
)

// bristles is one side's whiskers on the outline's muzzle at rest, each
// a line of points from its root on the skin to its tip.
func (outline sealOutline) bristles(side float64) [][]maths.Vec {
	var strands [][]maths.Vec
	for row, count := range sealBristleRows {
		turn := side * (sealBristleTurn - sealBristleRise*float64(row))
		for column := range count {
			root, out := outline.radial(sealBristleX+
				sealBristleStep*float64(column), turn)
			long := math.Max(sealBristleLeast, sealBristleLong-
				sealBristleLess*float64(column)-sealBristleUp*float64(row))
			lean := sealBristleAft + (sealBristleFore-sealBristleAft)*
				float64(column)/sealBristleColumns
			way := out.Add(maths.V(lean, 0, 0)).Unit()
			strand := make([]maths.Vec, sealBristleRods+1)
			for at := range strand {
				share := float64(at) / sealBristleRods
				strand[at] = root.Add(way.Scale(long * share)).Add(
					maths.V(0, -sealBristleDroop*long*share*share, 0))
			}
			strands = append(strands, strand)
		}
	}
	return strands
}

var (
	// sealFaceEyeLook is the eye: near black and wet. Picked.
	sealFaceEyeLook = &thing.Material{Name: "seal eye",
		Colour: maths.RGB(.010, .008, .007), Gloss: .6, Satin: .12}
	// sealBristleLook is a pale whisker. Picked.
	sealBristleLook = &thing.Material{Name: "seal whisker",
		Colour: maths.RGB(.46, .43, .37), Gloss: .15, Satin: .4}
)

// face is the eyes and the whiskers of both sides of the outline's
// head. place takes a rest point of the head to where it ends up, and
// scale is the animal's size over the rest space's.
func (outline sealOutline) face(place func(maths.Vec) maths.Vec,
	scale float64,
) []thing.Thing {
	var parts []thing.Thing
	for _, side := range [...]float64{1, -1} {
		onSkin, out := outline.eye(side)
		centre := place(onSkin.Sub(out.Scale(sealFaceEyeSet)))
		parts = append(parts, thing.Thing{
			Shape:    shape.Sphere{Radius: scale * sealFaceEyeRadius},
			Material: sealFaceEyeLook,
			Motion: thing.Still(maths.NewSubspace(centre,
				maths.Identity, 1))})
		for _, strand := range outline.bristles(side) {
			parts = append(parts, sealBristleRodsOf(strand, place,
				scale*sealBristleRadius)...)
		}
	}
	return parts
}

// sealBristleRodsOf is one whisker as rods of a radius, end to end
// through its placed points.
func sealBristleRodsOf(strand []maths.Vec,
	place func(maths.Vec) maths.Vec, radius float64,
) []thing.Thing {
	var rods []thing.Thing
	previous := place(strand[0])
	for _, rest := range strand[1:] {
		at := place(rest)
		run := at.Sub(previous)
		rods = append(rods, thing.Thing{
			Shape: shape.Tube{Radius: radius, Thick: 2 * radius,
				Height: run.Len()},
			Material: sealBristleLook,
			Motion: thing.Still(maths.NewSubspace(
				previous.Add(at).Scale(.5), maths.Upright(run), 1))})
		previous = at
	}
	return rods
}
