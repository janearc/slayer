package smoke

import (
	"math"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// The studio: an infinity cove, a wooden table and a glass. The first
// things in the smokes that are furniture and not a ball: a floor that
// turns up into a wall with no line where it turns, a board with a tree's
// rings in it, and a drinking glass turned as a potter turns one.

// The cove: a floor that curves up into a back wall.
const (
	coveWide   = 24.0 // picked: wider than the frame at the wall, so its side edges are never seen
	coveFront  = 10.0 // picked: how far the floor comes toward the eye, past it
	coveBack   = 5.0  // picked: how far behind the table the wall stands
	coveCurve  = 3.5  // picked: the radius the floor turns up by; large, so the turn has no line
	coveHigh   = 12.0 // picked: the wall's height, above the frame's top
	coveStrips = 48   // picked: strips round the quarter turn; with blended normals it reads as one curve
	quarter    = math.Pi / pair
)

// The table: an oak top on four legs.
const (
	tableLong    = 3.4  // picked: the top's length, along x
	tableDeep    = 1.9  // picked: the top's depth, along z
	tableThick   = 0.12 // picked: the top's thickness, a stout board
	tableHigh    = 1.3  // picked: the top's upper face above the floor
	tableLeg     = 0.14 // picked: a leg's side, square
	tableInset   = 0.12 // picked: how far in from the top's edge a leg stands
	tableRing    = 0.07 // picked: a year's growth in the oak
	tablePith    = 1.1  // picked: how far under the top's face the tree's pith ran, so the face cuts the rings at a slant and the grain has arches in it
	tableWander  = 0.6  // picked: how far the rings wander along the board, in ring widths
	tablePithOff = 0.35 // picked: the pith is off the board's middle, so the grain is not a mirror of itself
)

// The glass: a tumbler, turned.
const (
	glassRadius = 0.2   // picked: the tumbler's radius; with its height, a tumbler's shape
	glassHigh   = 0.62  // picked: the tumbler's height
	glassWall   = 0.02  // picked: the thickness of its wall
	glassBase   = 0.07  // picked: the thickness of its base, heavier than the wall as a tumbler's is
	glassSides  = 128   // picked: sides round the glass; flat ones, fine enough to read round
	glassAtX    = 0.35  // picked: where on the table the glass stands, right of the middle
	glassAtZ    = 0.15  // picked: and toward the eye from it
	glassIndex  = 1.5   // picked: a common glass's index of refraction, as the tracer's own default is
	glassGrey   = 0.04  // picked: a clear thing's own colour shows only where its light is not followed; dark, so that reads as glass and not paint
	glassHair   = 0.002 // picked: the glass stands a hair above the table, so its foot and the table's top are not one face and do not speckle
)

// The water in the glass. It is a body of water that runs half way into
// the glass's wall and half way into its base, and the glass outranks
// it: where the two overlap there is glass, so the water meets the glass
// with no gap of air between them and no two faces in one place.
const (
	waterFull  = 0.72 // picked: how full the glass is, as a share of its height
	waterIndex = 1.33 // picked: water's index as the tracer's own tests have it; not yet read from the refractiveindex database, which holds it
	glassRank  = 1    // defined: the glass outranks the water where they overlap
	waterRank  = 0    // defined: and the water gives way
)

// The studio's lamps and eye.
const (
	studioSpot  = 40.0 // picked: the key's strength, enough for a matte white at its middle to read white
	studioCone  = 24.0 // picked: the key's cone, in degrees, wide enough for the table
	studioSoft  = 12.0 // picked: how far past its cone the key fades, in degrees
	studioFill  = 0.3  // picked: the fill's share of daylight, so shadows are not black
	studioWide  = 38.0 // picked: the lens, in degrees across
	fortyNineNo = 49   // defined: the smoke's number
)

// Where the studio's key lamp stands and where its eye is.
var (
	studioKeyAt = maths.V(-3.2, 6.8, 3.6) // picked: high, to the left and in front
	studioEye   = maths.V(1.7, 2.3, 5.4)  // picked: a little above the table's top, to the right, looking down at the glass
	studioLook  = maths.V(0.25, 1.5, 0)   // picked: just above the table, at the glass
)

// block is a box as a closed mesh of flat faces, from its low corner to
// its high one.
func block(low, high maths.Vec) (*shape.Mesh, error) {
	corners := make([]maths.Vec, 0, 8)
	for _, z := range [2]float64{low.Z, high.Z} {
		for _, y := range [2]float64{low.Y, high.Y} {
			for _, x := range [2]float64{low.X, high.X} {
				corners = append(corners, maths.V(x, y, z))
			}
		}
	}
	// each face wound so its normal points out of the box
	faces := [][3]int{
		{1, 3, 7}, {1, 7, 5}, {0, 4, 6}, {0, 6, 2},
		{2, 6, 7}, {2, 7, 3}, {0, 1, 5}, {0, 5, 4},
		{4, 5, 7}, {4, 7, 6}, {0, 2, 3}, {0, 3, 1},
	}
	return shape.NewMesh(corners, faces, nil, false)
}

// turned is a closed solid turned about the y axis: a profile of radius
// and height, from the axis out and back to the axis, the solid on the
// profile's left as it is walked, swept round in so many flat sides.
func turned(profile [][2]float64, sides int) (*shape.Mesh, error) {
	var corners []maths.Vec
	first := make([]int, len(profile))
	for at, point := range profile {
		first[at] = len(corners)
		if point[0] == nought {
			corners = append(corners, maths.V(nought, point[1], nought))
			continue
		}
		for side := range sides {
			angle := pair * math.Pi * float64(side) / float64(sides)
			corners = append(corners, maths.V(point[0]*math.Cos(angle),
				point[1], point[0]*math.Sin(angle)))
		}
	}
	var faces [][3]int
	for at := range len(profile) - unity {
		from, to := profile[at], profile[at+unity]
		for side := range sides {
			next := (side + unity) % sides
			switch {
			case from[0] == nought && to[0] == nought:
				// a stretch along the axis has no surface
			case from[0] == nought:
				faces = append(faces, [3]int{first[at],
					first[at+unity] + side, first[at+unity] + next})
			case to[0] == nought:
				faces = append(faces, [3]int{first[at] + side,
					first[at+unity], first[at] + next})
			default:
				faces = append(faces,
					[3]int{first[at] + side, first[at+unity] + side,
						first[at+unity] + next},
					[3]int{first[at] + side, first[at+unity] + next,
						first[at] + next})
			}
		}
	}
	return shape.NewMesh(corners, faces, nil, false)
}

// cove is an infinity cove as one sheet: a floor from in front of the eye
// back to where it turns up, a quarter turn of coveCurve, and a wall above
// it. Its normals are the true ones of the curve at every corner and are
// blended across each face, so the turn from floor to wall has no line.
func cove() (*shape.Mesh, error) {
	type rung struct{ y, z, up, out float64 }
	turn := -(coveBack - coveCurve)
	rungs := []rung{{nought, coveFront, unity, nought}}
	for strip := range coveStrips + unity {
		angle := quarter * float64(strip) / float64(coveStrips)
		rungs = append(rungs, rung{coveCurve - coveCurve*math.Cos(angle),
			turn - coveCurve*math.Sin(angle), math.Cos(angle),
			math.Sin(angle)})
	}
	rungs = append(rungs, rung{coveHigh, -coveBack, nought, unity})
	var corners, normals []maths.Vec
	for _, each := range rungs {
		for _, x := range [2]float64{-coveWide / pair, coveWide / pair} {
			corners = append(corners, maths.V(x, each.y, each.z))
			normals = append(normals, maths.V(nought, each.up, each.out))
		}
	}
	var faces [][3]int
	for at := range len(rungs) - unity {
		left, right := pair*at, pair*at+unity
		faces = append(faces, [3]int{left, right, right + pair},
			[3]int{left, right + pair, left + pair})
	}
	return shape.NewMesh(corners, faces, normals, true)
}

// oak is the table's wood, its grain running a way and its pith passing
// through a point.
func oak(along, pith maths.Vec) *thing.Material {
	return &thing.Material{Name: "oak",
		Colour: swatches.Light(swatches.Oak),
		Texture: texture.Wood{Early: swatches.Light(swatches.Oak),
			Late: swatches.Light(swatches.OakLate), Pith: pith,
			Along: along, Ring: tableRing, Wander: tableWander}}
}

// table adds the table to a world: its top, grain along its length, and
// its four legs, grain up each.
func table(w *thing.World) {
	halfLong, halfDeep := tableLong/pair, tableDeep/pair
	under := tableHigh - tableThick
	top := oak(maths.XAxis, maths.V(nought, tableHigh-tablePith, tablePithOff))
	w.Add(thing.Thing{
		Shape: sure(block(maths.V(-halfLong, under, -halfDeep),
			maths.V(halfLong, tableHigh, halfDeep))),
		Material: top, Motion: thing.Still(maths.Local())})
	for _, x := range [2]float64{-unity, unity} {
		for _, z := range [2]float64{-unity, unity} {
			nearX := x * (halfLong - tableInset - tableLeg/pair)
			nearZ := z * (halfDeep - tableInset - tableLeg/pair)
			w.Add(thing.Thing{
				Shape: sure(block(
					maths.V(nearX-tableLeg/pair, nought, nearZ-tableLeg/pair),
					maths.V(nearX+tableLeg/pair, under, nearZ+tableLeg/pair))),
				Material: oak(maths.YAxis,
					maths.V(nearX+tablePithOff, nought, nearZ)),
				Motion: thing.Still(maths.Local())})
		}
	}
}

// tumbler is the glass's shape: out along its foot, up its outside, in
// over its rim, down its inside to the top of its heavy base, and in.
func tumbler() (*shape.Mesh, error) {
	inside := glassRadius - glassWall
	return turned([][2]float64{{nought, nought}, {glassRadius, nought},
		{glassRadius, glassHigh}, {inside, glassHigh}, {inside, glassBase},
		{nought, glassBase}}, glassSides)
}

// poured is the water's shape: a body of water as wide as the middle of
// the glass's wall, from the middle of its base up to the water's level.
func poured() (*shape.Mesh, error) { return pouredTo(waterFull) }

// pouredTo is a drink's shape in the tumbler, filled to a share of the
// glass's height: as wide as the middle of the glass's wall, from the
// middle of its base up to its level.
func pouredTo(full float64) (*shape.Mesh, error) {
	wide := glassRadius - glassWall/pair
	return turned([][2]float64{{nought, glassBase / pair},
		{wide, glassBase / pair}, {wide, glassHigh * full},
		{nought, glassHigh * full}}, glassSides)
}

// fortyNine is the forty-ninth milestone: a glass of water on a wooden
// table in an infinity cove. a white floor curves up into a white wall
// with no line between them. on it stands an oak table whose grain is a
// tree's rings, cut through by the board: long arches on the top, and on
// the board's end the rings themselves. on the table stands a tumbler of
// clear glass, most of the way full of water.
//
// a key lamp from high on the left throws the table's shadow across the
// floor and up the curve, and a soft fill keeps the shadow from black.
//
// the water touches the glass. a look through the middle of the glass
// bends four times: from the air into the glass, from the glass into the
// water, from the water into the glass, and from the glass into the air.
// below the water's level the table's grain is bent more than above it,
// where there is only glass and air.
func fortyNine() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: sure(cove()),
		Material: painted("cove", swatches.Light(swatches.Paint)),
		Motion:   thing.Still(maths.Local())})
	table(w)
	w.Add(thing.Thing{Shape: sure(tumbler()),
		Material: &thing.Material{Name: "glass",
			Colour: maths.Grey(glassGrey), Gloss: unity,
			Index: glassIndex, Through: unity, Rank: glassRank},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(glassAtX, tableHigh+glassHair, glassAtZ)})})
	w.Add(thing.Thing{Shape: sure(poured()),
		Material: &thing.Material{Name: "water",
			Colour: maths.Grey(glassGrey), Gloss: unity,
			Index: waterIndex, Through: unity, Rank: waterRank},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(glassAtX, tableHigh+glassHair, glassAtZ)})})
	glass := maths.V(glassAtX, tableHigh+glassHigh/pair, glassAtZ)
	// both lamps are daylight, so the white cove is white: a warm key
	// over a sky-blue fill painted it lavender
	w.Lamp(film.Spot{At: studioKeyAt, Aim: glass.Sub(studioKeyAt),
		Strength: matte(swatches.Light(swatches.Daylight).Scale(studioSpot)),
		Cone:     studioCone * maths.Degree, Soft: studioSoft * maths.Degree})
	w.Lamp(film.Flood{
		Strength: swatches.Light(swatches.Daylight).Scale(studioFill)})
	return w, maths.Subspace{Origin: studioEye,
		Orientation: maths.LookAt(studioEye, studioLook)}
}
