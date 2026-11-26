package whales

import (
	"github.com/janearc/slayer/manufacture/geometry"
	"math"
	"sync"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals"
	"github.com/janearc/slayer/maths"
)

// The mouth, the teeth, the blowhole, the eyes and the belly slits.
// Every number is chosen, in metres in rest space, from the proportions
// of a six metre cow; none is a measurement of an animal.
const (
	orcaMouthFront  = 3.03 // x where the mouth line meets the snout
	orcaMouthCorner = 2.12 // x of the corner of the gape, under the eye
	orcaMouthLow    = .33  // how far below the head's centre the line runs
	orcaMouthLift   = .05  // how far the line turns up at the corner
	orcaMouthWidth  = .02  // half width of the lip groove
	orcaMouthDepth  = .03  // depth of the lip groove
	orcaTeethFirst  = 2.22 // x of the rearmost tooth
	orcaTeethStep   = .065 // spacing of teeth along the jaw
	orcaTeethCount  = 12   // teeth a side, upper and lower in turn: 6 a jaw
	orcaToothRadius = .014 // radius of a tooth at its base
	orcaToothHeight = .034 // height of a tooth
	orcaToothRoot   = .018 // distance of a tooth's base from the mouth line
	orcaToothSink   = .012 // how far a tooth stands inside the skin surface
)

// orcaMouthLine is the height of the mouth line at x.
func (model *orcaModel) orcaMouthLine(x float64) float64 {
	s := model.orcaTrunkProfile().at(x)
	return s.centre - orcaMouthLow*s.high +
		orcaMouthLift*orcaStep(2.5, orcaMouthCorner, x)
}

// orcaMouth is how deep the lip groove cuts at p: zero away from the
// mouth line, and fading out at the corner and at the snout.
func (model *orcaModel) orcaMouth(p maths.Vec) float64 {
	if p.X < orcaMouthCorner-.1 || p.X > orcaMouthFront+.05 {
		return 0
	}
	s := model.orcaTrunkProfile().at(p.X)
	side := orcaStep(.45, .75, math.Abs(p.Z)/s.wide)
	ends := orcaStep(orcaMouthCorner-.05, orcaMouthCorner+.05, p.X) *
		orcaStep(orcaMouthFront+.01, orcaMouthFront-.05, p.X)
	gap := math.Abs(p.Y - model.orcaMouthLine(p.X))
	return orcaMouthDepth * side * ends * orcaSlit(gap, orcaMouthWidth)
}

// orcaTeeth is the distance from p to the nearest tooth: a cone set in
// the lip groove, the teeth of the upper jaw pointing down and those of
// the lower pointing up in turn, so each jaw a side has half of
// orcaTeethCount, six. That is fewer than the 10 to 12 of S3 in
// doc/model/bioform/orca/SOURCES.md; the bone's rows have twelve a
// jaw (orcaToothRows). The skin's count is kept as drawn.
func (model *orcaModel) orcaTeeth(p maths.Vec) float64 {
	last := orcaTeethFirst + orcaTeethStep*(orcaTeethCount-1)
	if p.X < orcaTeethFirst-.04 || p.X > last+.04 {
		return orcaFar
	}
	s := model.orcaTrunkProfile().at(p.X)
	if math.Abs(p.Z)/s.wide < .6 {
		return orcaFar
	}
	index := math.Round((p.X - orcaTeethFirst) / orcaTeethStep)
	index = math.Min(math.Max(index, 0), orcaTeethCount-1)
	across := orcaTeethFirst + index*orcaTeethStep
	line := model.orcaMouthLine(across)
	sideways := math.Sqrt(math.Max(0, 1-(line-s.centre)*(line-s.centre)/
		(s.high*s.high))) * s.wide
	direction := 1.
	if int(index)%2 == 0 {
		direction = -1
	}
	base := line - direction*orcaToothRoot
	along := direction * (p.Y - base)
	radius := orcaToothRadius * (1 - along/orcaToothHeight)
	radial := math.Hypot(p.X-across, math.Abs(p.Z)-(sideways-orcaToothSink))
	return math.Max(radial-radius, math.Max(-along, along-orcaToothHeight))
}

// The blowhole: a crescent slit on the midline of the head's top, its
// horns forward, an opening whose inside is painted dark. It lies over
// the back of the braincase, where the nasal passage rises from the
// bony nares at the skull's vertex and leans back; the place is chosen
// from the skull of orca_anatomy.go.
const (
	orcaBlowX     = 1.92 // x of the slit's middle
	orcaBlowHalf  = .10  // half the slit's length across the head
	orcaBlowHorn  = .04  // how far the horns lie ahead of the middle
	orcaBlowWidth = .014 // half width of the slit, a little over a lattice step
	orcaBlowDepth = .05  // depth of the slit
)

// orcaBlowhole is how deep the blowhole slit cuts at p.
func (model *orcaModel) orcaBlowhole(p maths.Vec) float64 {
	across := math.Abs(p.Z)
	if p.X < orcaBlowX-.05 || p.X > orcaBlowX+.1 || across > orcaBlowHalf+.01 {
		return 0
	}
	s := model.orcaTrunkProfile().at(p.X)
	top := orcaStep(s.centre+s.high-.10, s.centre+s.high-.05, p.Y)
	ends := orcaStep(orcaBlowHalf+.01, orcaBlowHalf-.03, across)
	bend := across / orcaBlowHalf
	gap := math.Abs(p.X - (orcaBlowX + orcaBlowHorn*bend*bend))
	return orcaBlowDepth * top * ends * orcaSlit(gap, orcaBlowWidth)
}

// The eye: small, low on the head, a little ahead of the eye patch.
const (
	orcaEyeX      = 2.07 // x of the eye
	orcaEyeY      = -.03 // height of the eye
	orcaEyeRadius = .034 // radius of the hollow the eye sits in
	orcaEyeDepth  = .018 // depth of that hollow
)

