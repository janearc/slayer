package cities

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/janearc/slayer/maths"
)

// Scavenged is a way of building whose units are whole things that
// something else already makes: a tank, a boat, a motorhome. Nothing in it
// is designed as a whole. A cluster is the largest of its parts, which is
// its core, and the rest, each docked against what is already there and
// banded to the part it touches, the way a people with a few hulls and a
// need for room might strap them together.
//
// This is version 1, in orbit, where nothing rests on anything. Each part
// is turned to an orientation the seed draws, in three dimensions, and
// docks to the cluster from a direction the seed draws, as close to the
// cluster's centre of mass as it fits, which is where it first meets the
// cluster as it slides in.
//
// Parts are boxes to one another: two parts never overlap in their boxes,
// and where one meets another they touch. Every part after the core is
// banded to the part it touches: a strap wraps both, its outline the
// outline of their two cross-sections joined by the lines that touch both
// from outside, built as a thin strip. Several bands stand along the shorter
// part's length, a spacing apart.
//
// A part's mass is the volume it encloses, and its room is some share of
// that volume, from which the cluster says how many people it holds and
// how crowded that is. Wear, dents, bending and water are not in version 1.
//
// Every number is in scavenged.yaml. Which kinds the parts are drawn from
// is the set's to say; the manufactory draws any that fit the size range if
// the set says none. The manufactory makes the parts and puts them where
// this package says; this package knows only their faces, boxes and
// volumes.

// roleLashing is the band that straps two parts together.
const roleLashing partRole = "lashing"

// parallel is how near to zero a component of a direction is to be
// taken as no motion along that axis.
const parallel = 1e-12

// touching is how far two boxes may sink into each other, in metres, and
// still be said to touch and not overlap.
const touching = 1e-9

// Piece is what accretion needs of one part, as it is turned in the
// cluster: its name, its faces about its own origin, their box, the volume
// they enclose, which is its mass, and its long axis, a unit direction,
// and how long it is along that axis.
type Piece struct {
	Name      string
	Triangles []Triangle
	Box       maths.Box
	Volume    float64
	Axis      maths.Vec
	Length    float64
}

// NewPiece is a part of a cluster from its faces, wound outward, its long
// axis and its length. A part that has no faces, no length or no axis, or
// encloses nothing, is refused.
func NewPiece(name string, triangles []Triangle, axis maths.Vec,
	length float64,
) (Piece, error) {
	volume := signedVolume(triangles)
	if len(triangles) == 0 || !(length > 0) || !(axis.Len() > 0) ||
		!(volume > 0) {
		return Piece{}, fmt.Errorf("cities: %s needs faces that enclose a "+
			"volume, an axis and a length, not %d faces enclosing %v on "+
			"%v for %v", name, len(triangles), volume, axis, length)
	}
	return Piece{Name: name, Triangles: triangles, Box: BoxOf(triangles),
		Volume: volume, Axis: axis.Unit(), Length: length}, nil
}

// BoxOf is the smallest box, with sides along the axes, that holds the
// faces.
func BoxOf(triangles []Triangle) maths.Box {
	low := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, triangle := range triangles {
		for _, corner := range triangle {
			low = maths.V(math.Min(low.X, corner.X),
				math.Min(low.Y, corner.Y), math.Min(low.Z, corner.Z))
			high = maths.V(math.Max(high.X, corner.X),
				math.Max(high.Y, corner.Y), math.Max(high.Z, corner.Z))
		}
	}
	return maths.Box{Min: low, Max: high}
}

// At is the piece's box when its own origin stands at an offset.
func (piece Piece) At(offset maths.Vec) maths.Box {
	return maths.Box{Min: piece.Box.Min.Add(offset),
		Max: piece.Box.Max.Add(offset)}
}

// strapBand is one strap round two parts: what is drawn, the place on the axis
// it stands at and the axis, a unit direction at right angles to its plane,
// and its inner outline, the convex outline of the two parts' cross-sections
// at that place, counter-clockwise about the axis.
type strapBand struct {
	Rendered Rendered
	Station  maths.Vec
	Axis     maths.Vec
	Hull     []maths.Vec
}

