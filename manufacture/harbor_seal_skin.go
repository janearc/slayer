package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// An opening replaces a rectangle in the body grid with a flipper
// whose first ring shares the opening's exact vertex indices.
type sealOpening struct {
	start, end  int
	first, last int
	side        float64
	hind        bool
}

var sealOpenings = [...]sealOpening{
	{22, 26, 6, 10, 1, false},
	{22, 26, 14, 18, -1, false},
	{2, 6, 4, 9, 1, true},
	{2, 6, 15, 20, -1, true},
}

// NewSealRig builds the rest skin once. Both foreflippers and both hind
// flippers grow out of the same indexed shell as the trunk and head.
func NewSealRig() *SealRig {
	return newSealRig(sealShoulder)
}

// NewSealRigWithFlex uses the same rest skin with a different spine
// articulation. The start is an X coordinate in metres on the rest body.
func NewSealRigWithFlex(start float64) (*SealRig, error) {
	if math.IsNaN(start) || start < -.16 || start > .52 {
		return nil, fmt.Errorf("seal rig: flex start outside [-0.16, 0.52] m")
	}
	return newSealRig(start), nil
}

// newSealRig is the rig of the unposed body; start is the station along
// the body where its flexing begins.
func newSealRig(start float64) *SealRig {
	return buildSealRig(start, (HarborSealPose{}).body(), true)
}

// NewHarborSealRig builds the rest skin with five-digit flippers, a
// grown tail and the folds a pose bunches up, on the drawn profile.
func NewHarborSealRig() *SealRig {
	return buildSealRig(sealShoulder, (HarborSealPose{}).body(), false)
}

// buildSealRig builds the rest skin on the trunk's rings, from the tail
// root to the muzzle, with the flippers grown from openings in them. A
// plain rig has flippers of plain rings and a capped tail root.
func buildSealRig(start float64, rings [][]maths.Vec, plain bool) *SealRig {
	rig := &SealRig{flexStart: start, plain: plain}
	for _, ring := range rings {
		for _, point := range ring {
			point = sealMuzzle(point)
			rig.add(point, point.X, 0, 0)
		}
	}
	for row := 0; row+1 < len(rings); row++ {
		for column := range sealSections {
			if !sealHole(row, column) {
				next := (column + 1) % sealSections
				rig.joinAround(column, sealIndex(row, column),
					sealIndex(row, next), sealIndex(row+1, next),
					sealIndex(row+1, column))
			}
		}
	}
	if plain {
		rig.cap(sealBodyLoop(0), alfaMiddle(rings[0]), true, 0, 0)
	} else {
		rig.tail(alfaMiddle(rings[0]))
	}
	last := len(rings) - 1
	rig.cap(sealBodyLoop(last), alfaMiddle(rings[last]), false, 0, 0)
	for limb, opening := range sealOpenings {
		rig.branch(limb, opening)
	}
	rig.orient()
	rig.rest = make([]maths.Vec, len(rig.vertices))
	for index, vertex := range rig.vertices {
		rig.rest[index] = vertex.point
	}
	rig.settle()
	return rig
}

// The two whisker pads swell out of the same skin as the bridge and jaw.
// Keep the centre of the upper muzzle clear for the shallow nose break.
func sealMuzzle(point maths.Vec) maths.Vec {
	along := 1 - maths.Smooth(0, 1, math.Abs(point.X-.772)/.072)
	pad := 1 - maths.Smooth(0, 1, math.Abs(point.Y-.047)/.042)
	point.Z *= 1 + .23*along*pad
	bridge := 1 - maths.Smooth(0, 1, math.Abs(point.Z)/.060)
	upper := maths.Smooth(.077, .121, point.Y)
	point.Y -= .012 * along * bridge * upper
	// The outer forehead gives the eye an upper rim continuous with the skull.
	brow := 1 - maths.Smooth(0, 1, math.Abs(point.X-.660)/.068)
	corner := 1 - maths.Smooth(0, 1, math.Abs(math.Abs(point.Z)-.075)/.045)
	point.Y += .021 * brow * corner * maths.Smooth(.103, .146, point.Y)
	return point
}