// orcaEyeCentre is where the eye lies on the skin, on the right side;
// the left is the mirror.
func (model *orcaModel) orcaEyeCentre() maths.Vec {
	s := model.orcaTrunkProfile().at(orcaEyeX)
	rise := (orcaEyeY - s.centre) / s.high
	return maths.V(orcaEyeX, orcaEyeY, s.wide*math.Sqrt(1-rise*rise))
}

// orcaEye is how deep the hollow round the eye cuts at p.
func (model *orcaModel) orcaEye(p maths.Vec) float64 {
	centre := model.orcaEyeCentre()
	gap := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(centre).Len()
	if gap > orcaEyeRadius {
		return 0
	}
	t := 1 - (gap/orcaEyeRadius)*(gap/orcaEyeRadius)
	return orcaEyeDepth * t * t
}

// The belly slits of a cow: the genital slit on the midline with a
// mammary slit each side of the end of it, and the anus behind.
const (
	orcaSlitWidth = .018 // half width of each ventral slit
	orcaSlitDepth = .016 // depth of each ventral slit
)

// orcaVentral is how deep the belly slits and the anal dimple cut at p.
func (model *orcaModel) orcaVentral(p maths.Vec) float64 {
	if p.X < -.95 || p.X > -.1 || math.Abs(p.Z) > .15 {
		return 0
	}
	s := model.orcaTrunkProfile().at(p.X)
	belly := orcaStep(.80, .92, (s.centre-p.Y)/s.high)
	if belly == 0 {
		return 0
	}
	across := math.Abs(p.Z)
	genital := orcaSlit(across, orcaSlitWidth) *
		orcaStep(-.52, -.47, p.X) * orcaStep(-.18, -.23, p.X)
	mammary := orcaSlit(math.Abs(across-.07), orcaSlitWidth) *
		orcaStep(-.77, -.73, p.X) * orcaStep(-.58, -.62, p.X)
	anus := orcaSlit(math.Hypot(p.X+.85, across), 2*orcaSlitWidth)
	return orcaSlitDepth * belly * math.Max(genital, math.Max(mammary, anus))
}

// orcaCarve is how far everything cut into the skin moves the surface
// inward at p, in metres.
func (model *orcaModel) orcaCarve(p maths.Vec) float64 {
	return model.orcaMouth(p) + model.orcaBlowhole(p) + model.orcaEye(p) + model.orcaVentral(p)
}

// orcaWorkers is how many goroutines sample the field at once. It is
// chosen to leave the machine's other work its share.
const orcaWorkers = 4

// orcaLattice is the regular grid of sample points the field is read on:
// its lowest corner, the spacing, and the points along each axis.
type orcaLattice struct {
	low        maths.Vec
	step       float64
	nx, ny, nz int
}

// orcaCorner lists a cell's eight corners as offsets, numbered so that
// bit zero is x, bit one is y and bit two is z.
var orcaCorner = [8][3]int{
	{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0},
	{0, 0, 1}, {1, 0, 1}, {0, 1, 1}, {1, 1, 1},
}

// orcaCellEdge lists a cell's twelve edges as pairs of corners.
var orcaCellEdge = [12][2]int{
	{0, 1}, {2, 3}, {4, 5}, {6, 7},
	{0, 2}, {1, 3}, {4, 6}, {5, 7},
	{0, 4}, {1, 5}, {2, 6}, {3, 7},
}

// index is the place of lattice point (i, j, k) in the sample array.
func (l orcaLattice) index(i, j, k int) int { return (k*l.ny+j)*l.nx + i }

// cell is the place of cell (i, j, k), the cell whose lowest corner is
// that lattice point, in the array of cell vertices.
func (l orcaLattice) cell(i, j, k int) int {
	return (k*(l.ny-1)+j)*(l.nx-1) + i
}

// sample reads the field at every lattice point, in parallel by layers.
func (l orcaLattice) sample(field func(maths.Vec) float64) []float32 {
	values := make([]float32, l.nx*l.ny*l.nz)
	var group sync.WaitGroup
	for worker := 0; worker < orcaWorkers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for k := worker; k < l.nz; k += orcaWorkers {
				l.fill(values, field, k)
			}
		}()
	}
	group.Wait()
	return values
}

// fill reads the field over layer k of the lattice.
func (l orcaLattice) fill(values []float32, field func(maths.Vec) float64,
	k int,
) {
	for j := 0; j < l.ny; j++ {
		for i := 0; i < l.nx; i++ {
			at := l.low.Add(maths.V(float64(i), float64(j), float64(k)).
				Scale(l.step))
			values[l.index(i, j, k)] = float32(field(at))
		}
	}
}

// vertex places the one surface vertex of cell (i, j, k): the mean of
// where the surface crosses the cell's edges. It reports false when the
// cell holds no surface.
func (l orcaLattice) vertex(values []float32, i, j, k int) (maths.Vec, bool) {
	var corner [8]float64
	inside := 0
	for c, offset := range orcaCorner {
		corner[c] = float64(values[l.index(i+offset[0], j+offset[1], k+offset[2])])
		if corner[c] < 0 {
			inside++
		}
	}
	if inside == 0 || inside == len(corner) {
		return maths.Vec{}, false
	}
	var sum maths.Vec
	crossings := 0.
	for _, edge := range orcaCellEdge {
		a, b := corner[edge[0]], corner[edge[1]]
		if (a < 0) == (b < 0) {
			continue
		}
		t := a / (a - b)
		from, to := orcaCorner[edge[0]], orcaCorner[edge[1]]
		sum = sum.Add(maths.V(
			float64(from[0])+t*float64(to[0]-from[0]),
			float64(from[1])+t*float64(to[1]-from[1]),
			float64(from[2])+t*float64(to[2]-from[2])))
		crossings++
	}
	mean := sum.Scale(1 / crossings)
	return l.low.Add(maths.V(float64(i)+mean.X, float64(j)+mean.Y,
		float64(k)+mean.Z).Scale(l.step)), true
}