// Straps is the dimensions of a band, in metres: how wide it is along the
// axis, how thick it is, and how far apart bands stand along a lashing by
// default.
type Straps struct{ Width, Thickness, Spacing float64 }

// Interior is how room is made from volume: the share of a part's enclosed
// volume that people use, the height of one storey, and the floor each
// person has.
type Interior struct{ Share, Storey, PerPerson float64 }

// ClusterPlan is everything a scavenged cluster takes from data: how many parts
// by default, the size range of a part drawn from any kind, how many tries
// a draw has, the bands and the interior.
type ClusterPlan struct {
	Count                     int
	SizeLeast, SizeMost       float64
	DrawTries, DirectionTries int
	Band                      Straps
	Interior                  Interior
}

// occupancy is how many people an enclosed volume holds and how much
// floor that is.
type occupancy struct{ Volume, FloorArea, Occupants float64 }

// density is the occupants a cubic metre of enclosed volume holds.
func (occupancy occupancy) density() float64 {
	return occupancy.Occupants / occupancy.Volume
}

// String is the occupancy in a line, for a report.
func (occupancy occupancy) String() string {
	return fmt.Sprintf("%.0f m^3 enclosed, %.0f m^2 of floor, %.1f people, "+
		"%.4f people a m^3", occupancy.Volume, occupancy.FloorArea,
		occupancy.Occupants, occupancy.density())
}

// Join is one part of a cluster: which piece it is, where its own origin
// stands in the cluster, which earlier piece it touches and the bands that
// strap them. The core touches nothing and has no bands.
type Join struct {
	Piece  int
	Offset maths.Vec
	To     int
	Bands  []strapBand
}

// Cluster is pieces docked: the joins in the order they were made, the
// core first, and the centre of mass and the enclosed volume of the lot.
type Cluster struct {
	Joins  []Join
	Centre maths.Vec
	Volume float64
}

// Lashings are the bands, every part's after the core's, in order.
func (cluster Cluster) Lashings() []Rendered {
	var bands []Rendered
	for _, join := range cluster.Joins {
		for _, band := range join.Bands {
			bands = append(bands, band.Rendered)
		}
	}
	return bands
}

// signedVolume is the volume of the triangles' solid by the divergence
// theorem, summed about the origin: positive for faces wound outward and
// negative for faces wound inward.
func signedVolume(triangles []Triangle) float64 {
	var sum float64
	for _, triangle := range triangles {
		sum += triangle[0].Dot(triangle[1].Cross(triangle[2])) / signedSixth
	}
	return sum
}

// signedSixth is six, the factor between a triple product and the volume
// of the tetrahedron it spans from the origin.
const signedSixth = 6

// EnclosedVolume is the volume a mesh encloses. A mesh wound inward has a
// negative volume and one that encloses nothing has none, and both are
// refused.
func EnclosedVolume(mesh *Mesh) (float64, error) {
	volume := signedVolume(mesh.Triangles)
	if !(volume > 0) {
		return 0, fmt.Errorf("cities: mesh %q encloses %v, so it is wound "+
			"inward or encloses nothing", mesh.Name, volume)
	}
	return volume, nil
}

// Occupancy is the people a volume holds: its usable share, over a
// storey, is floor, and floor over what each has is people.
func (interior Interior) Occupancy(volume float64) occupancy {
	floor := volume * interior.Share / interior.Storey
	return occupancy{Volume: volume, FloorArea: floor,
		Occupants: floor / interior.PerPerson}
}

