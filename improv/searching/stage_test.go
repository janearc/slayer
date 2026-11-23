package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

// Both Kecks receive one pose, but a fixed house must not inherit it.
func TestHeardGroundKeepsHouseAndSharesPointing(t *testing.T) {
	s := &heardStage{world: thing.NewWorld()}
	light := maths.Grey(1)
	s.world.Lamp(thing.Sun(maths.YAxis, light))
	for _, x := range []float64{-26, 26} {
		for role := range 3 {
			part := &thing.Thing{}
			s.objects = append(s.objects, stageObject{at: maths.V(x, 0, 0), pivot: maths.V(0, 10, 0), role: role, scale: 1, things: []*thing.Thing{part}})
		}
	}
	p := &stagePlan{Aim: [2]float64{-6, 6}, Elevation: [2]float64{-3, 2}, Sun: [2]float64{-15, -15}}
	if err := s.updateGround(p, State{At: 10}, .9); err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{0, 3} {
		pose := s.objects[i].things[0].At(0)
		if pose.Orientation != maths.Identity || pose.Origin != s.objects[i].at {
			t.Fatal("foundation moved")
		}
	}
	first, second := s.objects[2].things[0].At(0), s.objects[5].things[0].At(0)
	if first.Orientation != second.Orientation {
		t.Fatal("Kecks disagree")
	}
	if first.Orientation == maths.Identity {
		t.Fatal("optics did not move")
	}
	before := first
	if err := s.updateGround(p, State{At: 11}, 1); err != nil {
		t.Fatal(err)
	}
	if s.objects[2].things[0].At(0) != before {
		t.Fatal("settled optics kept moving")
	}
}