// vertices places a vertex in every cell the surface crosses. cells
// names each cell's vertex, or is minus one where there is none.
func (l orcaLattice) vertices(values []float32) ([]maths.Vec, []int32) {
	cells := make([]int32, (l.nx-1)*(l.ny-1)*(l.nz-1))
	var points []maths.Vec
	for k := 0; k < l.nz-1; k++ {
		for j := 0; j < l.ny-1; j++ {
			for i := 0; i < l.nx-1; i++ {
				at, found := l.vertex(values, i, j, k)
				if !found {
					cells[l.cell(i, j, k)] = -1
					continue
				}
				cells[l.cell(i, j, k)] = int32(len(points))
				points = append(points, at)
			}
		}
	}
	return points, cells
}

// orcaQuad turns the four cells round one lattice edge into two triangles,
// split along the shorter diagonal. The cells are given counterclockwise
// about the edge's direction; reverse says the inside lies at the far
// end of the edge, so the order is turned round.
func orcaQuad(points []maths.Vec, ring [4]int32, reverse bool) [][3]int {
	a, b, c, d := int(ring[0]), int(ring[1]), int(ring[2]), int(ring[3])
	if reverse {
		a, b, c, d = d, c, b, a
	}
	across := points[a].Sub(points[c]).Len()
	other := points[b].Sub(points[d]).Len()
	if across <= other {
		return [][3]int{{a, b, c}, {a, c, d}}
	}
	return [][3]int{{a, b, d}, {b, c, d}}
}

// faces joins the cell vertices into triangles: one quad round every
// lattice edge the surface crosses, wound so the outside is the side the
// field is positive on.
func (l orcaLattice) faces(values []float32, cells []int32,
	points []maths.Vec,
) [][3]int {
	faces := make([][3]int, 0, 2*len(points))
	at := func(i, j, k int) int32 { return cells[l.cell(i, j, k)] }
	for k := 1; k < l.nz-1; k++ {
		for j := 1; j < l.ny-1; j++ {
			for i := 1; i < l.nx-1; i++ {
				here := values[l.index(i, j, k)] < 0
				if (values[l.index(i+1, j, k)] < 0) != here {
					faces = append(faces, orcaQuad(points, [4]int32{
						at(i, j-1, k-1), at(i, j, k-1), at(i, j, k), at(i, j-1, k),
					}, !here)...)
				}
				if (values[l.index(i, j+1, k)] < 0) != here {
					faces = append(faces, orcaQuad(points, [4]int32{
						at(i-1, j, k-1), at(i-1, j, k), at(i, j, k), at(i, j, k-1),
					}, !here)...)
				}
				if (values[l.index(i, j, k+1)] < 0) != here {
					faces = append(faces, orcaQuad(points, [4]int32{
						at(i-1, j-1, k), at(i, j-1, k), at(i, j, k), at(i-1, j, k),
					}, !here)...)
				}
			}
		}
	}
	return faces
}

// orcaContour finds the surface where field crosses zero on the lattice,
// as shared points and triangles wound outward.
func orcaContour(field func(maths.Vec) float64,
	l orcaLattice,
) ([]maths.Vec, [][3]int) {
	values := l.sample(field)
	l.joinAll(values)
	points, cells := l.vertices(values)
	return orcaCompact(points, orcaDropTwins(l.faces(values, cells, points)))
}

// orcaJoinDepth is how far inside the surface a sample is put when it is
// joined to the inside, in metres. Chosen: any depth below nought will
// do, and a micrometre moves the surface by less than anything shows.
const orcaJoinDepth = 1e-6

// joinAll joins the lattice's alternating faces, pass after pass, until
// no face alternates. Putting a sample inside can make a face beside it
// alternate; each pass turns samples inside only, so the passes end.
func (l orcaLattice) joinAll(values []float32) {
	for again := true; again; {
		again = l.joined(values) > 0
	}
}

// joined puts inside the surface one sample of every lattice face whose
// corners are inside and outside by turns, as a thin tip leaves them,
// and reports how many faces it did.
//
// Such a face has two inside corners that meet only along its diagonal:
// the surface touches itself there, four quads share one edge between two
// cells, and the twins dropped for it leave a hole. Joining the inside
// across the face lets the surface pass round it once.
func (l orcaLattice) joined(values []float32) int {
	faces := 0
	for k := 0; k < l.nz; k++ {
		for j := 0; j < l.ny; j++ {
			for i := 0; i < l.nx; i++ {
				at := l.index(i, j, k)
				if i+1 < l.nx && j+1 < l.ny {
					faces += orcaJoinFace(values, at, 1, l.nx)
				}
				if i+1 < l.nx && k+1 < l.nz {
					faces += orcaJoinFace(values, at, 1, l.nx*l.ny)
				}
				if j+1 < l.ny && k+1 < l.nz {
					faces += orcaJoinFace(values, at, l.nx, l.nx*l.ny)
				}
			}
		}
	}
	return faces
}

// orcaJoinFace joins the face whose corners lie at sample at, a step
// across from it and a step up from it, if they alternate: it puts inside
// whichever of the two outside corners has the lower field, and reports
// one. Otherwise it changes nothing and reports nought.
func orcaJoinFace(values []float32, at, across, up int) int {
	a, b, c, d := at, at+across, at+up, at+across+up
	if (values[a] < 0) != (values[d] < 0) ||
		(values[b] < 0) != (values[c] < 0) ||
		(values[a] < 0) == (values[b] < 0) {
		return 0
	}
	first, second := a, d
	if values[a] < 0 {
		first, second = b, c
	}
	if values[second] < values[first] {
		first = second
	}
	values[first] = -orcaJoinDepth
	return 1
}

// orcaDropTwins removes every triangle that shares an edge, running the
// same way, with another. Where the surface is thinner than the lattice
// the contour can fold so; the fold is a pinhole, and leaving it would
// make the mesh refuse to build.
func orcaDropTwins(faces [][3]int) [][3]int {
	seen := make(map[[2]int]int, 3*len(faces))
	twin := make([]bool, len(faces))
	for n, f := range faces {
		for corner := 0; corner < 3; corner++ {
			edge := [2]int{f[corner], f[(corner+1)%3]}
			if other, again := seen[edge]; again {
				twin[n], twin[other] = true, true
				continue
			}
			seen[edge] = n
		}
	}
	kept := make([][3]int, 0, len(faces))
	for n, f := range faces {
		if !twin[n] {
			kept = append(kept, f)
		}
	}
	return kept
}

