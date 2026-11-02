package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// placed is a chart with its tiles at the given points.
func placed(at []maths.Vec, pairs ...[2]int) *chart {
	c := chained(len(at), pairs...)
	copy(c.at, at)
	return c
}

// two plates, a tile of each, at +X and +Z. plate 0 turns about -Y and
// plate 1 stands still. the ground at +X moves as the axis crossed with
// its place, (0,-1,0) x (1,0,0) = (0,0,1), toward the other plate: they
// close at (0,0,1) . (-1,0,1) / sqrt 2 = 1 / sqrt 2, and so from +Z.
//
// turning about +Y the ground at +X moves toward -Z, away, and they part
// at the same rate. each tile takes the effect meeting gives, unfaded.
func TestPlatesCloseOrPartBySpin(t *testing.T) {
	at := []maths.Vec{maths.V(1, 0, 0), maths.V(0, 0, 1)}
	w := on(placed(at, [2]int{0, 1}), 1)
	plate := []int{0, 1}
	r := 1 / math.Sqrt2
	for _, c := range []struct {
		name  string
		drift [2]maths.Vec
		land  [2]bool
		want  [2]float64
	}{
		{"a range", [2]maths.Vec{maths.V(0, -1, 0), {}}, [2]bool{true, true},
			[2]float64{1.1 * r, 1.1 * r}},
		{"a coast and its trench", [2]maths.Vec{maths.V(0, -1, 0), {}},
			[2]bool{true, false}, [2]float64{0.8 * r, -0.6 * r}},
		{"an island arc", [2]maths.Vec{maths.V(0, -1, 0), {}},
			[2]bool{false, false}, [2]float64{0.5 * r, 0.5 * r}},
		{"a range, both turning", [2]maths.Vec{maths.V(0, -2, 0), maths.V(0, -1, 0)},
			[2]bool{true, true}, [2]float64{1.1 * r, 1.1 * r}},
		{"a rift", [2]maths.Vec{maths.V(0, 1, 0), {}}, [2]bool{true, true},
			[2]float64{-0.35 * r, -0.35 * r}},
		{"a ridge", [2]maths.Vec{maths.V(0, 1, 0), {}}, [2]bool{true, false},
			[2]float64{0.12 * r, 0.12 * r}},
	} {
		got := w.boundaries(plate, c.land[:], c.drift[:])
		for i := range got {
			if !close12(got[i], c.want[i]) {
				t.Errorf("%s: tile %d is %v, want %v", c.name, i, got[i], c.want[i])
			}
		}
	}
}

// a tile between two plates takes the stronger of the two effects. B is
// at (0,0,1) on plate 1, which stands still; A at (1,0,1) and C at
// (-1,0,1) are plates 0 and 2, next to B and not to each other. at B a
// plate turning about (0,y,0) moves its ground by (y,0,0).
//
// so A turning at y = -6 closes on B at 6 along (1,0,0), and C turning
// at y = 5 closes on B at 5 along (-1,0,0). A and B both ocean is an arc
// of 0.5 * 6 = 3. C continent over ocean B is a trench of -0.6 * 5 = -3,
// equal and opposite, and B keeps the first it meets, A's.
func TestAGroundTakesTheStrongerEffect(t *testing.T) {
	at := []maths.Vec{maths.V(0, 0, 1), maths.V(1, 0, 1), maths.V(-1, 0, 1)}
	w := on(placed(at, [2]int{0, 1}, [2]int{0, 2}), 1)
	plate := []int{1, 0, 2}
	land := []bool{false, false, true}
	for _, c := range []struct {
		name   string
		d0, d2 float64 // the turning of A's plate and C's about y
		want   float64
	}{
		{"a tie keeps the first", -6, 5, 3},
		{"the second, stronger", -6, 10, -6},
		{"the first, stronger", -12, 5, 6},
	} {
		drift := []maths.Vec{maths.V(0, c.d0, 0), {}, maths.V(0, c.d2, 0)}
		got := w.boundaries(plate, land, drift)
		if !close12(got[0], c.want) {
			t.Errorf("%s: B is %v, want %v", c.name, got[0], c.want)
		}
	}
}

// the effect of a boundary fades inland: T0 of plate 0 is next to T1 of
// plate 1, and T2 of plate 1 is beyond it, next to T1 only. three tiles
// is a world about half a tile across, so the fall is the least it can
// be, 1.5 tiles, and T2 is one in: the effect times e to the -1 / 1.5.
func TestABoundaryFadesInland(t *testing.T) {
	at := []maths.Vec{maths.V(1, 0, 0), maths.V(0, 0, 1), maths.V(0, 1, 1)}
	w := on(placed(at, [2]int{0, 1}, [2]int{1, 2}), 1)
	plate := []int{0, 1, 1}
	land := []bool{true, true}
	drift := []maths.Vec{maths.V(0, -1, 0), {}}
	var got []float64
	ends(t, "boundaries", func() { got = w.boundaries(plate, land, drift) })
	r := 1.1 / math.Sqrt2
	want := []float64{r, r, r * math.Exp(-1/1.5)}
	for i := range want {
		if !close12(got[i], want[i]) {
			t.Errorf("tile %d is %v, want %v", i, got[i], want[i])
		}
	}
}