// sealHole is whether the cell at a row and column of the body grid lies
// in one of the openings left for the limbs; such cells have no faces.
func sealHole(row, column int) bool {
	for _, hole := range sealOpenings {
		if row >= hole.start && row < hole.end &&
			column >= hole.first && column < hole.last {
			return true
		}
	}
	return false
}

// sealIndex is the index of the vertex at a row and column of the body
// grid, which has sealSections to a row.
func sealIndex(row, column int) int { return row*sealSections + column }

// sealBodyLoop is the vertex indices round the body at one row, one for
// each section.
func sealBodyLoop(row int) []int {
	loop := make([]int, sealSections)
	for column := range loop {
		loop[column] = sealIndex(row, column)
	}
	return loop
}

// boundary is the indices of the vertices round the opening's edge, in a
// loop.
func (hole sealOpening) boundary() []int {
	loop := make([]int, 0, 2*(hole.end-hole.start+hole.last-hole.first))
	for column := hole.first; column < hole.last; column++ {
		loop = append(loop, sealIndex(hole.start, column))
	}
	for row := hole.start; row < hole.end; row++ {
		loop = append(loop, sealIndex(row, hole.last))
	}
	for column := hole.last; column > hole.first; column-- {
		loop = append(loop, sealIndex(hole.end, column))
	}
	for row := hole.end; row > hole.start; row-- {
		loop = append(loop, sealIndex(row, hole.first))
	}
	return loop
}

// add adds a vertex and returns its index: the point, its station along
// the body, the limb that moves it and the weight that limb has over it.
func (rig *SealRig) add(point maths.Vec, station float64, limb int, weight float64) int {
	index := len(rig.vertices)
	rig.vertices = append(rig.vertices, sealVertex{point, station, limb, weight})
	return index
}

// join adds the quad of four vertices as two triangles.
func (rig *SealRig) join(first, second, third, fourth int) {
	rig.faces = append(rig.faces, [3]int{first, second, third},
		[3]int{first, third, fourth})
}

// joinAround adds a quad of a ring joined to the next, at a column of the
// ring. The columns of the ring's first half are cut along the diagonal
// join cuts and those of its second half along the other, so that a quad
// and its mirror image across the midline plane (a ring's column c lies
// opposite its column sealSections - c) are cut as mirror images.
//
// The skin is then as bilateral in its faces, and so in the normals the
// landmarks are set by, as it is in its points. A plain rig is the bay
// scene's seal, held to its original skin by bay_seal_original_test.go,
// and keeps the one diagonal.
func (rig *SealRig) joinAround(column, first, second, third, fourth int) {
	if rig.plain || column < sealSections/2 {
		rig.join(first, second, third, fourth)
		return
	}
	rig.faces = append(rig.faces, [3]int{first, second, fourth},
		[3]int{second, third, fourth})
}

// cap closes a loop of vertices with a pole at the centre, as a fan of
// triangles turned over if reverse; limb and weight say what moves it.
func (rig *SealRig) cap(loop []int, centre maths.Vec, reverse bool, limb int, weight float64) {
	station := centre.X
	if weight > 0 {
		station = rig.limbs[limb].root.X
	}
	pole := rig.add(centre, station, limb, weight)
	for index, first := range loop {
		second := loop[(index+1)%len(loop)]
		if reverse {
			first, second = second, first
		}
		rig.faces = append(rig.faces, [3]int{pole, first, second})
	}
}

// sealBranchRing is one flipper section: the offset of its centre from
// the root, how much it scales the opening's outline across and through,
// its turn, how much of the spine's frame it gives up for the flipper's
// own, and fan, how far the digits have spread at this section. A section
// with fan above zero is a digit ring, with a web vertex between each pair
// of outline vertices.
type sealBranchRing struct {
	dx, dy, dz, width, height, turn, influence, fan float64
}