// orcaCompact renumbers the points so that only those a triangle uses
// remain, and returns them with the triangles renumbered to match.
func orcaCompact(points []maths.Vec, faces [][3]int) ([]maths.Vec, [][3]int) {
	renumber := make([]int, len(points))
	for i := range renumber {
		renumber[i] = -1
	}
	var used []maths.Vec
	for n, f := range faces {
		for corner, v := range f {
			if renumber[v] < 0 {
				renumber[v] = len(used)
				used = append(used, points[v])
			}
			faces[n][corner] = renumber[v]
		}
	}
	return used, faces
}

// The markings, chosen, in metres in rest space and in the trunk's own
// coordinates. The belly's edge is hard, as the animal's is; the saddle
// patch's is soft.
const (
	orcaBellyEdge   = -.34  // height of the belly's edge, in half heights
	orcaBellyLobe   = .62   // how far the white lobe climbs the flank
	orcaLobeStart   = .05   // x where the lobe begins to climb
	orcaLobePeak    = -1.05 // x where it has climbed fully
	orcaLobeEnd     = -1.28 // x where it ends, sharply
	orcaHardEdge    = .02   // half width of a hard edge, in half heights
	orcaPatchX      = 1.82  // x of the middle of the eye patch
	orcaPatchY      = .12   // height of the middle of the eye patch
	orcaPatchLong   = .21   // half the patch's length
	orcaPatchShort  = .065  // half the patch's breadth
	orcaPatchTilt   = .26   // tilt of the patch, rear end up, in radians
	orcaSaddleFront = .45   // x where the saddle patch begins, behind the fin
	orcaSaddleRear  = -.95  // x where the saddle patch has faded away
	orcaMarkPigment = .45   // share of the pigment a rake or scar has lost
)

// orcaMix is a blended with b, none of b at zero and all of it at one.
func orcaMix(a, b maths.Light, t float64) maths.Light {
	return maths.RGB(a.R+(b.R-a.R)*t, a.G+(b.G-a.G)*t, a.B+(b.B-a.B)*t)
}

// orca05Paint is the animal's colour at a point of its rest shape: the
// markings follow the body by where they lie on it, not by where it is.
type orca05Paint struct {
	model   *orcaModel
	palette Palette
}

// At is the colour at rest point p.
func (paint orca05Paint) At(p maths.Vec) maths.Light {
	model, palette := paint.model, paint.palette
	colour := palette.Black
	if model.orcaTrunk(p) < .03 {
		colour = orcaMix(colour, palette.White, model.orcaBelly(p))
		colour = orcaMix(colour, palette.Saddle, model.orcaSaddlePatch(p))
		colour = orcaMix(colour, palette.White, model.orcaEyePatch(p))
	} else {
		colour = orcaMix(colour, palette.White, orcaFlukeUnderside(p))
	}
	marks := math.Max(model.orcaRakeMark(p), model.orcaScarMark(p))
	colour = orcaMix(colour, palette.Pale, orcaMarkPigment*marks)
	colour = orcaMix(colour, palette.Gum, orcaStep(.01, .02, model.orcaMouth(p)))
	colour = orcaMix(colour, palette.Pupil, orcaStep(.01, .025, model.orcaBlowhole(p)))
	colour = orcaMix(colour, palette.Ivory, 1-orcaStep(.002, .006, model.orcaTeeth(p)))
	return model.orcaEyeColour(p, colour, palette)
}

// orcaBelly is how far p lies in the white belly: one on the throat, the
// belly and the lobes that climb the flanks, zero on the black above.
func (model *orcaModel) orcaBelly(p maths.Vec) float64 {
	s := model.orcaTrunkProfile().at(p.X)
	height := (p.Y - s.centre) / s.high
	climb := orcaStep(orcaLobeStart, orcaLobePeak, p.X) *
		orcaStep(orcaLobeEnd, orcaLobePeak, p.X)
	edge := orcaBellyEdge + orcaBellyLobe*climb
	return 1 - orcaStep(edge-orcaHardEdge, edge+orcaHardEdge, height)
}

// orcaSaddlePatch is how far p lies in the grey saddle behind the dorsal
// fin: soft at its lower edge and at its ends.
func (model *orcaModel) orcaSaddlePatch(p maths.Vec) float64 {
	s := model.orcaTrunkProfile().at(p.X)
	height := (p.Y - s.centre) / s.high
	// the patch's lower edge climbs from low behind the fin to the back
	edge := .02 + .55*orcaStep(orcaSaddleFront-.2, orcaSaddleRear, p.X) +
		.05*math.Sin(5*p.X+1)
	slant := .6 * (1 - math.Min(height, 1)) // lower down, the front lies further back
	along := orcaStep(orcaSaddleFront-slant, orcaSaddleFront-slant-.25, p.X) *
		orcaStep(orcaSaddleRear, orcaSaddleRear+.35, p.X)
	return along * orcaStep(edge, edge+.3, height)
}

// orcaEyePatch is how far p lies in the white oval above and behind the
// eye, on either flank, with a hard edge.
func (model *orcaModel) orcaEyePatch(p maths.Vec) float64 {
	if model.orcaFlank(p) == 0 {
		return 0
	}
	dx, dy := p.X-orcaPatchX, p.Y-orcaPatchY
	cosine, sine := math.Cos(orcaPatchTilt), math.Sin(orcaPatchTilt)
	along := (-dx*cosine + dy*sine) / orcaPatchLong
	across := (dx*sine + dy*cosine) / orcaPatchShort
	return model.orcaFlank(p) * (1 - orcaStep(.9, 1.0, math.Hypot(along, across)))
}

// orcaFlukeUnderside is how far p lies in the white underside of the
// flukes: white below the plane of the flukes, dark along the trailing
// edge, black above.
func orcaFlukeUnderside(p maths.Vec) float64 {
	if p.X > -2.2 || p.Y > orcaFlukeLevel {
		return 0
	}
	_, trail := orcaFlukeEdges(math.Abs(p.Z))
	return orcaStep(trail+.02, trail+.07, p.X)
}

