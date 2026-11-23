package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The numbers of the lattice hash the cloth noise is drawn with, each
// defined by the published hash it is taken from: the three primes are
// Teschner and others' spatial hash, and the multiplier and the shifts
// are MurmurHash2's finaliser.
const (
	// clothHashA is defined by Teschner's spatial hash: the prime that
	// mixes a lattice point's x.
	clothHashA = 73856093
	// clothHashB is defined by Teschner's spatial hash: the prime for y.
	clothHashB = 19349663
	// clothHashC is defined by Teschner's spatial hash: the prime for z.
	clothHashC = 83492791
	// clothHashMix is defined by MurmurHash2: its multiplier.
	clothHashMix = 0x5bd1e995
	// clothHashShift is defined by MurmurHash2: its first right shift.
	clothHashShift = 13
	// clothHashRange is defined: the largest 16 bit value, which a hash is
	// kept to.
	clothHashRange = 0xffff
	// twistEdge is defined: the middle of a strand's wave, which runs
	// from nought to one as a cosine laid on a half.
	twistEdge = 0.5
	// diamondInside is defined: the diamond's edge as a share of its
	// cell, half, the largest a lozenge can be and leave its neighbours
	// touching at their corners.
	diamondInside = 0.5
)

// the numbers the cloth is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// clothStreakSize is how wide, in metres, a rain streak is.
	clothStreakSize = pickedNumber("cloth-streak-size")
	// clothMottle is how big, in metres, the mottle of soil is.
	clothMottle = pickedNumber("cloth-mottle")
	// clothMottleBit is how much of the soil the mottle moves.
	clothMottleBit = pickedNumber("cloth-mottle-bit")
	// clothStreakFall is how many soil heights up a rain streak still
	// shows.
	clothStreakFall = pickedNumber("cloth-streak-fall")
	// fibreFine is how many times finer the fine fibre is than the coarse.
	fibreFine = pickedNumber("fibre-fine")
	// fibreCoarseBit is how much the coarse fibre moves the light.
	fibreCoarseBit = pickedNumber("fibre-coarse-bit")
	// fibreFineBit is how much the fine fibre moves the light.
	fibreFineBit = pickedNumber("fibre-fine-bit")
	// fibreFleckAbove is the level above which the fine fibre is a fleck
	// of dark wool.
	fibreFleckAbove = pickedNumber("fibre-fleck-above")
	// wovenInk is how thick the chevron's ink is, as a share of a row.
	wovenInk = pickedNumber("woven-ink")
	// wovenEdge is how thick the chevron's edge line is.
	wovenEdge = pickedNumber("woven-edge")
)

// clothHash is a number from nought to one that is the same each time for
// the same lattice point.
func clothHash(x, y, z int) float64 {
	bits := uint32(x)*clothHashA ^ uint32(y)*clothHashB ^ uint32(z)*clothHashC
	bits ^= bits >> clothHashShift
	bits *= clothHashMix
	bits ^= bits >> (clothHashShift + pair)
	return float64(bits&clothHashRange) / clothHashRange
}

// lattice is the hash blended along x between a lattice point and the
// next one, by the share of the way from one to the other.
func lattice(x, y, z int, share float64) float64 {
	return clothHash(x, y, z)*(unity-share) + clothHash(x+unity, y, z)*share
}

// clothNoise is smooth value noise at the point, from nought to one, with
// one lattice point to the unit.
func clothNoise(point maths.Vec) float64 {
	x, y, z := math.Floor(point.X), math.Floor(point.Y), math.Floor(point.Z)
	fx, fy, fz := smoothed(point.X-x), smoothed(point.Y-y), smoothed(point.Z-z)
	ix, iy, iz := int(x), int(y), int(z)
	low := lattice(ix, iy, iz, fx)*(unity-fy) + lattice(ix, iy+unity, iz, fx)*fy
	high := lattice(ix, iy, iz+unity, fx)*(unity-fy) +
		lattice(ix, iy+unity, iz+unity, fx)*fy
	return low*(unity-fz) + high*fz
}

// blend is a light between two, from the first at nought to the second at
// one.
func blend(first, second maths.Light, share float64) maths.Light {
	share = math.Max(nought, math.Min(unity, share))
	return first.Scale(unity - share).Add(second.Scale(share))
}

// arc is how far round a vertical axis a point is, in metres at the radius
// it is at, from the axis's own z.
func arc(point maths.Vec) float64 {
	return math.Atan2(point.X, point.Z) * math.Hypot(point.X, point.Z)
}

// Canvas is plain-weave cloth hung round a vertical axis through the
// origin, soiled with mud near the ground and streaked by rain. The weave
// is the threads' pitch, and it is drawn only where a pixel is smaller
// than a thread: further off, the cloth is its own average colour.
type Canvas struct {
	// Cloth is the clean cloth, and Soil what soils it.
	Cloth, Soil maths.Light
	// Pitch is the distance between threads, in metres, and Depth how
	// much darker the weave dips between them, from nought to one.
	Pitch, Depth float64
	// Radius is the radius the threads are counted round, so that the
	// warp runs straight up the cone.
	Radius float64
	// SoilHeight is how high the mud climbs, as the height at which it
	// has fallen to a third, Soiling how dark it is at the ground, and
	// Streaks how much rain has run down the cloth, from nought to one.
	SoilHeight, Soiling, Streaks float64
}

// At is the canvas at the point, with no pixel to size it.
func (canvas Canvas) At(point maths.Vec) maths.Light {
	return canvas.Within(point, nought)
}

