package cities

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The ground the slope tests stand buildings on: a plane that rises along
// x, steeply enough that a bay of a terrace stands noticeably above the
// one before it.
const (
	rise      = 0.15
	heightTol = 1e-9
)

// flatWall is a material for a wall the test draws.
func flatWall(t *testing.T) *thing.Material {
	t.Helper()
	look, err := material(roleGlazing)
	if err != nil {
		t.Fatal(err)
	}
	return look
}

// incline is a ground that rises by a gradient along x and another along z.
func incline(alongX, alongZ float64) Terrain {
	return func(x, z float64) float64 { return alongX*x + alongZ*z }
}

// worldReach is the least and greatest corner of a drawn part, in the
// frame of the design it is part of.
func worldReach(one Rendered) (low, high [3]float64) {
	low = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	high = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, triangle := range one.Mesh.Triangles {
		for _, corner := range triangle {
			point := one.Place.Out(corner)
			for axis, value := range components(point) {
				low[axis] = math.Min(low[axis], value)
				high[axis] = math.Max(high[axis], value)
			}
		}
	}
	return low, high
}

// plinthsOf are the drawn plinths of a design.
func plinthsOf(t *testing.T, design Design) []Rendered {
	t.Helper()
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	var plinths []Rendered
	for _, one := range drawn {
		if one.Mesh.Name == "plinth" {
			plinths = append(plinths, one)
		}
	}
	return plinths
}

func TestFootingIsTheHighestAndLowestGroundUnderTheFootprint(t *testing.T) {
	footing, err := footingOn(rect{0, 0, 10, 6}, incline(0.2, 0.1))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(footing.Base-2.6) > heightTol || footing.Low != 0 ||
		math.Abs(footing.drop()-2.6) > heightTol {
		t.Errorf("footing %+v, want a base of 2.6 over a low of 0", footing)
	}
}

func TestAMoundUnderTheMiddleOfAFootprintRaisesTheFloor(t *testing.T) {
	mound := func(x, z float64) float64 {
		return math.Max(0, 3-math.Hypot(x-5, z-5))
	}
	footing, err := footingOn(rect{0, 0, 10, 10}, mound)
	if err != nil {
		t.Fatal(err)
	}
	if footing.Base != 3 || footing.Low != 0 {
		t.Errorf("footing %+v, want a floor on the mound at 3", footing)
	}
}