// orcaEyeColour is colour, except inside the eye, where it is the iris
// and the pupil.
func (model *orcaModel) orcaEyeColour(p maths.Vec, colour maths.Light, palette Palette) maths.Light {
	gap := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(model.orcaEyeCentre()).Len()
	colour = orcaMix(colour, palette.Iris, 1-orcaStep(.024, .028, gap))
	return orcaMix(colour, palette.Pupil, 1-orcaStep(.012, .016, gap))
}

// orcaFar is the distance a part reports for a point beyond its reach.
const orcaFar = 1.0

// orcaEdgeRadius rounds every fin edge, so no edge is thinner than the
// lattice can hold. It is chosen: about one lattice step.
const orcaEdgeRadius = .016

// orcaEdgeShape is the thickness profile across a fin's chord, zero at
// both edges, blunt at the leading one: u runs from the trailing edge
// (0) to the leading edge (1). Its largest value is one.
func orcaEdgeShape(u float64) float64 {
	const peak = 2.6 // makes the profile's maximum one, at u = 2/3
	return peak * u * math.Sqrt(1-u)
}

// orcaClamp01 limits x to the range zero to one.
func orcaClamp01(x float64) float64 { return math.Min(math.Max(x, 0), 1) }

// orcaStep is zero at from, one at to and eased between, in either
// order of the two.
func orcaStep(from, to, x float64) float64 {
	t := orcaClamp01((x - from) / (to - from))
	return t * t * (3 - 2*t)
}

// orcaSmoothMin is the smaller of a and b with the corner between them
// rounded over a width of k: this is what fillets a fin into the body.
func orcaSmoothMin(a, b, k float64) float64 {
	h := math.Max(k-math.Abs(a-b), 0) / k
	return math.Min(a, b) - h*h*k/4
}

// orcaRounded combines a distance across a part's outline and a distance
// across its thickness into one, with every edge rounded by radius.
func orcaRounded(outline, thickness, radius float64) float64 {
	a, b := outline+radius, thickness+radius
	return math.Hypot(math.Max(a, 0), math.Max(b, 0)) +
		math.Min(math.Max(a, b), 0) - radius
}

// orcaSlit is a groove's profile: one on its line, falling to zero at
// width away.
func orcaSlit(gap, width float64) float64 {
	t := math.Max(0, 1-(gap/width)*(gap/width))
	return t * t
}

// The dorsal fin of a cow: falcate, with the tip swept back over the
// trailing base. These numbers were chosen, in metres in rest space.
const (
	orcaFinFoot  = .40  // height where the profile begins, inside the back
	orcaFinTip   = 1.62 // height of the tip, a little under a metre clear
	orcaFinLead  = 1.05 // x of the leading edge at the foot
	orcaFinRake  = .80  // how far back the leading edge runs to the tip
	orcaFinCurve = 1.7  // power of the leading edge's sweep
	orcaFinTrail = .30  // x of the trailing edge at the foot
	orcaFinBow   = .22  // how far forward the trailing edge is hollowed
	orcaFinHalf  = .07  // half thickness at the foot
	orcaFinTaper = .6   // share of that thickness lost by the tip
	orcaFinChord = .06  // chord of the tip, which is blunt, not a cusp
)

// orcaDorsal is the distance from p to the dorsal fin: a flat profile
// with a blunt leading edge and a fine trailing one.
func orcaDorsal(p maths.Vec) float64 {
	if p.X < -.1 || p.X > 1.3 || p.Y < .3 || p.Y > orcaFinTip+.1 ||
		math.Abs(p.Z) > .2 {
		return orcaFar
	}
	span := orcaFinTip - orcaFinFoot
	t := orcaClamp01((p.Y - orcaFinFoot) / span)
	tipX := orcaFinLead - orcaFinRake - orcaFinChord
	lead := orcaFinLead - orcaFinRake*math.Pow(t, orcaFinCurve)
	trail := orcaFinTrail - (orcaFinTrail-tipX)*t +
		orcaFinBow*math.Sin(math.Pi*t)
	leadSlope := orcaFinRake * orcaFinCurve * math.Pow(t, orcaFinCurve-1) / span
	trailSlope := ((tipX - orcaFinTrail) +
		orcaFinBow*math.Pi*math.Cos(math.Pi*t)) / span
	outline := math.Max((p.X-lead)/math.Hypot(1, leadSlope),
		(trail-p.X)/math.Hypot(1, trailSlope))
	outline = math.Max(outline, p.Y-orcaFinTip)
	u := 0.
	if lead > trail {
		u = orcaClamp01((p.X - trail) / (lead - trail))
	}
	half := orcaFinHalf * (1 - orcaFinTaper*t) * orcaEdgeShape(u)
	return orcaRounded(outline, math.Abs(p.Z)-half, orcaEdgeRadius)
}

// The pectoral flipper: a paddle, broad and rounded, set low on the
// flank and directed down, out and a little back. Chosen, in metres.
const (
	orcaFlipperRootX = 1.55 // x of the root, buried in the flank
	orcaFlipperRootY = -.22 // height of the root
	orcaFlipperRootZ = .30  // distance of the root from the midline
	orcaFlipperAt    = .29  // where along the span the paddle's middle lies
	orcaFlipperReach = .68  // half the paddle's length along the span
	orcaFlipperChord = .27  // half the paddle's greatest width
	orcaFlipperSweep = .10  // how far the paddle's middle trails per metre
	orcaFlipperHalf  = .055 // half thickness at the middle
)

var (
	// orcaFlipperSpan runs from the root to the tip.
	orcaFlipperSpan = maths.V(-.45, -.40, 1).Unit()
	// orcaFlipperChordway lies across the paddle, forward.
	orcaFlipperChordway = maths.V(1, 0, 0).Sub(
		orcaFlipperSpan.Scale(orcaFlipperSpan.X)).Unit()
	// orcaFlipperNormal stands through the paddle's thickness. Its X
	// component nearly cancels, so choose the fused operation explicitly:
	// compiler-dependent fusion changes the flipper's skin binding.
	orcaFlipperNormal = func() maths.Vec {
		normal := orcaFlipperSpan.Cross(orcaFlipperChordway)
		normal.X = math.FMA(-orcaFlipperSpan.Z, orcaFlipperChordway.Y,
			orcaFlipperSpan.Y*orcaFlipperChordway.Z)
		return normal.Unit()
	}()
)

