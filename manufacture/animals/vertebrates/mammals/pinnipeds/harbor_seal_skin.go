package pinnipeds

import (
	"errors"
	"math"
	"strings"

	"github.com/janearc/slayer/manufacture/geometry"
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

// newSealRig is the rig of the unposed body; start is the station along
// the body where its flexing begins.
func newSealRig(start float64) *SealRig {
	return buildSealRig(start, sealRestRings(), true)
}

// NewHarborSealRig builds the rest skin with five-digit flippers, a
// grown tail and the folds a pose bunches up, on the drawn profile.
func NewHarborSealRig() *SealRig {
	return buildSealRig(sealShoulder, sealRestRings(), false)
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
		rig.cap(sealBodyLoop(0), geometry.Middle(rings[0]), true, 0, 0)
	} else {
		rig.tail(geometry.Middle(rings[0]))
	}
	last := len(rings) - 1
	rig.cap(sealBodyLoop(last), geometry.Middle(rings[last]), false, 0, 0)
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

// sealTailRing is one section of the tail: how far back of the last
// trunk section it lies, and its share of that section's outline.
type sealTailRing struct{ back, scale float64 }

// sealTailRings make the short tail, about eight centimetres, that tapers
// out of the end of the trunk between the hindflippers. The numbers are
// chosen.
var sealTailRings = []sealTailRing{
	{-.026, .84}, {-.052, .62}, {-.070, .36}, {-.080, .12},
}

// tail grows the short tail out of the rear trunk ring, as more of the
// same indexed skin, and closes it. middle is the centre of that ring.
func (rig *SealRig) tail(middle maths.Vec) {
	previous := sealBodyLoop(0)
	for _, section := range sealTailRings {
		ring := make([]int, len(previous))
		for column := range ring {
			offset := rig.vertices[sealIndex(0, column)].point.Sub(middle)
			point := middle.Add(maths.V(section.back, 0, 0)).
				Add(offset.Scale(section.scale))
			ring[column] = rig.add(point, point.X, 0, 0)
		}
		for column, first := range ring {
			next := (column + 1) % len(ring)
			rig.joinAround(column, first, ring[next], previous[next],
				previous[column])
		}
		previous = ring
	}
	end := sealTailRings[len(sealTailRings)-1].back
	rig.cap(previous, middle.Add(maths.V(end, 0, 0)), true, 0, 0)
}

// settle works out what depends on the finished rest skin: its vertex
// normals, the way the fur lies at each vertex, and which vertices are
// bare skin, the digit tips, where claws grow instead of fur.
func (rig *SealRig) settle() {
	rig.normals = make([]maths.Vec, len(rig.rest))
	for index, normal := range geometry.SmoothNormals(rig.rest, rig.faces) {
		rig.normals[index] = normal.Unit()
	}
	field := rig.flowField()
	rig.flows = make([]maths.Vec, len(rig.rest))
	for index, point := range rig.rest {
		rig.flows[index] = field.at(point)
	}
	rig.bare = make([]bool, len(rig.rest))
	for _, limb := range rig.limbs {
		for _, index := range limb.outer {
			rig.bare[index] = true
		}
		rig.bare[limb.pole] = true
	}
}

// sealFlow is the direction the coat lies at a point, in rest space.
type sealFlow struct{ roots [len(sealOpenings)]maths.Vec }

// flowField is the coat's direction field: from the muzzle back toward
// the tail, fanned out from the midline on the head, and away from each
// flipper root over the flipper.
func (rig *SealRig) flowField() sealFlow {
	var flow sealFlow
	for index, limb := range rig.limbs {
		flow.roots[index] = limb.root
	}
	return flow
}

const (
	// sealFlowLean is the distance from the midline, in metres, at which
	// the head's coat has leaned half way out. Chosen.
	sealFlowLean = .04
	// sealFlowFan is how far the head's coat fans out against how far it
	// runs back. Chosen.
	sealFlowFan = .8
	// sealFlowRoot is how strongly a flipper root pulls the coat outward
	// against the run toward the tail. Chosen.
	sealFlowRoot = 1.6
	// sealFlowNear and sealFlowFar are the distances from a flipper root,
	// in metres, between which its pull falls away.
	sealFlowNear, sealFlowFar = .04, .26
	// sealFlowHeadStart and sealFlowHeadEnd are the X, in metres, over
	// which the coat's fan comes in toward the muzzle.
	sealFlowHeadStart, sealFlowHeadEnd = .58, .72
)

// at is the unit direction the coat lies at a rest point.
func (flow sealFlow) at(point maths.Vec) maths.Vec {
	lean := point.Z / (math.Abs(point.Z) + sealFlowLean)
	head := maths.Smooth(sealFlowHeadStart, sealFlowHeadEnd, point.X)
	direction := maths.V(-1, 0, sealFlowFan*head*lean)
	for _, root := range flow.roots {
		away := point.Sub(root)
		if away.Len() == 0 {
			continue
		}
		pull := 1 - maths.Smooth(sealFlowNear, sealFlowFar, away.Len())
		direction = direction.Add(away.Unit().Scale(sealFlowRoot * pull))
	}
	return direction.Unit()
}

// restHit is where a ray first meets the rest skin, with the skin's unit
// normal there. It is how every landmark is set on the surface, not near
// it.
func (rig *SealRig) restHit(origin, direction maths.Vec) (maths.Vec, maths.Vec, bool) {
	best := math.Inf(1)
	var point, normal maths.Vec
	for _, face := range rig.faces {
		corners := [3]maths.Vec{rig.rest[face[0]], rig.rest[face[1]], rig.rest[face[2]]}
		distance, first, second, hit := sealRayTriangle(origin, direction, corners)
		if !hit || distance >= best {
			continue
		}
		best = distance
		point = origin.Add(direction.Scale(distance))
		normal = rig.normals[face[0]].Scale(1 - first - second).
			Add(rig.normals[face[1]].Scale(first)).
			Add(rig.normals[face[2]].Scale(second)).Unit()
	}
	return point, normal, !math.IsInf(best, 1)
}

// sealRayTriangle is the distance along a ray to a triangle and the
// triangle's barycentric weights for its second and third corners.
// Möller and Trumbore, "Fast, minimum storage ray-triangle intersection",
// Journal of Graphics Tools 2(1), 1997. The triangle is hit from either
// side, and a miss behind the origin is a miss.
func sealRayTriangle(origin, direction maths.Vec, corners [3]maths.Vec) (float64, float64, float64, bool) {
	const parallel = 1e-14
	first, second := corners[1].Sub(corners[0]), corners[2].Sub(corners[0])
	pivot := direction.Cross(second)
	determinant := first.Dot(pivot)
	if math.Abs(determinant) < parallel {
		return 0, 0, 0, false
	}
	from := origin.Sub(corners[0])
	u := from.Dot(pivot) / determinant
	across := from.Cross(first)
	v := direction.Dot(across) / determinant
	distance := second.Dot(across) / determinant
	return distance, u, v, u >= 0 && v >= 0 && u+v <= 1 && distance > 0
}

// sealMass is one mass of the body at rest, held on a bone: an
// ellipsoid whose centre is offset from the bone's head, with half axes
// along the body, up and across, all in metres. A mirrored mass is on
// both sides, written for the right.
type sealMass struct {
	bone         string
	offset, half maths.Vec
	mirrored     bool
}

// sealMasses are the masses the trunk, neck and head are made of, from
// the tail forward. Each is placed on the bone it lies against; offsets
// and half axes are chosen, as no published girths were reached.
var sealMasses = []sealMass{
	// The root of the tail between the hind flippers.
	{"caudal3", maths.V(-.123, -.054, 0), maths.V(.08, .05, .045), false},
	// The hip and the muscles of the pelvic stroke.
	{"sacral1", maths.V(-.11, -.074, .04), maths.V(.18, .10, .08), true},
	// The gut and the belly wall under the lumbar column.
	{"lumbar2", maths.V(-.03, -.099, 0), maths.V(.30, .17, .19), false},
	// The long back muscles on each side of the column.
	{"thoracic10", maths.V(-.17, -.024, .05), maths.V(.55, .07, .06), true},
	// The heart, the lungs and the rib cage with its muscle.
	{"thoracic8", maths.V(-.08, -.109, 0), maths.V(.36, .18, .20), false},
	// The shoulder: the muscles over the scapula and the arm.
	{"fore-right-scapula", maths.V(-.03, .04, -.05), maths.V(.12, .14, .10),
		true},
	// The neck's muscles round the cervical column.
	{"neck3", maths.V(.007, -.04, 0), maths.V(.16, .11, .11), false},
	// The throat and the floor of the mouth.
	{"skull", maths.V(.02, -.101, 0), maths.V(.10, .06, .07), false},
	// The braincase and its jaw muscles.
	{"skull", maths.V(.06, -.016, 0), maths.V(.10, .085, .09), false},
	// The muzzle over the rostrum, with the whisker pads.
	{"skull", maths.V(.17, -.051, 0), maths.V(.06, .045, .055), false},
}

// Blubber over the masses, in metres: thick on the trunk, thin on the
// head and gone at the nose's tip. Chosen, under the 25 mm a search
// summary gave for one resting seal; that page was not opened.
const (
	sealTrunkBlubber = .025
	sealHeadBlubber  = .012
	sealBlubberFall  = .55 // the X where the head's thin blubber begins
	sealBlubberBlend = .08 // how far along X the two blend
	sealNoseStart    = .80 // the X where the blubber thins to the tip
	sealNoseTip      = .835
)

// sealSoftness is how far, in metres, the join of two masses is rounded
// rather than creased: the width of their soft maximum. Chosen.
const sealSoftness = .012

// sealLeast is the smallest reach of the skin from its centre, in
// metres, where no mass lies, as at the tip of the tail. Chosen.
const sealLeast = .004

var errSealMassBone = errors.New("harbor seal: a mass's bone is not here")

// sealBody is the masses placed in the rest space of the skeleton.
type sealBody struct{ centres, halves []maths.Vec }

// newSealBody places every mass on its bone, a mirrored one on both
// sides.
func newSealBody(bones *sealBones) (sealBody, error) {
	var body sealBody
	for _, mass := range sealMasses {
		sides := []float64{1}
		if mass.mirrored {
			sides = []float64{-1, 1}
		}
		for _, side := range sides {
			head, _, found := bones.ends(sealSideBone(mass.bone, side))
			if !found {
				return sealBody{}, errSealMassBone
			}
			body.centres = append(body.centres,
				head.Add(sealMirror(mass.offset, side)))
			body.halves = append(body.halves, mass.half)
		}
	}
	return body, nil
}

// sealSideBone is a right-side bone's name for the side; a name with no
// side is itself.
func sealSideBone(name string, side float64) string {
	return strings.Replace(name, "-right-", "-"+sealSideName(side)+"-", 1)
}

// reach is how far the body's surface lies from centre along way, a
// unit direction: the soft maximum of where way leaves each mass it
// passes through, and the blubber over it.
func (body sealBody) reach(centre, way maths.Vec) float64 {
	total, found := 0.0, false
	for index, middle := range body.centres {
		out, through := sealExit(centre, way, middle, body.halves[index])
		if through {
			total += math.Exp(out / sealSoftness)
			found = true
		}
	}
	reach := sealLeast
	if found {
		reach = math.Max(reach, sealSoftness*math.Log(total))
	}
	return reach + sealBlubber(centre.X)
}

// sealBlubber is the blubber's depth at x.
func sealBlubber(x float64) float64 {
	head := maths.Smooth(sealBlubberFall-sealBlubberBlend,
		sealBlubberFall+sealBlubberBlend, x)
	depth := sealTrunkBlubber + (sealHeadBlubber-sealTrunkBlubber)*head
	return depth * (1 - maths.Smooth(sealNoseStart, sealNoseTip, x))
}

// sealExit is how far along way from start a ray leaves the ellipsoid
// of centre middle and half axes half, and whether it leaves it ahead
// of start at all.
func sealExit(start, way, middle, half maths.Vec) (float64, bool) {
	scale := func(point maths.Vec) maths.Vec {
		return maths.V(point.X/half.X, point.Y/half.Y, point.Z/half.Z)
	}
	from, toward := scale(start.Sub(middle)), scale(way)
	a := toward.Dot(toward)
	b := 2 * from.Dot(toward)
	c := from.Dot(from) - 1
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0, false
	}
	far := (-b + math.Sqrt(discriminant)) / (2 * a)
	return far, far > 0
}

// shape is each rest ring moved out or in along its own spokes to the
// body's surface. The rings are changed in place and returned.
func (body sealBody) shape(rings [][]maths.Vec) [][]maths.Vec {
	for _, ring := range rings {
		centre := geometry.Middle(ring)
		for index, point := range ring {
			way := point.Sub(centre).Unit()
			ring[index] = centre.Add(way.Scale(body.reach(centre, way)))
		}
	}
	return rings
}
