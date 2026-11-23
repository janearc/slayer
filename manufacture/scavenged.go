package manufacture

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// ClusterKind is the name of the kind scavenged-cluster. A cluster is never
// a part of a cluster.
const ClusterKind = "scavenged-cluster"

// scavengedStream is the stream of draws a cluster takes its parts from,
// which no other use of its seed shares.
const scavengedStream = 7

// ClusterAsk is what a set asks of a cluster: the kinds its parts are
// drawn from, each with a weight, how many parts it has, and how far apart
// its bands stand. A kind written with twice the weight is twice as likely.
// A part that is nought or empty is left to the manufactory's data: any kind
// that fits the size range, the count and the spacing in scavenged.yaml.
type ClusterAsk struct {
	Parts       map[string]float64
	Count       int
	BandSpacing float64
}

// Clustered is the cluster kind as a set asks for it. It refuses a row that
// is not the cluster kind, a part that is not a kind the manufactory
// makes or is a cluster, a weight that is not above nought, and a count or
// a spacing below nought.
func Clustered(row Kind, ask ClusterAsk) (Kind, error) {
	if row.name != ClusterKind {
		return Kind{}, fmt.Errorf("manufacture: %s is not a cluster, and "+
			"takes no parts", row.name)
	}
	known := table()
	for name, weight := range ask.Parts {
		if _, made := known[name]; !made || name == ClusterKind ||
			!(weight > 0) {
			return Kind{}, fmt.Errorf("manufacture: a cluster takes %q "+
				"at weight %v, which is no kind of part or no weight", name,
				weight)
		}
	}
	if ask.Count < 0 || ask.BandSpacing < 0 {
		return Kind{}, fmt.Errorf("manufacture: a cluster has a count and "+
			"a spacing that are not below nought, not %d and %v", ask.Count,
			ask.BandSpacing)
	}
	return Kind{name: row.name, how: scavengedRow{ask: ask}}, nil
}

// scavengedRow is the kind scavenged-cluster: whole things the manufactory
// already makes, strapped where they touch, as package cities describes.
// It draws the parts, turns and makes each, asks package cities where they
// dock and what straps them, and gathers the lot into one good. It is made
// in orbit, where nothing settles.
type scavengedRow struct{ ask ClusterAsk }

// scavengedPart is a part drawn: the good made of its kind and turned, and
// what accretion needs to know of it.
type scavengedPart struct {
	kind   string
	turned good
	piece  cities.Piece
}

// scavenged is a cluster laid out: the plan it was drawn by, its parts as
// drawn, and where each docks.
type scavenged struct {
	plan    cities.ClusterPlan
	parts   []scavengedPart
	cluster cities.Cluster
}