// Scavenged is the plan the data gives, or why it does not.
func Scavenged() (ClusterPlan, error) {
	reader := &choiceReader{}
	plan := ClusterPlan{Count: reader.whole("scavenged-part-count"),
		SizeLeast:      reader.of("scavenged-size-least"),
		SizeMost:       reader.of("scavenged-size-most"),
		DrawTries:      reader.whole("scavenged-draw-tries"),
		DirectionTries: reader.whole("scavenged-direction-tries"),
		Band: Straps{Width: reader.of("scavenged-band-width"),
			Thickness: reader.of("scavenged-band-thickness"),
			Spacing:   reader.of("scavenged-band-spacing")},
		Interior: Interior{Share: reader.of("scavenged-usable-share"),
			Storey:    reader.of("scavenged-storey-height"),
			PerPerson: reader.of("scavenged-floor-per-person")}}
	if reader.trouble != nil {
		return ClusterPlan{}, reader.trouble
	}
	return plan, plan.sound()
}

// sound refuses a plan that cannot make a cluster: no parts, a size range
// that is not one, a band or an interior of no size.
func (plan ClusterPlan) sound() error {
	switch {
	case plan.Count < 1 || plan.DrawTries < 1 || plan.DirectionTries < 1:
		return fmt.Errorf("cities: a cluster needs a part and a try at "+
			"each draw, not %+v", plan)
	case !(plan.SizeLeast > 0 && plan.SizeMost >= plan.SizeLeast):
		return fmt.Errorf("cities: a part's size runs from %v to %v, "+
			"which is no range", plan.SizeLeast, plan.SizeMost)
	case !(plan.Band.Width > 0 && plan.Band.Thickness > 0 &&
		plan.Band.Spacing > 0):
		return fmt.Errorf("cities: a band has a width, a thickness and a "+
			"spacing, not %+v", plan.Band)
	case !(plan.Interior.Share > 0 && plan.Interior.Share <= 1 &&
		plan.Interior.Storey > 0 && plan.Interior.PerPerson > 0):
		return fmt.Errorf("cities: an interior is a share of a volume, a "+
			"storey and a floor a person, not %+v", plan.Interior)
	}
	return nil
}

// components is a vector as its three components, so an axis can be a
// number.
func components(point maths.Vec) [3]float64 {
	return [3]float64{point.X, point.Y, point.Z}
}

// vector is three components as a vector.
func vector(of [3]float64) maths.Vec { return maths.V(of[0], of[1], of[2]) }

// centreOf is the middle of a box, by axis.
func centreOf(box maths.Box) [3]float64 {
	return components(box.Min.Add(box.Max).Scale(0.5))
}

// halfOf is half the size of a box, by axis.
func halfOf(box maths.Box) [3]float64 {
	return components(box.Max.Sub(box.Min).Scale(0.5))
}

// Overlap is whether the insides of two boxes meet, by more than a
// tolerance of the same unit as the boxes'.
func Overlap(first, second maths.Box, tolerance float64) bool {
	low, high := components(first.Min), components(first.Max)
	other, otherHigh := components(second.Min), components(second.Max)
	for axis := range low {
		if math.Min(high[axis], otherHigh[axis])-
			math.Max(low[axis], other[axis]) <= tolerance {
			return false
		}
	}
	return true
}

// docking is the state of a cluster as it is built: the boxes of what is
// placed, whose they are and where, the centre of mass and the mass so
// far, and the draws that choose directions.
type docking struct {
	plan    ClusterPlan
	pieces  []Piece
	rng     *rand.Rand
	boxes   []maths.Box
	owners  []int
	offsets map[int]maths.Vec
	centre  [3]float64
	mass    float64
}

// contact is where a sliding piece first meets the cluster: how far along
// its direction, the box it meets and the axis it meets it across.
type contact struct {
	along float64
	with  int
	axis  int
}

// direction is a direction drawn evenly over a sphere: its height from
// minus one to one, and its turn about the vertical a whole circle.
func direction(rng *rand.Rand) maths.Vec {
	up := 2*rng.Float64() - 1
	round := fullTurn * rng.Float64()
	flat := math.Sqrt(1 - up*up)
	return maths.V(flat*math.Cos(round), up, flat*math.Sin(round))
}