// Within is the canvas at the point for a pixel footprint wide.
func (canvas Canvas) Within(point maths.Vec, footprint float64) maths.Light {
	depth := canvas.Depth * math.Max(nought, unity-footprint/canvas.Pitch)
	warp := math.Atan2(point.X, point.Z) * canvas.Radius
	weave := half - half*math.Cos(fullTurn*warp/canvas.Pitch)*
		math.Cos(fullTurn*point.Y/canvas.Pitch)
	high := math.Max(point.Y, nought)
	mud := canvas.Soiling * math.Exp(-high/canvas.SoilHeight)
	streaks := clothNoise(maths.V(warp/clothStreakSize, nought, nought))
	streak := canvas.Streaks * streaks *
		math.Exp(-high/(clothStreakFall*canvas.SoilHeight))
	mottled := clothNoise(point.Scale(unity / clothMottle))
	mottle := clothMottleBit * canvas.Soiling * (mottled - half)
	return blend(canvas.Cloth.Scale(unity-depth*weave), canvas.Soil,
		mud+streak+mottle)
}

// Twist is a rope of two strands laid round each other. The strands run
// round the axis, and wind about the rope's own length once in a Lay.
type Twist struct {
	// A and B are the colours of the two strands.
	A, B maths.Light
	// Pitch is how far along the rope one twist of a strand goes, in
	// metres, and Lay how high up the rope a strand climbs as it winds.
	Pitch, Lay float64
}

// At is the rope's colour at the point.
func (twist Twist) At(point maths.Vec) maths.Light {
	phase := arc(point)/twist.Pitch + point.Y/twist.Lay
	return blend(twist.A, twist.B, twistEdge+twistEdge*math.Cos(fullTurn*phase))
}

// Woven is a band of wool woven in chevrons: a ground, a line of ink that
// zigzags along the band, and a finer line of edge beside it.
type Woven struct {
	// Ground, Ink and Edge are the colours of the three.
	Ground, Ink, Edge maths.Light
	// Cell is how far along the band one zigzag runs, in metres, and
	// Rise how high up it one row of the pattern is.
	Cell, Rise float64
}

// At is the band's colour at the point.
func (woven Woven) At(point maths.Vec) maths.Light {
	along := arc(point) / woven.Cell
	zig := math.Abs(pair*(along-math.Floor(along)) - unity)
	up := point.Y / woven.Rise
	row := up - math.Floor(up)
	switch {
	case math.Abs(row-zig) < wovenInk:
		return woven.Ink
	case math.Abs(row-(unity-zig)) < wovenEdge:
		return woven.Edge
	}
	return woven.Ground
}

// Fibre is felt: wool matted so that its fibres show, a coarse unevenness
// of the mat and a finer one, with the odd fleck of darker wool.
type Fibre struct {
	// Wool is the colour of the felt, and Fleck of its darker wool.
	Wool, Fleck maths.Light
	// Scale is the size of the coarse unevenness, in metres.
	Scale float64
}

// At is the felt at the point.
func (fibre Fibre) At(point maths.Vec) maths.Light {
	coarse := clothNoise(point.Scale(unity / fibre.Scale))
	fine := clothNoise(point.Scale(fibreFine / fibre.Scale))
	moved := unity + fibreCoarseBit*(coarse-half) + fibreFineBit*(fine-half)
	wool := fibre.Wool.Scale(moved)
	if fine > fibreFleckAbove {
		return blend(wool, fibre.Fleck, half)
	}
	return wool
}

// Stained is wood painted thinly: the paint over the grain, with the grain
// showing through by how much the paint does not cover.
type Stained struct {
	// Grain is the wood beneath, and Paint the colour of the paint.
	Grain Texture
	Paint maths.Light
	// Cover is how much of the grain the paint hides, from nought to
	// one.
	Cover float64
}

// At is the painted wood at the point.
func (stained Stained) At(point maths.Vec) maths.Light {
	return blend(stained.Grain.At(point), stained.Paint, stained.Cover)
}

// ornamentInks is defined: the colours of an ornament's rings, red,
// orange and blue.
const ornamentInks = 3

// Ornament is a repeating painted ornament on a surface: lozenges in
// rings of red, orange and blue, on whatever is beneath. Coords says
// where a point is on the surface, in metres, so that one ornament can be
// laid on a door, round a ring or over a rug. A cell or a ring that is not
// over nought has no lozenges, and the surface beneath shows everywhere.
type Ornament struct {
	// Under is what shows between the lozenges.
	Under Texture
	// Inks are the colours of the rings, from the middle out, and they
	// repeat.
	Inks [ornamentInks]maths.Light
	// Cell is how wide one lozenge is, and Ring how wide each ring of
	// it, in metres.
	Cell, Ring float64
	// Coords is where a point is on the surface.
	Coords func(point maths.Vec) (across, up float64)
}

// At is the ornament at the point.
func (ornament Ornament) At(point maths.Vec) maths.Light {
	if !(ornament.Cell > nought) || !(ornament.Ring > nought) {
		return ornament.Under.At(point)
	}
	across, up := ornament.Coords(point)
	x := across/ornament.Cell - math.Floor(across/ornament.Cell) - half
	y := up/ornament.Cell - math.Floor(up/ornament.Cell) - half
	reach := (math.Abs(x) + math.Abs(y)) * ornament.Cell
	if reach > diamondInside*ornament.Cell {
		return ornament.Under.At(point)
	}
	return ornament.Inks[int(reach/ornament.Ring)%len(ornament.Inks)]
}