// The foreflipper is a wrist, then three fan rings that spread five
// digits. Offsets and widths are chosen, to a flipper about a sixth of
// the body's length.
var foreBranch = []sealBranchRing{
	{-.011, -.019, .047, .99, .86, .08, .44, 0},
	{-.043, -.042, .092, 1.03, .63, .26, .66, 0},
	{-.087, -.063, .130, .98, .40, .53, .87, 0},
	{-.136, -.083, .156, .71, .29, .70, 1, 0},
	{-.168, -.087, .168, .74, .20, .74, 1, .30},
	{-.198, -.089, .178, .78, .13, .77, 1, .68},
	{-.224, -.090, .186, .80, .09, .77, 1, 1},
}

// The hindflipper runs back from the hip, its fan ending in five digits
// of nearly equal length with the outer two longest.
var hindBranch = []sealBranchRing{
	{-.018, -.007, .040, .87, .84, .35, .18, 0},
	{-.065, -.015, .066, .88, .70, .74, .53, 0},
	{-.127, -.024, .091, 1.16, .55, 1.05, .85, 0},
	{-.198, -.027, .110, 1.37, .38, 1.26, 1, 0},
	{-.232, -.026, .116, 1.24, .26, 1.34, 1, .30},
	{-.264, -.022, .122, 1.14, .17, 1.38, 1, .68},
	{-.290, -.018, .126, 1.08, .10, 1.38, 1, 1},
}

// A plain rig's foreflipper: six plain rings from the opening to a
// narrow tip, with no digits.
var forePlainBranch = []sealBranchRing{
	{-.011, -.019, .047, .99, .86, .08, .44, 0},
	{-.043, -.042, .092, 1.03, .63, .26, .66, 0},
	{-.087, -.063, .130, .98, .40, .53, .87, 0},
	{-.136, -.083, .156, .71, .29, .70, 1, 0},
	{-.189, -.089, .176, .37, .16, .77, 1, 0},
	{-.211, -.089, .178, .12, .08, .77, 1, 0},
}

// A plain rig's hindflipper: seven plain rings, with no digits.
var hindPlainBranch = []sealBranchRing{
	{-.018, -.007, .040, .87, .84, .35, .18, 0},
	{-.065, -.015, .066, .88, .70, .74, .53, 0},
	{-.127, -.024, .091, 1.16, .55, 1.05, .85, 0},
	{-.198, -.027, .110, 1.37, .38, 1.26, 1, 0},
	{-.253, -.025, .120, 1.20, .24, 1.38, 1, 0},
	{-.282, -.019, .126, .63, .14, 1.38, 1, 0},
	{-.294, -.017, .128, .23, .09, 1.38, 1, 0},
}

// sections is a flipper's rings from its opening outward: the plain
// rig's own, or the given ones.
func (rig *SealRig) sections(hind bool,
	profile []sealBranchRing,
) []sealBranchRing {
	if !rig.plain {
		return profile
	}
	if hind {
		return hindPlainBranch
	}
	return forePlainBranch
}

// branch grows a limb out of an opening, ring by ring from the opening's
// boundary toward the tip.
func (rig *SealRig) branch(limb int, hole sealOpening) {
	boundary := hole.boundary()
	root := maths.Vec{}
	for _, index := range boundary {
		root = root.Add(rig.vertices[index].point.Scale(1 / float64(len(boundary))))
	}
	profile, hand := foreBranch, foreHand
	if hole.hind {
		profile, hand = hindBranch, hindHand
	}
	profile = rig.sections(hole.hind, profile)
	rig.limbs[limb] = sealLimb{root: root, side: hole.side, hind: hole.hind}
	if !hole.hind {
		rig.shoulder(limb, hole)
	}
	previous, before := boundary, boundary
	for _, section := range profile {
		var ring []int
		var centre maths.Vec
		if section.fan > 0 {
			ring, centre = rig.fanRing(limb, boundary, section, hand)
		} else {
			ring, centre = rig.branchRing(limb, boundary, section)
		}
		rig.bridge(previous, ring)
		before, previous = previous, ring
		rig.limbs[limb].tip = centre
	}
	rig.cap(previous, rig.limbs[limb].tip, false, limb, 1)
	rig.limbs[limb].inner, rig.limbs[limb].outer = before, previous
	rig.limbs[limb].digits = hand.labels(len(boundary))
	rig.limbs[limb].pole = len(rig.vertices) - 1
}