// passes is how far along a heading, from a point, a box of the given half
// sizes first leaves another box that is a given way off, which is where it
// stops overlapping that box on its way in, and across which axis. The
// room is the other box's middle less the point, and the sums are the two
// boxes' halves, by axis. It says false when the two never overlap along
// the heading.
func passes(heading, away, sums [3]float64) (leaves float64, axis int,
	hits bool,
) {
	low, high, last := math.Inf(-1), math.Inf(1), -1
	for each := range heading {
		if math.Abs(heading[each]) < parallel {
			if math.Abs(away[each]) >= sums[each] {
				return 0, 0, false
			}
			continue
		}
		first := (away[each] - sums[each]) / heading[each]
		second := (away[each] + sums[each]) / heading[each]
		low = math.Max(low, math.Min(first, second))
		if far := math.Max(first, second); far < high {
			high, last = far, each
		}
	}
	return high, last, last >= 0 && low < high
}

// slide is where a piece, sliding in along a way to the centre of mass,
// first meets the cluster: where it comes to rest on the farthest box it
// passes through. It says false when it misses the whole cluster.
func (dock *docking) slide(piece Piece, way maths.Vec) (contact, bool) {
	heading, own := components(way), halfOf(piece.Box)
	found := contact{along: math.Inf(-1)}
	for index, box := range dock.boxes {
		away, half := centreOf(box), halfOf(box)
		var sums [3]float64
		for each := range sums {
			away[each] -= dock.centre[each]
			sums[each] = own[each] + half[each]
		}
		leaves, axis, hits := passes(heading, away, sums)
		if hits && leaves > found.along {
			found = contact{along: leaves, with: index, axis: axis}
		}
	}
	return found, found.along >= 0
}

// adopt keeps a placed piece: its box, whose it is and where, and the
// centre of mass with its volume added in, at its box's middle.
func (dock *docking) adopt(index int, offset maths.Vec) {
	placed := dock.pieces[index].At(offset)
	middle, mass := centreOf(placed), dock.pieces[index].Volume
	for axis := range dock.centre {
		dock.centre[axis] = (dock.centre[axis]*dock.mass +
			middle[axis]*mass) / (dock.mass + mass)
	}
	dock.mass += mass
	dock.boxes = append(dock.boxes, placed)
	dock.owners = append(dock.owners, index)
	dock.offsets[index] = offset
}

// crowded is whether a box overlaps any box already placed.
func (dock *docking) crowded(placed maths.Box) bool {
	for _, box := range dock.boxes {
		if Overlap(placed, box, touching) {
			return true
		}
	}
	return false
}

// join docks a piece to the cluster: from the first direction the seed
// draws that meets it and leaves it clear, as close to the centre of mass
// as it fits, touching the part it meets, and banded to it. The first
// piece is the core and is left where it was made.
func (dock *docking) join(index int) (Join, error) {
	piece := dock.pieces[index]
	if len(dock.boxes) == 0 {
		dock.adopt(index, maths.Vec{})
		return Join{Piece: index, To: -1}, nil
	}
	for range dock.plan.DirectionTries {
		way := direction(dock.rng)
		met, hit := dock.slide(piece, way)
		if !hit {
			continue
		}
		middle := components(way.Scale(met.along))
		for axis := range middle {
			middle[axis] += dock.centre[axis]
		}
		offset := vector(middle).Sub(vector(centreOf(piece.Box)))
		if dock.crowded(piece.At(offset)) {
			continue
		}
		owner := dock.owners[met.with]
		bands := dock.strap(index, offset, owner, met.axis)
		if len(bands) == 0 {
			continue
		}
		dock.adopt(index, offset)
		return Join{Piece: index, Offset: offset, To: owner,
			Bands: bands}, nil
	}
	return Join{}, fmt.Errorf("cities: %s met the cluster from none of %d "+
		"directions, or could not be banded to it", piece.Name,
		dock.plan.DirectionTries)
}

