package cities

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// clusterSeeds are the seeds each cluster test is tried with.
var clusterSeeds = []Seed{1, 2, 3, 17, 99}

// turnedBox is the piece of a box of a size, centred on its own origin and
// turned, with its long axis turned with it.
func turnedBox(t *testing.T, size maths.Vec, turn maths.Orientation) Piece {
	t.Helper()
	mesh, err := Box{size.X, size.Y, size.Z}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	middle := size.Scale(0.5)
	faces := make([]Triangle, len(mesh.Triangles))
	for index, triangle := range mesh.Triangles {
		for corner := range triangle {
			faces[index][corner] = turn.Out(triangle[corner].Sub(middle))
		}
	}
	long, length := maths.XAxis, size.X
	if size.Y > length {
		long, length = maths.YAxis, size.Y
	}
	if size.Z > length {
		long, length = maths.ZAxis, size.Z
	}
	piece, err := NewPiece("box", faces, turn.Out(long), length)
	if err != nil {
		t.Fatal(err)
	}
	return piece
}

// piecesOfBoxes is a dozen pieces of boxes of different sizes, each turned
// a different way, each of the volume of its box.
func piecesOfBoxes(t *testing.T) []Piece {
	var pieces []Piece
	for index := range 12 {
		size := maths.V(3+float64(index%4), 2+float64(index%3),
			4+float64(index%5))
		axis := maths.V(1, float64(index%3),
			float64(index%5)-1)
		turn := maths.Rotation(axis, 0.3+float64(index))
		pieces = append(pieces, turnedBox(t, size, turn))
	}
	return pieces
}

// theCluster is the dozen boxes docked from a seed, with a plan, or the
// test's end.
func theCluster(t *testing.T, plan ClusterPlan, seed Seed) ([]Piece, Cluster) {
	t.Helper()
	pieces := piecesOfBoxes(t)
	cluster, err := Accrete(plan, pieces, seed)
	if err != nil {
		t.Fatal(err)
	}
	return pieces, cluster
}