// bridge joins a ring to the one before it, one face pair to an outline
// edge, or three faces where the next ring has a web vertex between each
// pair of outline vertices.
func (rig *SealRig) bridge(previous, ring []int) {
	for index, first := range previous {
		next := (index + 1) % len(previous)
		if len(ring) == len(previous) {
			rig.join(first, previous[next], ring[next], ring[index])
			continue
		}
		low, middle := ring[2*index], ring[2*index+1]
		high := ring[(2*index+2)%len(ring)]
		rig.faces = append(rig.faces, [3]int{first, previous[next], middle},
			[3]int{previous[next], high, middle}, [3]int{first, middle, low})
	}
}

// shoulder carries the foreflipper's movement into the surrounding trunk
// skin, with an influence that falls off away from the shared root.
func (rig *SealRig) shoulder(limb int, hole sealOpening) {
	const rows, columns = 4, 2
	const rootInfluence = .27
	for row := hole.start - rows; row <= hole.end+rows; row++ {
		for column := hole.first - columns; column <= hole.last+columns; column++ {
			long := max(0, max(hole.start-row, row-hole.end))
			across := max(0, max(hole.first-column, column-hole.last))
			reach := math.Hypot(float64(long)/rows, float64(across)/columns)
			if reach < 1 {
				point := &rig.vertices[sealIndex(row, column)]
				point.limb = limb
				point.weight = rootInfluence * (1 - sealEase(reach))
			}
		}
	}
}

// branchRing is one ring of a limb's branch, the boundary moved toward the
// section's offset from the joint; it returns the ring's vertices and its
// middle.
func (rig *SealRig) branchRing(limb int, boundary []int, section sealBranchRing) ([]int, maths.Vec) {
	joint := rig.limbs[limb]
	to := joint.root.Add(maths.V(section.dx, section.dy, joint.side*section.dz))
	ring := make([]int, 0, len(boundary))
	for _, index := range boundary {
		base := rig.vertices[index].point
		point, station := rig.sectionPoint(limb, base, section, 1, maths.Vec{})
		ring = append(ring, rig.add(point, station, limb, section.influence))
	}
	return ring, to
}

// sectionPoint carries a point of the opening's outline to a section: its
// offset from the root is scaled across and through, turned and set at the
// section's centre. thick scales the section's height at this point, and
// extra is a further offset in the flipper's own axes, chord, thickness
// and span. It returns the point and the spine station that carries it.
func (rig *SealRig) sectionPoint(limb int, base maths.Vec, section sealBranchRing,
	thick float64, extra maths.Vec,
) (maths.Vec, float64) {
	joint := rig.limbs[limb]
	to := joint.root.Add(maths.V(section.dx, section.dy, joint.side*section.dz))
	turn := maths.Rotation(maths.Up, -joint.side*section.turn)
	offset := base.Sub(joint.root)
	shaped := maths.V(offset.X*section.width, offset.Y*section.height*thick,
		offset.Z*(1-section.influence))
	station := base.X + section.influence*(joint.root.X-base.X)
	return to.Add(turn.Out(shaped.Add(extra))), station
}

// orient turns the faces over if the skin's signed volume is negative, so
// that they face out.
func (rig *SealRig) orient() {
	var signedVolume float64
	for _, face := range rig.faces {
		a, b, c := rig.vertices[face[0]].point, rig.vertices[face[1]].point, rig.vertices[face[2]].point
		signedVolume += a.Dot(b.Cross(c))
	}
	if signedVolume < 0 {
		for index := range rig.faces {
			face := &rig.faces[index]
			face[1], face[2] = face[2], face[1]
		}
	}
}