// order is the pieces in the order they dock: the one of the most volume
// first, the core, and the rest in the order they were drawn.
func order(pieces []Piece) []int {
	core := 0
	for index, piece := range pieces {
		if piece.Volume > pieces[core].Volume {
			core = index
		}
	}
	docked := []int{core}
	for index := range pieces {
		if index != core {
			docked = append(docked, index)
		}
	}
	return docked
}

// Accrete docks pieces into a cluster: the largest by volume is the core
// at the origin, and each next piece, in the order drawn, is placed
// touching the cluster and overlapping no part, from a direction the seed
// draws, as near the centre of mass as it fits, and banded to the part it
// touches. It refuses an empty list of pieces.
func Accrete(plan ClusterPlan, pieces []Piece, seed Seed) (Cluster, error) {
	if len(pieces) == 0 {
		return Cluster{}, fmt.Errorf("cities: a cluster needs a piece")
	}
	if err := plan.sound(); err != nil {
		return Cluster{}, err
	}
	dock := &docking{plan: plan, pieces: pieces,
		rng:     rand.New(rand.NewPCG(uint64(seed), 3)),
		offsets: map[int]maths.Vec{}}
	var cluster Cluster
	for _, index := range order(pieces) {
		join, err := dock.join(index)
		if err != nil {
			return Cluster{}, err
		}
		cluster.Joins = append(cluster.Joins, join)
		cluster.Volume += pieces[index].Volume
	}
	cluster.Centre = vector(dock.centre)
	return cluster, nil
}

// span is how far along an axis a piece's faces reach, from the lowest to
// the highest, when its origin stands at an offset.
func span(piece Piece, offset, axis maths.Vec) (low, high float64) {
	low, high = math.Inf(1), math.Inf(-1)
	for _, triangle := range piece.Triangles {
		for _, corner := range triangle {
			along := corner.Add(offset).Dot(axis)
			low, high = math.Min(low, along), math.Max(high, along)
		}
	}
	return low, high
}

// seam is where two parts are banded: the axis the bands are across, and
// how far along it the two parts overlap.
type seam struct {
	axis      maths.Vec
	low, high float64
}

// seamOf is where a new part and the part it touches are banded: across the
// shorter one's own axis if the two overlap along it by a band's width, and
// otherwise across whichever axis of the boxes, but the one they touch
// across, they overlap longest on. Along a box's axis the parts' faces reach
// the box's own sides, so the boxes' overlap there is the faces'.
func (dock *docking) seamOf(index int, offset maths.Vec, owner,
	contactAxis int,
) (seam, bool) {
	newer, older := dock.pieces[index], dock.pieces[owner]
	shorter := newer
	if older.Length < newer.Length {
		shorter = older
	}
	candidates := []maths.Vec{shorter.Axis}
	for axis, unit := range []maths.Vec{maths.XAxis, maths.YAxis,
		maths.ZAxis} {
		if axis != contactAxis {
			candidates = append(candidates, unit)
		}
	}
	best, found := seam{}, false
	for rank, axis := range candidates {
		firstLow, firstHigh := span(newer, offset, axis)
		secondLow, secondHigh := span(older, dock.offsets[owner], axis)
		each := seam{axis: axis, low: math.Max(firstLow, secondLow),
			high: math.Min(firstHigh, secondHigh)}
		if each.high-each.low < dock.plan.Band.Width {
			continue
		}
		if rank == 0 {
			return each, true
		}
		if !found || each.high-each.low > best.high-best.low {
			best, found = each, true
		}
	}
	return best, found
}

// stations are the places along a seam the bands stand: evenly, about a
// spacing apart, at least one, none closer to either end than half a
// band's width.
func (dock *docking) stations(over seam) []float64 {
	length := over.high - over.low
	count := max(1, int(length/dock.plan.Band.Spacing))
	least := over.low + dock.plan.Band.Width/2
	most := over.high - dock.plan.Band.Width/2
	places := make([]float64, count)
	for index := range places {
		at := over.low + (float64(index)+0.5)*length/float64(count)
		places[index] = math.Min(math.Max(at, least), most)
	}
	return places
}