// orcaFlipper is the distance from p to the pair of flippers; the left
// is the mirror of the right.
func orcaFlipper(p maths.Vec) float64 {
	q := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(
		maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ))
	along := q.Dot(orcaFlipperSpan)
	across := q.Dot(orcaFlipperChordway)
	through := q.Dot(orcaFlipperNormal)
	if along < -.5 || along > 1.1 || math.Abs(across) > .45 ||
		math.Abs(through) > .2 {
		return orcaFar
	}
	a := (along - orcaFlipperAt) / orcaFlipperReach
	b := (across + orcaFlipperSweep*along) / orcaFlipperChord
	rim := a*a + b*b
	outline := (math.Sqrt(rim) - 1) * orcaFlipperChord
	half := orcaFlipperHalf * math.Sqrt(math.Max(0, 1-rim))
	return orcaRounded(outline, math.Abs(through)-half, orcaEdgeRadius)
}

// The flukes: swept back to fine tips, a median notch, thickest at the
// root. Chosen, in metres in rest space.
const (
	orcaFlukeLead  = -2.30 // x of the leading edge at the midline
	orcaFlukeRake  = .78   // how far back the leading edge runs to the tip
	orcaFlukePower = 1.35  // power of the leading edge's sweep
	orcaFlukeTrail = -2.72 // x of the trailing edge at the midline, un-notched
	orcaFlukeDrop  = .41   // how far back the trailing edge runs to the tip
	orcaFlukeSpan  = 1.0   // half the span, tip to midline
	orcaFlukeNotch = .11   // depth of the median notch
	orcaFlukeCleft = .16   // half width of the notch at the trailing edge
	orcaFlukeHalf  = .06   // half thickness at the root
	orcaFlukeTaper = .5    // share of that thickness lost by the tip
	orcaFlukeLevel = -.01  // height of the flukes' plane
)

// orcaFlukeEdges is the x of the flukes' leading and trailing edges at
// distance sigma from the midline.
func orcaFlukeEdges(sigma float64) (lead, trail float64) {
	reach := math.Min(sigma, orcaFlukeSpan)
	lead = orcaFlukeLead - orcaFlukeRake*math.Pow(reach, orcaFlukePower)
	cleft := math.Max(0, 1-sigma/orcaFlukeCleft)
	trail = orcaFlukeTrail - orcaFlukeDrop*reach + orcaFlukeNotch*cleft*cleft
	return lead, trail
}

// orcaFlukes is the distance from p to the flukes, which lie in one
// horizontal plane and are the same each side of the midline.
func orcaFlukes(p maths.Vec) float64 {
	sigma := math.Abs(p.Z)
	if p.X > -2.2 || p.X < -3.2 || sigma > orcaFlukeSpan+.1 ||
		math.Abs(p.Y-orcaFlukeLevel) > .2 {
		return orcaFar
	}
	reach := math.Min(sigma, orcaFlukeSpan)
	lead, trail := orcaFlukeEdges(sigma)
	leadSlope := orcaFlukeRake * orcaFlukePower * math.Pow(reach, orcaFlukePower-1)
	outline := math.Max((p.X-lead)/math.Hypot(1, leadSlope),
		(trail-p.X)/math.Hypot(1, orcaFlukeDrop))
	outline = math.Max(outline, sigma-orcaFlukeSpan)
	u := 0.
	if lead > trail {
		u = orcaClamp01((p.X - trail) / (lead - trail))
	}
	half := orcaFlukeHalf * (1 - orcaFlukeTaper*reach) * orcaEdgeShape(u)
	level := math.Abs(p.Y-orcaFlukeLevel) - half
	return orcaRounded(outline, level, orcaEdgeRadius)
}

// The keels along the tail stock, a ridge above and another below.
const (
	orcaKeelHeight = .06   // how far the ridge stands clear of the skin
	orcaKeelHalf   = .022  // half its thickness
	orcaKeelStart  = -1.70 // x where the ridge begins to rise
	orcaKeelFull   = -2.10 // x where it has risen fully
	orcaKeelEnd    = -2.62 // x where the trunk ends
)

// orcaKeel is the distance from p to the dorsal keel (side one) or the
// ventral keel (side minus one), each a thin ridge on the tail stock.
func (model *orcaModel) orcaKeel(p maths.Vec, side float64) float64 {
	if p.X < orcaKeelEnd || p.X > orcaKeelStart || math.Abs(p.Z) > .1 {
		return orcaFar
	}
	s := model.orcaTrunkProfile().at(p.X)
	rise := orcaKeelHeight * orcaStep(orcaKeelStart, orcaKeelFull, p.X)
	depth := side * (p.Y - s.centre)
	outline := math.Max(depth-(s.high+rise), (s.high-.05)-depth)
	return orcaRounded(outline, math.Abs(p.Z)-orcaKeelHalf, orcaEdgeRadius)
}

// orcaField is the signed distance, near the surface, from p to the
// whole animal in rest space: the trunk, the fins and flukes joined to
// it with filleted roots, the grooves cut into it, and the teeth.
func (model *orcaModel) orcaField(p maths.Vec) float64 {
	const (
		rootBlend = .12 // fillet width at the dorsal fin and flukes
		limbBlend = .10 // fillet width at the flipper
		keelBlend = .05 // fillet width at the keels
	)
	d := model.orcaTrunk(p)
	d = orcaSmoothMin(d, orcaDorsal(p), rootBlend)
	d = orcaSmoothMin(d, orcaFlipper(p), limbBlend)
	d = orcaSmoothMin(d, orcaFlukes(p), rootBlend)
	d = orcaSmoothMin(d, model.orcaKeel(p, 1), keelBlend)
	d = orcaSmoothMin(d, model.orcaKeel(p, -1), keelBlend)
	d += model.orcaCarve(p)
	return math.Min(d, model.orcaTeeth(p))
}