func TestLevelGroundNeedsNoPlinthAndLeavesTheBuildingAsDrawn(t *testing.T) {
	footprint := rect{0, 0, 8, 6}
	parts := []placedElement{put("wall", Box{8, 3, 6}, nil, maths.Vec{})}
	footing, err := footingOn(footprint, incline(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	planted, err := planted(parts, footprint, footing)
	if err != nil {
		t.Fatal(err)
	}
	if footing.drop() != 0 || len(planted) != 1 {
		t.Errorf("a drop of %v and %d parts on level ground",
			footing.drop(), len(planted))
	}
}

func TestNoBuildingFloatsAndNoneIsBuriedPastItsPlinth(t *testing.T) {
	random := rand.New(rand.NewPCG(5, 1))
	for range 40 {
		width, depth := 4+20*random.Float64(), 4+20*random.Float64()
		ground := incline(0.4*random.Float64()-0.2,
			0.4*random.Float64()-0.2)
		footprint := rect{-width / 2, -depth / 2, width / 2, depth / 2}
		footing, err := footingOn(footprint, ground)
		if err != nil {
			t.Fatal(err)
		}
		parts := []placedElement{put("wall", Box{width, 3, depth}, flatWall(t),
			maths.V(footprint.X0, 0, footprint.Z0))}
		planted, err := planted(parts, footprint, footing)
		if err != nil {
			t.Fatal(err)
		}
		lowest, floor := math.Inf(1), math.Inf(1)
		for _, part := range planted {
			one, err := part.Element.Rendered("quick")
			if err != nil {
				t.Fatal(err)
			}
			one.Place = part.Place
			low, _ := worldReach(one)
			lowest = math.Min(lowest, low[1])
			if one.Mesh.Name == "wall" {
				floor = low[1]
			}
		}
		for column := 0.0; column <= 1; column += 0.1 {
			for row := 0.0; row <= 1; row += 0.1 {
				under := ground(footprint.X0+column*width,
					footprint.Z0+row*depth)
				if under > floor+heightTol || under < lowest-heightTol {
					t.Fatalf("ground %v under a floor at %v and a plinth to "+
						"%v", under, floor, lowest)
				}
			}
		}
		if math.Abs(lowest-footing.Low) > heightTol {
			t.Errorf("the building reaches down to %v, and the lowest "+
				"ground is %v", lowest, footing.Low)
		}
	}
}

func TestATerraceOnASlopeStepsDownItOneBayAtATime(t *testing.T) {
	for _, culture := range []string{"groningen", "portofino", "midwest"} {
		design, err := TerraceOn(culture, 1, incline(rise, 0))
		if err != nil {
			t.Fatal(err)
		}
		plan := drawerOf(t, culture, 1).row()
		plinths := plinthsOf(t, design)
		if len(plinths) != plan.bays {
			t.Fatalf("%s: %d plinths for %d bays", culture, len(plinths),
				plan.bays)
		}
		for index, plinth := range plinths {
			low, high := worldReach(plinth)
			if math.Abs(low[1]-rise*float64(index)*plan.pitch) > heightTol ||
				math.Abs(high[1]-rise*float64(index+1)*plan.pitch) > heightTol {
				t.Errorf("%s: bay %d has a plinth from %v to %v", culture,
					index, low[1], high[1])
			}
		}
	}
}

func TestACondominiumOnASlopeStandsEachBuildingOnItsOwnGround(t *testing.T) {
	design, err := CondominiumOn(1, incline(rise, 0))
	if err != nil {
		t.Fatal(err)
	}
	plinths := plinthsOf(t, design)
	buildings := drawerOf(t, condominiumCulture, 1).count("buildings",
		place{})
	if len(plinths) != buildings+1 {
		t.Fatalf("%d plinths for a court and %d buildings", len(plinths),
			buildings)
	}
	for _, plinth := range plinths {
		low, high := worldReach(plinth)
		if math.Abs(low[1]-rise*low[0]) > heightTol ||
			math.Abs(high[1]-rise*high[0]) > heightTol {
			t.Errorf("a plinth from %v to %v over x from %v to %v", low[1],
				high[1], low[0], high[0])
		}
	}
}

func TestOnFlatGroundTheSlopeKindsAreWhatTheyWere(t *testing.T) {
	flat, err := TerraceOn("groningen", 3, incline(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	bare, err := Terrace("groningen", 3)
	if err != nil {
		t.Fatal(err)
	}
	// A terrace's bays are nodes, which hold functions; it is the same
	// terrace if it draws the same.
	flatDrawn, err := flat.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	bareDrawn, err := bare.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	if len(plinthsOf(t, flat)) != 0 ||
		digestOf(t, flatDrawn, nil) != digestOf(t, bareDrawn, nil) {
		t.Error("a terrace on level ground is not the terrace as drawn")
	}
}

func TestAGroundThatIsNoHeightOrAFootprintOfNoAreaIsRefused(t *testing.T) {
	notANumber := func(float64, float64) float64 { return math.NaN() }
	endless := func(float64, float64) float64 { return math.Inf(1) }
	for name, ground := range map[string]Terrain{"nan": notANumber,
		"infinite": endless} {
		if _, err := footingOn(rect{0, 0, 4, 4}, ground); err == nil {
			t.Errorf("a %s ground was read", name)
		}
		if _, err := TerraceOn("groningen", 1, ground); err == nil {
			t.Errorf("a terrace was stood on a %s ground", name)
		}
	}
	if _, err := footingOn(rect{0, 0, 0, 4}, incline(0, 0)); err == nil {
		t.Error("a footprint of no width was read")
	}
}