// strap is the bands that strap a new part to the part it touches: one at
// each station along their seam, wherever the two parts' cross-sections
// make an outline. It is none if there is no seam or no outline.
func (dock *docking) strap(index int, offset maths.Vec, owner,
	contactAxis int,
) []strapBand {
	over, banded := dock.seamOf(index, offset, owner, contactAxis)
	if !banded {
		return nil
	}
	var bands []strapBand
	for _, at := range dock.stations(over) {
		points := slice(dock.pieces[index], offset, over.axis, at)
		points = append(points, slice(dock.pieces[owner],
			dock.offsets[owner], over.axis, at)...)
		if band, made := dock.band(over.axis, at, points); made {
			bands = append(bands, band)
		}
	}
	return bands
}

// slice is the points where a piece's faces cross the plane at a place
// along an axis, when its origin stands at an offset: its cross-section's
// outline, as points.
func slice(piece Piece, offset, axis maths.Vec, at float64) []maths.Vec {
	var points []maths.Vec
	for _, triangle := range piece.Triangles {
		var apart [3]float64
		for corner := range triangle {
			apart[corner] = triangle[corner].Add(offset).Dot(axis) - at
		}
		for corner := range triangle {
			here, there := apart[corner], apart[(corner+1)%len(triangle)]
			from := triangle[corner].Add(offset)
			switch {
			case here == 0:
				points = append(points, from)
			case here*there < 0:
				to := triangle[(corner+1)%len(triangle)].Add(offset)
				points = append(points, from.Add(to.Sub(from).
					Scale(here/(here-there))))
			}
		}
	}
	return points
}

// plane is two unit directions at right angles to an axis and to each
// other, so that across, up and the axis are right handed.
func plane(axis maths.Vec) (across, up maths.Vec) {
	helper := maths.XAxis
	if math.Abs(axis.X) > math.Abs(axis.Y) {
		helper = maths.YAxis
	}
	across = axis.Cross(helper).Unit()
	return across, axis.Cross(across)
}

// hull is the convex outline of points seen along an axis, as points at a
// place along the axis, counter-clockwise about it.
func hull(points []maths.Vec, axis maths.Vec, at float64) []maths.Vec {
	across, up := plane(axis)
	flat := make([][2]float64, len(points))
	for index, point := range points {
		flat[index] = [2]float64{snapped(point.Dot(across)),
			snapped(point.Dot(up))}
	}
	slices.SortFunc(flat, func(first, second [2]float64) int {
		if first[0] != second[0] {
			return compared(first[0], second[0])
		}
		return compared(first[1], second[1])
	})
	flat = slices.Compact(flat)
	var corners []maths.Vec
	for _, corner := range convexOf(flat) {
		point := axis.Scale(at).Add(across.Scale(corner[0])).
			Add(up.Scale(corner[1]))
		if len(corners) == 0 || point.Sub(corners[len(corners)-1]).Len() >
			sameCorner {
			corners = append(corners, point)
		}
	}
	for len(corners) > 1 && corners[0].Sub(corners[len(corners)-1]).Len() <=
		sameCorner {
		corners = corners[:len(corners)-1]
	}
	return corners
}

// sameCorner is how near two corners of an outline, in metres, are to be
// taken as one: nearer than that and the edge between them has no
// direction to speak of.
const sameCorner = 1e-6

// hullGrid is the step, in metres, that the points of a cross-section are
// rounded to before their outline is found, so that the same point, found
// twice by neighbouring faces, is one point and not two a rounding error
// apart, which would give the outline an edge with no direction.
const hullGrid = 1e-7

// snapped is a coordinate rounded to the hull's grid.
func snapped(coordinate float64) float64 {
	return math.Round(coordinate/hullGrid) * hullGrid
}

// compared orders two numbers.
func compared(first, second float64) int {
	switch {
	case first < second:
		return -1
	case first > second:
		return 1
	}
	return 0
}