// The lattice the rest shape is cut on, in metres. The spacing is
// chosen: finer than a lip groove's width, coarse enough that the field
// is read in seconds.
const (
	orcaLatticeStep = .0125
	orcaLatticeLowX = -3.20
	orcaLatticeLowY = -.75
	orcaLatticeLowZ = -1.12
	orcaLatticeNX   = 513 // points along the body: 6.4 m
	orcaLatticeNY   = 199 // points from belly to the tip of the fin
	orcaLatticeNZ   = 181 // points across the flukes
)

// orcaRestSkin is the animal's rest shape: the surface points as the
// field gives them, the same points with the skin's relief applied, and
// the triangles joining them.
type orcaRestSkin struct {
	points, relieved []maths.Vec
	faces            [][3]int
}

// orcaRest returns the rest shape, cut from the field on first use and
// shared afterwards.
func (model *orcaModel) orcaRest() *orcaRestSkin {
	model.restOnce.Do(func() {
		lattice := orcaLattice{
			low:  maths.V(orcaLatticeLowX, orcaLatticeLowY, orcaLatticeLowZ),
			step: orcaLatticeStep,
			nx:   orcaLatticeNX, ny: orcaLatticeNY, nz: orcaLatticeNZ,
		}
		points, faces := orcaContour(model.orcaField, lattice)
		normals := geometry.UnitNormals(points, faces)
		relieved := make([]maths.Vec, len(points))
		for i, p := range points {
			relieved[i] = p.Add(normals[i].Scale(model.orcaRelief(p)))
		}
		model.rest = &orcaRestSkin{points: points, relieved: relieved,
			faces: faces}
	})
	return model.rest
}

// The dermal ridges: faint lines running the length of the body with
// the flow of water over it. The count is chosen; the spacing it gives
// is wider than a real animal's, so that the mesh can carry it.
const (
	orcaRidgeCount  = 64    // ridges round the trunk
	orcaRidgeHeight = .0012 // height of a ridge, in metres
	orcaRidgeWander = 3.    // how far the lines wander, in radians of phase
)

// orcaRidge is the height of the dermal ridges at p, in metres, fading
// out where the trunk is too slender for them to be a mesh apart.
func (model *orcaModel) orcaRidge(p maths.Vec) float64 {
	s := model.orcaTrunkProfile().at(p.X)
	angle := math.Atan2(p.Z, p.Y-s.centre)
	girth := math.Sqrt(s.high * s.wide)
	phase := orcaRidgeCount*angle + orcaRidgeWander*math.Sin(2.1*p.X+3*angle)
	patchy := .6 + .4*math.Sin(1.3*p.X+2*angle)
	return orcaRidgeHeight * patchy * orcaStep(.30, .45, girth) * math.Sin(phase)
}

// orcaRake is a set of parallel scratches left by another animal's
// teeth, on one flank, running from one end to the other.
type orcaRake struct {
	side         float64
	fromX, fromY float64
	toX, toY     float64
	lines        int
}

// orcaRakes are the rake marks: chosen places, on both flanks.
var orcaRakes = []orcaRake{
	{side: 1, fromX: .55, fromY: -.10, toX: 1.00, toY: .14, lines: 4},
	{side: -1, fromX: 1.15, fromY: .22, toX: 1.55, toY: .04, lines: 3},
	{side: 1, fromX: 1.30, fromY: .30, toX: 1.62, toY: .38, lines: 3},
	{side: -1, fromX: -.85, fromY: .05, toX: -.40, toY: -.10, lines: 4},
	{side: 1, fromX: -1.30, fromY: .12, toX: -.95, toY: .02, lines: 3},
}

// orcaScar is a healed wound on one flank, an oval of shallow scar.
type orcaScar struct {
	side         float64
	atX, atY     float64
	halfX, halfY float64
}

// orcaScars are the scars: chosen places, on both flanks.
var orcaScars = []orcaScar{
	{side: 1, atX: 1.15, atY: .0, halfX: .10, halfY: .035},
	{side: -1, atX: .25, atY: -.05, halfX: .12, halfY: .040},
	{side: 1, atX: -1.20, atY: .22, halfX: .08, halfY: .030},
	{side: -1, atX: 1.80, atY: .30, halfX: .07, halfY: .025},
}

const (
	orcaRakeGap   = .034 // distance between the scratches of a rake
	orcaRakeWidth = .011 // half width of a scratch
	orcaRakeDepth = .005 // depth of a scratch
	orcaScarDepth = .006 // depth of a scar at its middle
)

// orcaFlank is how far p lies on the side of the body, rather than on
// the back or the belly: one on the flank, zero off it.
func (model *orcaModel) orcaFlank(p maths.Vec) float64 {
	s := model.orcaTrunkProfile().at(p.X)
	return orcaStep(.55, .80, math.Abs(p.Z)/s.wide)
}

// orcaSegmentGap is the distance in the xy plane from (x, y) to the
// segment from (ax, ay) to (bx, by), and how far along it the nearest
// point lies, from zero to one.
func orcaSegmentGap(x, y, ax, ay, bx, by float64) (gap, along float64) {
	dx, dy := bx-ax, by-ay
	along = orcaClamp01(((x-ax)*dx + (y-ay)*dy) / (dx*dx + dy*dy))
	return math.Hypot(x-(ax+along*dx), y-(ay+along*dy)), along
}

// orcaScratchTaper is how a scratch fades toward its ends: one in the
// middle, falling to zero at each end, where a tooth lifts or lands.
func orcaScratchTaper(along float64) float64 {
	return math.Sqrt(math.Sin(math.Pi * along))
}