// made is the cluster at the quick detail level.
func (row scavengedRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the cluster at the named detail level. It returns errGood when
// the seed is not made.
func (row scavengedRow) madeAt(from seed, detail string) (good, error) {
	laid, err := laidOut(from, detail, row.ask)
	if err != nil {
		return good{}, err
	}
	return laid.assembled()
}

// laidOut draws a cluster's parts from the seed and docks them. It
// returns errGood when the seed is not made.
func laidOut(from seed, detail string, ask ClusterAsk) (scavenged, error) {
	if !from.made() {
		return scavenged{}, errGood
	}
	plan, err := cities.Scavenged()
	if err != nil {
		return scavenged{}, err
	}
	if ask.Count > 0 {
		plan.Count = ask.Count
	}
	if ask.BandSpacing > 0 {
		plan.Band.Spacing = ask.BandSpacing
	}
	rng := rand.New(rand.NewPCG(from.number, scavengedStream))
	laid := scavenged{plan: plan}
	var pieces []cities.Piece
	for range plan.Count {
		part, err := drawPart(plan, ask, rng, detail)
		if err != nil {
			return scavenged{}, err
		}
		laid.parts = append(laid.parts, part)
		pieces = append(pieces, part.piece)
	}
	laid.cluster, err = cities.Accrete(plan, pieces, cities.Seed(from.number))
	return laid, err
}

// assembled is the cluster as one good: each part, turned, placed where it
// docks, and the bands, all gathered.
func (laid scavenged) assembled() (good, error) {
	var goods []good
	for _, join := range laid.cluster.Joins {
		placed, err := laid.parts[join.Piece].turned.placed(
			maths.NewSubspace(join.Offset, maths.Identity, 1))
		if err != nil {
			return good{}, err
		}
		goods = append(goods, placed)
	}
	if bands := laid.cluster.Lashings(); len(bands) > 0 {
		strapped, err := goodOfRendered(bands)
		if err != nil {
			return good{}, err
		}
		goods = append(goods, strapped)
	}
	return gather(goods...)
}

// report is what the cluster holds: its parts by kind, the volume they
// enclose and the people that makes room for.
func (laid scavenged) report() string {
	kinds := make([]string, len(laid.parts))
	for index, part := range laid.parts {
		kinds[index] = part.kind
	}
	return fmt.Sprintf("%d parts %v: %v", len(kinds), kinds,
		laid.plan.Interior.Occupancy(laid.cluster.Volume))
}

// drawPart draws one part: a kind, by the set's weights or from the whole
// catalogue, a seed for it and a turn, and makes it. A kind drawn from the
// catalogue is tried again if it cannot be made, is too heavy for a part
// or does not fit the size range; one the set names must be made. It says
// so if no kind fits in the tries the plan allows.
func drawPart(plan cities.ClusterPlan, ask ClusterAsk, rng *rand.Rand,
	detail string,
) (scavengedPart, error) {
	for range plan.DrawTries {
		name := drawKind(ask, rng)
		number := rng.Uint64()
		turn := randomTurn(rng)
		if len(ask.Parts) > 0 {
			return partOf(name, number, turn, detail)
		}
		if part, err := fittingPart(plan, name, number, turn,
			detail); err == nil {
			return part, nil
		}
	}
	return scavengedPart{}, fmt.Errorf("manufacture: no kind of a size "+
		"from %v to %v m was made in %d draws", plan.SizeLeast,
		plan.SizeMost, plan.DrawTries)
}

var (
	// errNoFit is a kind drawn from the catalogue whose size is outside
	// the plan's range.
	errNoFit = fmt.Errorf("manufacture: a part outside the size range")
	// errHeavy is a kind drawn from the catalogue that comes to more
	// faces, each repeated mesh counted where it stands, than a made
	// thing's budgets are drawn up within.
	errHeavy = fmt.Errorf("manufacture: a part of more than %d faces",
		mostFaces)
)

// fittingPart is a part drawn from the catalogue, or why it is not one:
// it cannot be made, it is too heavy, or it does not fit the plan's size
// range.
//
// A part is docked by its faces, every one of them held at once, so a kind
// that comes to more than mostFaces with its repeated meshes counted where
// they stand, a kelp forest, is not drawn for a part: another kind is.
// That is a choice among kinds nobody asked for by name, and no refusal of
// a thing: a set that names such a kind for a part has it made. The size
// is taken before any face is kept.
func fittingPart(plan cities.ClusterPlan, name string, number uint64,
	turn maths.Orientation, detail string,
) (scavengedPart, error) {
	made, err := madePart(name, number, detail)
	if err != nil {
		return scavengedPart{}, err
	}
	if heldFaces(made) > mostFaces {
		return scavengedPart{}, errHeavy
	}
	length, err := longestSide(made)
	if err != nil {
		return scavengedPart{}, err
	}
	if !fitsSize(plan, length) {
		return scavengedPart{}, errNoFit
	}
	return turnedPart(name, made, turn)
}

// drawKind is a kind name drawn from the set's parts by weight if it has
// any, and otherwise from the catalogue, less the cluster itself.
func drawKind(ask ClusterAsk, rng *rand.Rand) string {
	if len(ask.Parts) > 0 {
		names := make([]string, 0, len(ask.Parts))
		var total float64
		for name, weight := range ask.Parts {
			names = append(names, name)
			total += weight
		}
		slices.Sort(names)
		pick := rng.Float64() * total
		for _, name := range names {
			if pick -= ask.Parts[name]; pick < 0 {
				return name
			}
		}
		return names[len(names)-1]
	}
	var names []string
	for _, name := range kindNames() {
		if name != ClusterKind {
			names = append(names, name)
		}
	}
	return names[rng.IntN(len(names))]
}

// randomTurn is an orientation drawn evenly from all those there are: a
// direction for the axis drawn evenly over a sphere, and an angle about it
// drawn with the share of turns that large, by rejection.
func randomTurn(rng *rand.Rand) maths.Orientation {
	up := 2*rng.Float64() - 1
	round := 2 * math.Pi * rng.Float64()
	flat := math.Sqrt(1 - up*up)
	axis := maths.V(flat*math.Cos(round), up, flat*math.Sin(round))
	for {
		angle := math.Pi * rng.Float64()
		if share := math.Sin(angle / 2); rng.Float64() < share*share {
			return maths.Rotation(axis, angle)
		}
	}
}

// fitsSize is whether a length is in the plan's size range.
func fitsSize(plan cities.ClusterPlan, length float64) bool {
	return length >= plan.SizeLeast && length <= plan.SizeMost
}

// partOf is one of a kind made from a number at a detail level and turned,
// as a part: its faces, their box, the volume they enclose and its long
// axis. A part that encloses nothing cannot be lived in or weighed, and is
// refused.
func partOf(name string, number uint64, turn maths.Orientation,
	detail string,
) (scavengedPart, error) {
	made, err := madePart(name, number, detail)
	if err != nil {
		return scavengedPart{}, err
	}
	return turnedPart(name, made, turn)
}

// madePart is one of a kind made from a number at a detail level, before
// it is turned and before its faces are taken. A cluster is refused.
func madePart(name string, number uint64, detail string) (good, error) {
	if name == ClusterKind {
		return good{}, fmt.Errorf("manufacture: a cluster is not "+
			"a part of a cluster, and %s was drawn", name)
	}
	row, err := kindNamed(name)
	if err != nil {
		return good{}, err
	}
	made, err := MakeWithComplexity(row, sow(number), detail)
	if err != nil {
		return good{}, fmt.Errorf("manufacture: part %s: %w", name, err)
	}
	return made, nil
}

// heldFaces is how many faces a good comes to when every one is held, as
// trianglesOf holds them: each repeated mesh counted at every place it
// stands. It walks no face. A native shape that is no mesh counts for none
// here, and is refused when its faces are asked for.
func heldFaces(made good) int {
	faces := 0
	for _, native := range made.native {
		switch body := native.Shape.(type) {
		case *mesh.Mesh:
			if body != nil {
				faces += body.FaceCount()
			}
		case *shape.Mesh:
			if body != nil {
				faces += len(body.Faces)
			}
		case *shape.MeshGroup:
			if body != nil {
				faces += body.FaceCount()
			}
		}
	}
	skins := made.skins()
	for _, of := range made.slots() {
		if group := made.groups[of]; group != nil {
			faces += group.FaceCount()
			continue
		}
		faces += len(skins[of])
	}
	return faces
}

// longestSide is how long a good is along its longest axis: the longest side
// of the box of its faces, taken face by face and holding none of them.
func longestSide(made good) (float64, error) {
	box := cities.BoxOf(nil)
	low, high := box.Min, box.Max
	err := eachTriangle(made, func(face cities.Triangle) {
		for _, corner := range face {
			low = maths.V(math.Min(low.X, corner.X),
				math.Min(low.Y, corner.Y), math.Min(low.Z, corner.Z))
			high = maths.V(math.Max(high.X, corner.X),
				math.Max(high.Y, corner.Y), math.Max(high.Z, corner.Z))
		}
	})
	_, length := longest(high.Sub(low))
	return length, err
}

// turnedPart is a made good turned, as a part: its faces, their box, the
// volume they enclose and its long axis.
func turnedPart(name string, made good, turn maths.Orientation,
) (scavengedPart, error) {
	faces, err := trianglesOf(made)
	if err != nil {
		return scavengedPart{}, fmt.Errorf("manufacture: part %s: %w", name, err)
	}
	size := cities.BoxOf(faces).Max.Sub(cities.BoxOf(faces).Min)
	long, length := longest(size)
	for index, triangle := range faces {
		for corner := range triangle {
			faces[index][corner] = turn.Out(triangle[corner])
		}
	}
	piece, err := cities.NewPiece(name, faces, turn.Out(long), length)
	if err != nil {
		return scavengedPart{}, fmt.Errorf("manufacture: part %s: %w", name,
			err)
	}
	turned, err := made.placed(maths.NewSubspace(maths.Vec{}, turn, 1))
	return scavengedPart{kind: name, turned: turned, piece: piece}, err
}

// longest is the axis a size is longest along, as a unit direction, and how
// long it is.
func longest(size maths.Vec) (axis maths.Vec, length float64) {
	axis, length = maths.XAxis, size.X
	if size.Y > length {
		axis, length = maths.YAxis, size.Y
	}
	if size.Z > length {
		axis, length = maths.ZAxis, size.Z
	}
	return axis, length
}

// trianglesOf is every face of a good, in its own space: each part's
// surface, and each member of its repeated meshes. The matters are taken in
// their order, so that the same good always gives the same faces in the
// same order.
func trianglesOf(made good) ([]cities.Triangle, error) {
	var triangles []cities.Triangle
	err := eachTriangle(made, func(face cities.Triangle) {
		triangles = append(triangles, face)
	})
	if err != nil {
		return nil, err
	}
	return triangles, nil
}

// eachTriangle shows every face of a good, in its own space, to see, and
// holds none of them: each part's surface, and each member of its repeated
// meshes, the matters in their order.
func eachTriangle(made good, see func(cities.Triangle)) error {
	for _, native := range made.native {
		err := eachNativeTriangle(native.Shape, native.At(0), see)
		if err != nil {
			return err
		}
	}
	skins := made.skins()
	for _, of := range made.slots() {
		if group := made.groups[of]; group != nil {
			for _, member := range group.Members() {
				eachMemberTriangle(member, see)
			}
			continue
		}
		for _, each := range skins[of] {
			see(cities.Triangle(each))
		}
	}
	return nil
}

// nativeTriangles expands a native mesh in the good's local space. A shape
// with no triangle representation is refused: silently omitting one part
// would calculate the wrong volume and leave its berth too small.
func nativeTriangles(solid shape.Shape, at maths.Subspace) ([]cities.Triangle, error) {
	var triangles []cities.Triangle
	err := eachNativeTriangle(solid, at, func(face cities.Triangle) {
		triangles = append(triangles, face)
	})
	if err != nil {
		return nil, err
	}
	return triangles, nil
}

// eachNativeTriangle shows every face of a native mesh, in the good's
// local space, to see, or refuses a shape with no triangle representation.
func eachNativeTriangle(solid shape.Shape, at maths.Subspace,
	see func(cities.Triangle),
) error {
	switch body := solid.(type) {
	case *mesh.Mesh:
		if body == nil {
			return fmt.Errorf("native mesh is nil")
		}
		for i := 0; i < body.FaceCount(); i++ {
			a, b, c := body.Corners(i)
			see(cities.Triangle{at.Out(a), at.Out(b), at.Out(c)})
		}
	case *shape.Mesh:
		if body == nil {
			return fmt.Errorf("native shape mesh is nil")
		}
		for _, face := range body.Faces {
			see(cities.Triangle{at.Out(body.Vertices[face[0]]), at.Out(body.Vertices[face[1]]), at.Out(body.Vertices[face[2]])})
		}
	case *shape.MeshGroup:
		if body == nil {
			return fmt.Errorf("native mesh group is nil")
		}
		for _, member := range body.Members() {
			err := eachNativeTriangle(member.Mesh, member.Space.Within(at),
				see)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("native shape %T has no triangle representation", solid)
	}
	return nil
}

// eachMemberTriangle shows the faces of one member of a mesh group, in the
// space the group is in, to see. A member that is not a triangle mesh has
// none.
func eachMemberTriangle(member shape.MeshMember, see func(cities.Triangle)) {
	made, isMesh := member.Mesh.(*mesh.Mesh)
	if !isMesh {
		return
	}
	for index := range made.FaceCount() {
		first, second, third := made.Corners(index)
		see(cities.Triangle{member.Space.Out(first),
			member.Space.Out(second), member.Space.Out(third)})
	}
}