// convexOf is the convex outline of points sorted by their first then
// second coordinate, counter-clockwise, by Andrew's monotone chain, with no
// point on a straight run.
func convexOf(sorted [][2]float64) [][2]float64 {
	if len(sorted) < leastSides {
		return nil
	}
	var outline [][2]float64
	for range 2 {
		start := len(outline)
		for _, point := range sorted {
			for len(outline) >= start+2 && turns(outline[len(outline)-2],
				outline[len(outline)-1], point) <= 0 {
				outline = outline[:len(outline)-1]
			}
			outline = append(outline, point)
		}
		outline = outline[:len(outline)-1]
		slices.Reverse(sorted)
	}
	return outline
}

// turns is the cross product of the turn from one point through a second to
// a third: above nought for a turn to the left.
func turns(first, second, third [2]float64) float64 {
	return (second[0]-first[0])*(third[1]-first[1]) -
		(second[1]-first[1])*(third[0]-first[0])
}

// band is the strap round the points at a place along an axis: a thin strip
// round their convex outline. It says false if the points make no outline.
func (dock *docking) band(axis maths.Vec, at float64,
	points []maths.Vec,
) (strapBand, bool) {
	outline := hull(points, axis, at)
	if len(outline) < leastSides {
		return strapBand{}, false
	}
	material, err := material(roleLashing)
	if err != nil {
		return strapBand{}, false
	}
	mesh := stripAround(outline, axis, dock.plan.Band)
	return strapBand{Rendered: Rendered{Mesh: mesh, Material: material,
		Place: maths.NewSubspace(maths.Vec{}, maths.Identity, 1)},
		Station: axis.Scale(at), Axis: axis, Hull: outline}, true
}

// stripAround is a thin closed strip round a convex outline, counter-
// clockwise about the axis, as wide as the straps' width along the axis and
// as thick as their thickness outward from the outline, wound outward.
func stripAround(outline []maths.Vec, axis maths.Vec, straps Straps) *Mesh {
	outer := offsetOutline(outline, axis, straps.Thickness)
	half := axis.Scale(straps.Width / 2)
	mesh := &Mesh{Name: string(roleLashing)}
	for index := range outline {
		next := (index + 1) % len(outline)
		edge := outline[next].Sub(outline[index]).Cross(axis).Unit()
		mesh.Triangles = append(mesh.Triangles,
			quad(outer[index].Sub(half), outer[next].Sub(half),
				outer[next].Add(half), outer[index].Add(half), edge)...)
		mesh.Triangles = append(mesh.Triangles,
			quad(outline[index].Sub(half), outline[next].Sub(half),
				outline[next].Add(half), outline[index].Add(half),
				edge.Neg())...)
		mesh.Triangles = append(mesh.Triangles,
			quad(outline[index].Add(half), outline[next].Add(half),
				outer[next].Add(half), outer[index].Add(half), axis)...)
		mesh.Triangles = append(mesh.Triangles,
			quad(outline[index].Sub(half), outline[next].Sub(half),
				outer[next].Sub(half), outer[index].Sub(half), axis.Neg())...)
	}
	return mesh
}

// quad is a four cornered face as two triangles, wound to face the way
// given.
func quad(first, second, third, fourth, away maths.Vec) []Triangle {
	return []Triangle{outward(Triangle{first, second, third}, away),
		outward(Triangle{first, third, fourth}, away)}
}

// offsetOutline is a convex outline, counter-clockwise about an axis, moved
// outward by a distance everywhere: each corner along the middle of its two
// edges' outward directions, far enough that each edge moves the distance.
func offsetOutline(outline []maths.Vec, axis maths.Vec,
	distance float64,
) []maths.Vec {
	moved := make([]maths.Vec, len(outline))
	for index, corner := range outline {
		before := outline[(index+len(outline)-1)%len(outline)]
		after := outline[(index+1)%len(outline)]
		first := corner.Sub(before).Cross(axis).Unit()
		second := after.Sub(corner).Cross(axis).Unit()
		middle := first.Add(second).Unit()
		moved[index] = corner.Add(middle.Scale(distance / middle.Dot(first)))
	}
	return moved
}
