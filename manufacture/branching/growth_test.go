package branching

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

type traceSeed struct {
	path string
	log  *[]string
}

func (s traceSeed) Draw(row, index, which int) float64 {
	*s.log = append(*s.log, fmt.Sprintf("draw %s %d", s.path, index))
	return 0
}
func (s traceSeed) Swing(row, index, which int) float64 { return s.Draw(row, index, which) }
func (s traceSeed) OnPiece(piece int) Seed {
	*s.log = append(*s.log, fmt.Sprintf("piece %s %d", s.path, piece))
	return s
}
func (s traceSeed) Sprout(row, index int) Seed {
	*s.log = append(*s.log, fmt.Sprintf("sprout %s %d", s.path, index))
	return traceSeed{fmt.Sprintf("%s.%d", s.path, index), s.log}
}

type traceArrangement struct{ count int }

func (a traceArrangement) Rows() int      { return 1 }
func (a traceArrangement) Across(int) int { return a.count }
func (a traceArrangement) At(_ Size, row, index int, from Draws) (Place, error) {
	from.Draw(row, index, 0)
	space, err := SpaceAlong(maths.Vec{}, maths.YAxis)
	return Place{Space: space}, err
}

func TestGrowthKeepsDepthFirstPartsAndSeedOrder(t *testing.T) {
	skin, err := NewSleeve([]Joint{{OfGirth: 1}, {OfGirth: .5}}, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	makeLimb := func(role int) Limb {
		skin.Role = role
		limb, err := NewLimb(1, 1, 0, skin)
		if err != nil {
			t.Fatal(err)
		}
		return limb
	}
	leaf, err := NewGrowth(makeLimb(3))
	if err != nil {
		t.Fatal(err)
	}
	branch, err := NewGrowth(makeLimb(2), Carried{Where: traceArrangement{1}, What: leaf})
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewGrowth(makeLimb(1), Carried{Where: traceArrangement{2}, What: branch})
	if err != nil {
		t.Fatal(err)
	}
	space, err := SpaceAlong(maths.Vec{}, maths.YAxis)
	if err != nil {
		t.Fatal(err)
	}
	on := Place{Space: space, Size: Size{Length: measure.Metres(1), Girth: measure.Metres(.1)}}
	var log []string
	makeParts := func(limb Limb, _ Size, _ []Place) ([]int, error) { return []int{limb.Skins[0].Role}, nil }
	parts, err := Parts(root, on, traceSeed{"root", &log}, makeParts)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1, 2, 3, 2, 3}; !reflect.DeepEqual(parts, want) {
		t.Fatalf("parts order %v, want %v", parts, want)
	}
	want := []string{"piece root 0", "draw root 0", "sprout root 0", "piece root.0 0", "draw root.0 0", "sprout root.0 0", "draw root 1", "sprout root 1", "piece root.1 0", "draw root.1 0", "sprout root.1 0"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("seed order %v, want %v", log, want)
	}
	refusal := errors.New("cannot make limb")
	log = nil
	parts, err = Parts(root, on, traceSeed{"root", &log}, func(Limb, Size, []Place) ([]int, error) { return nil, refusal })
	if !errors.Is(err, refusal) || parts != nil || len(log) != 0 {
		t.Fatalf("late or partial refusal: parts %v, calls %v, err %v", parts, log, err)
	}
}