// thePlan is the plan the data gives, or the test's end.
func thePlan(t *testing.T) ClusterPlan {
	t.Helper()
	plan, err := Scavenged()
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// separation is how far apart two boxes are, nought if they touch or
// overlap.
func separation(first, second maths.Box) float64 {
	low, high := components(first.Min), components(first.Max)
	other, otherHigh := components(second.Min), components(second.Max)
	var squares float64
	for axis := range low {
		apart := math.Max(other[axis]-high[axis], low[axis]-otherHigh[axis])
		squares += math.Pow(math.Max(apart, 0), 2)
	}
	return math.Sqrt(squares)
}

// insideOutline is whether a point lies in or on a convex outline, counter-
// clockwise about an axis, to a tolerance.
func insideOutline(outline []maths.Vec, axis, point maths.Vec) bool {
	for index, corner := range outline {
		next := outline[(index+1)%len(outline)]
		edge := next.Sub(corner)
		if edge.Cross(point.Sub(corner)).Dot(axis)/edge.Len() < -1e-6 {
			return false
		}
	}
	return true
}

func TestNoTwoPartsOfAClusterOverlap(t *testing.T) {
	for _, seed := range clusterSeeds {
		pieces, cluster := theCluster(t, thePlan(t), seed)
		for first, one := range cluster.Joins {
			for _, other := range cluster.Joins[first+1:] {
				if Overlap(pieces[one.Piece].At(one.Offset),
					pieces[other.Piece].At(other.Offset), 1e-6) {
					t.Fatalf("seed %d: parts %d and %d overlap", seed,
						one.Piece, other.Piece)
				}
			}
		}
	}
}

func TestEveryPartAfterTheCoreTouchesAnEarlierPartWithNoGap(t *testing.T) {
	for _, seed := range clusterSeeds {
		pieces, cluster := theCluster(t, thePlan(t), seed)
		boxes := map[int]maths.Box{}
		for place, join := range cluster.Joins {
			box := pieces[join.Piece].At(join.Offset)
			boxes[join.Piece] = box
			if place == 0 {
				continue
			}
			met, earlier := boxes[join.To]
			if !earlier {
				t.Fatalf("seed %d: part %d touches %d, which is not "+
					"earlier", seed, join.Piece, join.To)
			}
			if got := separation(box, met); got > 1e-6 {
				t.Errorf("seed %d: part %d is %v from the part it touches",
					seed, join.Piece, got)
			}
		}
	}
}

func TestEveryPartAfterTheCoreHasAtLeastOneBand(t *testing.T) {
	for _, seed := range clusterSeeds {
		pieces, cluster := theCluster(t, thePlan(t), seed)
		bands := 0
		for place, join := range cluster.Joins {
			if place > 0 && len(join.Bands) == 0 {
				t.Errorf("seed %d: part %d has no band", seed, join.Piece)
			}
			bands += len(join.Bands)
		}
		if got := len(cluster.Lashings()); got != bands || got < len(pieces)-1 {
			t.Errorf("seed %d: %d lashings for %d bands and %d parts", seed,
				got, bands, len(pieces))
		}
	}
}

func TestABandEnclosesBothPartsCrossSectionsAtItsStation(t *testing.T) {
	for _, seed := range clusterSeeds {
		pieces, cluster := theCluster(t, thePlan(t), seed)
		offsets := map[int]maths.Vec{}
		for _, join := range cluster.Joins {
			offsets[join.Piece] = join.Offset
		}
		for _, join := range cluster.Joins {
			for _, band := range join.Bands {
				at := band.Station.Dot(band.Axis)
				for _, piece := range []int{join.Piece, join.To} {
					for _, point := range slice(pieces[piece],
						offsets[piece], band.Axis, at) {
						if !insideOutline(band.Hull, band.Axis, point) {
							t.Fatalf("seed %d: a band of part %d leaves "+
								"out %v of part %d", seed, join.Piece,
								point, piece)
						}
					}
				}
			}
		}
	}
}

func TestABandIsAThinClosedStripWoundOutward(t *testing.T) {
	plan := thePlan(t)
	_, cluster := theCluster(t, plan, 1)
	for _, join := range cluster.Joins {
		for _, band := range join.Bands {
			mesh := band.Rendered.Mesh
			if !closed(mesh) || volume(mesh) <= 0 {
				t.Fatalf("a band of part %d is not a closed solid wound "+
					"outward", join.Piece)
			}
			low, high := math.Inf(1), math.Inf(-1)
			for _, triangle := range mesh.Triangles {
				for _, corner := range triangle {
					low = math.Min(low, corner.Dot(band.Axis))
					high = math.Max(high, corner.Dot(band.Axis))
				}
			}
			if math.Abs(high-low-plan.Band.Width) > 1e-6 {
				t.Errorf("a band is %v wide, want %v", high-low,
					plan.Band.Width)
			}
		}
	}
}

func TestACloserSpacingMakesMoreBands(t *testing.T) {
	count := func(spacing float64) int {
		plan := thePlan(t)
		plan.Band.Spacing = spacing
		_, cluster := theCluster(t, plan, 1)
		return len(cluster.Lashings())
	}
	if near, far := count(1), count(1000); near <= far {
		t.Errorf("%d bands at a metre apart, %d at a kilometre", near, far)
	}
}

func TestTheHullOfTwoSeparateSquaresIsTheRectangleAroundThem(t *testing.T) {
	var points []maths.Vec
	for _, across := range []float64{0, 1, 2, 3} {
		for _, up := range []float64{0, 1} {
			points = append(points, maths.V(across, up, 5))
		}
	}
	points = append(points, maths.V(0.5, 0.5, 5), maths.V(2.5, 0.5, 5))
	outline := hull(points, maths.ZAxis, 5)
	if len(outline) != 4 {
		t.Fatalf("the hull has %d corners, want 4: %v", len(outline), outline)
	}
	var area float64
	for index, corner := range outline {
		next := outline[(index+1)%len(outline)]
		area += corner.X*next.Y - next.X*corner.Y
	}
	if math.Abs(area/2-3) > 1e-9 {
		t.Errorf("the hull has an area of %v counter-clockwise, want 3",
			area/2)
	}
}

func TestTheHullOfManyPointsHoldsThemAllAndTurnsOneWay(t *testing.T) {
	rng := rand.New(rand.NewPCG(4, 4))
	axis := maths.V(0.8865784263355807, 0.2509924623654698,
		-0.3885633510665199).Unit()
	across, up := plane(axis)
	for round := range 200 {
		var points []maths.Vec
		for range 300 {
			angle := fullTurn * rng.Float64()
			radius := 5 + 0.01*float64(round%7)*rng.Float64()
			rim := across.Scale(radius * math.Cos(angle))
			lift := up.Scale(radius * math.Sin(angle))
			end := axis.Scale(3)
			ringPoint := rim.Add(lift)
			points = append(points, ringPoint.Add(end))
		}
		outline := hull(points, axis, 3)
		for index, corner := range outline {
			next := outline[(index+1)%len(outline)]
			after := outline[(index+2)%len(outline)]
			if next.Sub(corner).Cross(after.Sub(next)).Dot(axis) <= 0 {
				t.Fatalf("round %d: the hull turns the wrong way at %d",
					round, index)
			}
		}
		for _, point := range points {
			if !insideOutline(outline, axis, point) {
				t.Fatalf("round %d: the hull leaves out %v", round, point)
			}
		}
	}
}

func TestTheCoreIsTheLargestPartAndStandsAtTheOrigin(t *testing.T) {
	pieces, cluster := theCluster(t, thePlan(t), 1)
	core := cluster.Joins[0]
	for _, piece := range pieces {
		if piece.Volume > pieces[core.Piece].Volume {
			t.Errorf("the core is %v and a part is %v",
				pieces[core.Piece].Volume, piece.Volume)
		}
	}
	if core.Offset != (maths.Vec{}) || core.To != -1 {
		t.Errorf("the core stands at %v touching %d", core.Offset, core.To)
	}
}

func TestTheCentreOfMassIsTheVolumeWeightedMiddle(t *testing.T) {
	pieces, cluster := theCluster(t, thePlan(t), 1)
	var sum maths.Vec
	var volume float64
	for _, join := range cluster.Joins {
		centre := centreOf(pieces[join.Piece].At(join.Offset))
		middle := vector(centre)
		sum = sum.Add(middle.Scale(pieces[join.Piece].Volume))
		volume += pieces[join.Piece].Volume
	}
	if got := sum.Scale(1 / volume); got.Sub(cluster.Centre).Len() > 1e-6 {
		t.Errorf("the centre is %v, the weighted middle %v", cluster.Centre,
			got)
	}
	if math.Abs(cluster.Volume-volume) > 1e-6 {
		t.Errorf("the cluster encloses %v, the parts %v", cluster.Volume,
			volume)
	}
}

func TestTheSameSeedMakesTheSameCluster(t *testing.T) {
	plan := thePlan(t)
	_, first := theCluster(t, plan, 5)
	_, again := theCluster(t, plan, 5)
	_, other := theCluster(t, plan, 6)
	same, differs := true, false
	for index := range first.Joins {
		same = same && first.Joins[index].Offset == again.Joins[index].Offset &&
			first.Joins[index].To == again.Joins[index].To
		differs = differs ||
			first.Joins[index].Offset != other.Joins[index].Offset
	}
	if !same {
		t.Error("seed 5 made two clusters")
	}
	if !differs {
		t.Error("seeds 5 and 6 made one cluster")
	}
}

func TestTheEnclosedVolumeOfABoxIsItsVolume(t *testing.T) {
	for box, want := range map[Box]float64{{1, 1, 1}: 1, {2, 1, 1}: 2,
		{1, 3, 2}: 6} {
		mesh, err := box.Mesh("quick")
		if err != nil {
			t.Fatal(err)
		}
		got, err := EnclosedVolume(mesh)
		if err != nil || math.Abs(got-want) > tolerance {
			t.Errorf("%v encloses %v, %v; want %v", box, got, err, want)
		}
	}
}

func TestTheVolumeOfAPrismIsItsAreaOverItsHeight(t *testing.T) {
	mesh, err := prism{Bottom: outlineCircle{2}, Top: outlineCircle{2},
		Height: 3}.Mesh("demo")
	if err != nil {
		t.Fatal(err)
	}
	sides := mustChoose(t, "circle-sides-demo")
	area := sides * math.Sin(fullTurn/sides) / 2 * 4
	if got, _ := EnclosedVolume(mesh); math.Abs(got-area*3) > tolerance {
		t.Errorf("the prism encloses %v, want %v", got, area*3)
	}
}

func TestAnInwardMeshHasANegativeVolumeAndIsRefused(t *testing.T) {
	mesh, err := Box{1, 1, 1}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	for index, triangle := range mesh.Triangles {
		mesh.Triangles[index] = Triangle{triangle[0], triangle[2], triangle[1]}
	}
	if got := signedVolume(mesh.Triangles); math.Abs(got+1) > tolerance {
		t.Errorf("a box wound inward has a volume of %v, want -1", got)
	}
	if _, err := EnclosedVolume(mesh); err == nil {
		t.Error("a mesh wound inward was taken as enclosing a volume")
	}
	if _, err := NewPiece("inward", mesh.Triangles, maths.XAxis, 1); err == nil {
		t.Error("a part wound inward was taken as a piece")
	}
	if _, err := EnclosedVolume(&Mesh{}); err == nil {
		t.Error("a mesh of no faces was taken as enclosing a volume")
	}
}

func TestAnInteriorTurnsVolumeIntoFloorAndPeople(t *testing.T) {
	plan := thePlan(t)
	have := plan.Interior.Occupancy(1000)
	floor := 1000 * plan.Interior.Share / plan.Interior.Storey
	if math.Abs(have.FloorArea-floor) > tolerance ||
		math.Abs(have.Occupants-floor/plan.Interior.PerPerson) > tolerance ||
		math.Abs(have.density()-have.Occupants/1000) > tolerance {
		t.Errorf("1000 m^3 holds %v", have)
	}
	if double := plan.Interior.Occupancy(2000); math.Abs(double.density()-
		have.density()) > tolerance {
		t.Errorf("density changed with size: %v, %v", have.density(),
			double.density())
	}
}

func TestThePlanIsTheDatas(t *testing.T) {
	plan := thePlan(t)
	if plan.Count != 12 || plan.Band.Width != 0.3 ||
		plan.Band.Thickness != 0.05 || plan.Band.Spacing <= 0 {
		t.Errorf("the plan is %+v", plan)
	}
}

func TestAClusterOfNothingOrOfAPartWithNoFacesIsRefused(t *testing.T) {
	plan := thePlan(t)
	if _, err := Accrete(plan, nil, 1); err == nil {
		t.Error("a cluster of no pieces was made")
	}
	if _, err := NewPiece("bare", nil, maths.XAxis, 1); err == nil {
		t.Error("a part with no faces was taken as a piece")
	}
	mesh, _ := Box{1, 1, 1}.Mesh("quick")
	if _, err := NewPiece("short", mesh.Triangles, maths.XAxis, 0); err == nil {
		t.Error("a part with no length was taken as a piece")
	}
	if _, err := NewPiece("aimless", mesh.Triangles, maths.Vec{}, 1); err == nil {
		t.Error("a part with no axis was taken as a piece")
	}
}

func TestAPlanThatCannotMakeAClusterIsRefused(t *testing.T) {
	good := thePlan(t)
	for name, change := range map[string]func(*ClusterPlan){
		"no parts":         func(plan *ClusterPlan) { plan.Count = 0 },
		"a size backwards": func(plan *ClusterPlan) { plan.SizeMost = plan.SizeLeast / 2 },
		"no band":          func(plan *ClusterPlan) { plan.Band.Width = 0 },
		"no spacing":       func(plan *ClusterPlan) { plan.Band.Spacing = 0 },
		"too much usable":  func(plan *ClusterPlan) { plan.Interior.Share = 1.5 },
		"no floor":         func(plan *ClusterPlan) { plan.Interior.PerPerson = 0 },
	} {
		plan := good
		change(&plan)
		if plan.sound() == nil {
			t.Errorf("a plan with %s was taken", name)
		}
	}
}
