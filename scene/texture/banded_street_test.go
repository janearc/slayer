package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// A street of Count storms has one spot to a count, each with its
// numbers from the street's hash, worked here from noise.Hash:
//
//	Lon     (i + 0.6*hash(seed+1, i)) / Count of a turn
//	Lat     the street's, up to half its size either way, by hash(seed+2, i)
//	Size    0.7 to 1.3 times the street's, by hash(seed+3, i)
//	Stretch 1.3 to 2.3, by hash(seed+4, i)
//
// and the turn and paint are the street's own. a street of one is one
// spot, of two is two, and one of none is none.
func TestAStreetIsAsManySpotsAsItsCountWithTheirOwnNumbers(t *testing.T) {
	paint := maths.RGB(0.7, 0.3, 0.1)
	for _, count := range []int{1, 2, 3, 7} {
		st := Street{Lat: -0.4, Count: count, Size: 0.2, Turn: 1.5, Paint: paint, Seed: 5}
		got := st.spots()
		if len(got) != count {
			t.Fatalf("a street of %d has %d spots", count, len(got))
		}
		for i, s := range got {
			k := int64(i)
			lon := (float64(i) + 0.6*noise.Hash(6, k)) / float64(count) * 2 * math.Pi
			lat := -0.4 + (noise.Hash(7, k)-0.5)*0.2
			size := 0.2 * (0.7 + 0.6*noise.Hash(8, k))
			stretch := 1.3 + noise.Hash(9, k)
			if !within(s.Lon, lon, 1e-12) || !within(s.Lat, lat, 1e-12) ||
				!within(s.Size, size, 1e-12) || !within(s.Stretch, stretch, 1e-12) ||
				s.Turn != 1.5 || s.Paint != paint {
				t.Errorf("spot %d of %d is %+v; want lon %v lat %v size %v stretch %v",
					i, count, s, lon, lat, size, stretch)
			}
		}
	}
}