// orcaRakeMark is how strongly the rake marks touch p: one on a
// scratch's line, falling to zero at the scratch's edge.
func (model *orcaModel) orcaRakeMark(p maths.Vec) float64 {
	mark := 0.
	for _, rake := range orcaRakes {
		if rake.side*p.Z <= 0 {
			continue
		}
		long := math.Hypot(rake.toX-rake.fromX, rake.toY-rake.fromY)
		nx, ny := -(rake.toY-rake.fromY)/long, (rake.toX-rake.fromX)/long
		for line := 0; line < rake.lines; line++ {
			offset := (float64(line) - float64(rake.lines-1)/2) * orcaRakeGap
			// each scratch starts and ends a little differently
			shorten := .07 * float64((line*5)%3)
			gap, along := orcaSegmentGap(p.X, p.Y,
				rake.fromX+nx*offset+shorten*(rake.toX-rake.fromX),
				rake.fromY+ny*offset+shorten*(rake.toY-rake.fromY),
				rake.toX+nx*offset-shorten*(rake.toX-rake.fromX),
				rake.toY+ny*offset-shorten*(rake.toY-rake.fromY))
			mark = math.Max(mark,
				orcaSlit(gap, orcaRakeWidth)*orcaScratchTaper(along))
		}
	}
	return mark * model.orcaFlank(p)
}

// orcaScarMark is how strongly the scars touch p: one at a scar's
// middle, falling to zero at its edge.
func (model *orcaModel) orcaScarMark(p maths.Vec) float64 {
	mark := 0.
	for _, scar := range orcaScars {
		if scar.side*p.Z <= 0 {
			continue
		}
		x, y := (p.X-scar.atX)/scar.halfX, (p.Y-scar.atY)/scar.halfY
		mark = math.Max(mark, orcaSlit(math.Hypot(x, y), 1))
	}
	return mark * model.orcaFlank(p)
}

// orcaRelief is how far the skin at rest point p stands off the smooth
// shape, along the outward normal, in metres: ridges up and down, rake
// scratches and scars sunk. It is zero away from the trunk, so the thin
// edges of the fins are left as they are.
func (model *orcaModel) orcaRelief(p maths.Vec) float64 {
	onTrunk := 1 - orcaStep(0, .03, model.orcaTrunk(p))
	if onTrunk == 0 {
		return 0
	}
	height := model.orcaRidge(p) - orcaRakeDepth*model.orcaRakeMark(p) -
		orcaScarDepth*model.orcaScarMark(p)
	return onTrunk * height
}

// orcaStation is one cross-section of the trunk in rest space: where it
// lies along the body, the half height and half width of its ellipse,
// and the height of the ellipse's centre.
type orcaStation struct{ x, high, wide, centre float64 }

// orcaTrunkSamples is how many entries the interpolated profile holds.
const orcaTrunkSamples = 4096

// orcaTrunkTable is the stations interpolated onto an even spacing in x,
// so that a lookup is one step and one blend.
type orcaTrunkTable struct {
	first, last, step float64
	entries           []orcaStation
}

// orcaTrunkProfile returns the interpolated profile, built on first use.
func (model *orcaModel) orcaTrunkProfile() *orcaTrunkTable {
	model.trunkOnce.Do(func() {
		first := model.stations[0].x
		last := model.stations[len(model.stations)-1].x
		table := &orcaTrunkTable{first: first, last: last,
			step: (last - first) / (orcaTrunkSamples - 1)}
		for i := 0; i < orcaTrunkSamples; i++ {
			table.entries = append(table.entries,
				model.orcaStationAt(first+float64(i)*table.step))
		}
		model.trunk = table
	})
	return model.trunk
}

// orcaSlope is the finite-difference slope of one station quantity at
// station index i, taken across its neighbours.
func (model *orcaModel) orcaSlope(i int, pick func(orcaStation) float64) float64 {
	low, high := max(i-1, 0), min(i+1, len(model.stations)-1)
	return (pick(model.stations[high]) - pick(model.stations[low])) /
		(model.stations[high].x - model.stations[low].x)
}

// orcaBlend is one station quantity interpolated at x.
func (model *orcaModel) orcaBlend(x float64, i int, pick func(orcaStation) float64) float64 {
	a, b := model.stations[i], model.stations[i+1]
	return mammals.Hermite(a.x, b.x, pick(a), pick(b),
		model.orcaSlope(i, pick), model.orcaSlope(i+1, pick), x)
}

// orcaStationAt is the smooth cross-section at x, from the cubic through
// the listed stations.
func (model *orcaModel) orcaStationAt(x float64) orcaStation {
	i := 0
	for i < len(model.stations)-2 && x > model.stations[i+1].x {
		i++
	}
	return orcaStation{
		x:      x,
		high:   model.orcaBlend(x, i, func(s orcaStation) float64 { return s.high }),
		wide:   model.orcaBlend(x, i, func(s orcaStation) float64 { return s.wide }),
		centre: model.orcaBlend(x, i, func(s orcaStation) float64 { return s.centre }),
	}
}

// at returns the cross-section at x, clamped to the ends of the trunk.
func (table *orcaTrunkTable) at(x float64) orcaStation {
	f := (x - table.first) / table.step
	i := int(math.Floor(f))
	i = max(0, min(i, len(table.entries)-2))
	t := math.Min(math.Max(f-float64(i), 0), 1)
	a, b := table.entries[i], table.entries[i+1]
	return orcaStation{
		x:      x,
		high:   a.high + t*(b.high-a.high),
		wide:   a.wide + t*(b.wide-a.wide),
		centre: a.centre + t*(b.centre-a.centre),
	}
}

// orcaTrunk is the signed distance, near the surface, from p to the
// trunk: negative inside. Each cross-section is an ellipse and the
// distance to it is the first-order estimate, the ellipse's own
// function over the length of its gradient.
func (model *orcaModel) orcaTrunk(p maths.Vec) float64 {
	table := model.orcaTrunkProfile()
	s := table.at(p.X)
	down := p.Y - s.centre
	ring := math.Hypot(down/s.high, p.Z/s.wide)
	gap := -math.Min(s.high, s.wide)
	if ring > 0 {
		slope := math.Hypot(down/(s.high*s.high), p.Z/(s.wide*s.wide)) / ring
		gap = (ring - 1) / slope
	}
	return math.Max(gap, math.Max(p.X-table.last, table.first-p.X))
}
