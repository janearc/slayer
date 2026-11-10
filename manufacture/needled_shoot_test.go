package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestRecursiveShootHasAttachedContractingAxes(t *testing.T) {
	for _, style := range []int{1, 2} {
		plan := planShoot(style, 0.2, Sow(11))
		orders := map[int]bool{}
		for index, axis := range plan.axes {
			orders[axis.order] = true
			if index == 0 {
				continue
			}
			parent := plan.axes[axis.parent]
			delta := axis.at.Sub(parent.at)
			along := delta.Dot(parent.direction)
			if along <= 0 || along >= parent.length || delta.Sub(parent.direction.Scale(along)).Len() > 1e-12 {
				t.Fatal("lateral axis detached from its parent")
			}
			if axis.length >= parent.length || axis.radius >= parent.radius {
				t.Fatal("child did not contract")
			}
			// For the authored exponent 3/2, cross-section area contracts
			// as the cube of relative length. Check this relation without
			// repeating the implementation's fractional power.
			ratio := axis.length / parent.length
			area := axis.radius * axis.radius / (parent.radius * parent.radius)
			if math.Abs(area-ratio*ratio*ratio) > 1e-14 {
				t.Fatal("twig cross-section does not follow the authored contraction")
			}
		}
		if len(orders) != 3 {
			t.Fatalf("style %d has only %d branching orders", style, len(orders))
		}
	}
}

func TestNeedleDimensionsAndRanksFollowTheSpeciesDescriptions(t *testing.T) {
	for _, style := range []int{1, 2} {
		for _, length := range []float64{0.025, 0.05, 0.1, 0.2, 0.4} {
			plan := planShoot(style, length, Sow(7))
			ranks := map[int]bool{}
			for _, needle := range plan.needles {
				ranks[needle.rank] = true
				long := needle.tip.Sub(needle.root).Len()
				low, high := 0.01, 0.02 // OSU redwood description, converted cm to m.
				if style == 2 {
					low, high = 0.003, 0.006
				} // OSU leader scales, mm to m.
				if long < low-1e-12 || long > high+1e-12 {
					t.Fatalf("style %d leaf length %g m", style, long)
				}
				axis := plan.axes[needle.axis]
				delta := needle.root.Sub(axis.at)
				axial := delta.Dot(axis.direction)
				radial := delta.Sub(axis.direction.Scale(axial)).Len()
				wantRadius := axis.radius * (1 - 0.85*axial/axis.length)
				if math.Abs(radial-wantRadius) > 1e-12 {
					t.Fatal("needle root is not on the twig surface")
				}
				if style == 1 && math.Abs(needle.normal.Dot(maths.ZAxis)) < 1-1e-12 {
					t.Fatal("redwood spray lost its plane")
				}
			}
			want := 2
			if style == 2 {
				want = 3
			}
			if len(ranks) != want {
				t.Fatalf("style %d has %d ranks", style, len(ranks))
			}
		}
	}
}

func TestNeedledShootHasVisibleGapsAndClosedIndividualLeaves(t *testing.T) {
	shoot, err := prototype(1, 3, 0, "demo")
	if err != nil {
		t.Fatal(err)
	}
	box := shoot.leaves.Bounds()
	hits, misses := 0, 0
	for x := range 80 {
		for y := range 120 {
			at := maths.V(box.Min.X+(box.Max.X-box.Min.X)*(float64(x)+0.5)/80,
				box.Min.Y+(box.Max.Y-box.Min.Y)*(float64(y)+0.5)/120, 1)
			if _, hit := shoot.leaves.Hit(maths.Ray{At: at, Dir: maths.V(0, 0, -1)}, 0, 2); hit {
				hits++
			} else {
				misses++
			}
		}
	}
	if hits < 200 || misses < 2000 {
		t.Fatalf("leaf silhouette: %d hits, %d gaps", hits, misses)
	}
	for _, detail := range []string{"quick", "demo"} {
		needle := shoot.plan.needles[0]
		skin := needle.skin(detail)
		edges := map[[2]maths.Vec]int{}
		for _, face := range skin {
			for i := range 3 {
				edges[[2]maths.Vec{face[i], face[(i+1)%3]}]++
			}
		}
		for edge, count := range edges {
			if count != 1 || edges[[2]maths.Vec{edge[1], edge[0]}] != 1 {
				t.Fatal("leaf is not closed")
			}
		}
	}
}

func TestRedwoodTreeUsesSharedShootsInsteadOfGreenSleeves(t *testing.T) {
	for _, detail := range []string{"clumps", "quick", "demo"} {
		row, _ := KindNamed("coast-redwood")
		made, err := MakeWithComplexity(row, Sow(3), detail)
		if err != nil {
			t.Fatal(err)
		}
		things, err := made.Things(maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		var foliage *shape.MeshGroup
		for _, one := range things {
			if one.Material.Name == "redwood foliage" {
				foliage, _ = one.Shape.(*shape.MeshGroup)
			}
		}
		if foliage == nil || len(foliage.Members()) < 100 {
			t.Fatal("tree did not attach foliage groups")
		}
		if foliage.FaceCount() < 10*foliage.StoredFaces() {
			t.Fatal("shoot geometry was expanded instead of shared")
		}
		for _, part := range made.parts {
			if part.of == leaf {
				t.Fatal("green sleeve retained under the needles")
			}
		}
	}
}

// Check the actual polygonal wood too, not only its ideal circular envelope.
func TestNeedleBasesMeetTheRenderedWood(t *testing.T) {
	for _, style := range []int{1, 2} {
		for _, detail := range []string{"clumps", "quick", "demo"} {
			shoot, err := prototype(style, 4, 0, detail)
			if err != nil {
				t.Fatal(err)
			}
			step := max(1, len(shoot.plan.needles)/24)
			for i := 0; i < len(shoot.plan.needles); i += step {
				gap := math.Abs(shoot.wood.Distance(shoot.plan.needles[i].root))
				if gap > 1e-10 {
					t.Fatalf("style %d %s: base is %g m off the wood", style, detail, gap)
				}
			}
		}
	}
}
