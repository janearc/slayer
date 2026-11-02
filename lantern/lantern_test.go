package lantern

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// covered is the share of directions out from the star that meet a panel
// before the shell's far side: how much of its light the swarm catches
func covered(t *testing.T, orbits, panels int, wide float64,
	samples int) float64 {
	var rings []*mesh.Mesh
	for number := range orbits {
		ring, err := ringMesh(orbitOf(number, orbits), number, panels,
			shell, wide, 0)
		if err != nil {
			t.Fatal(err)
		}
		rings = append(rings, ring)
	}
	caught := 0
	for index := range samples {
		if meets(rings, spread(index, samples)) {
			caught++
		}
	}
	return float64(caught) / float64(samples)
}

// spread is one of count directions spread evenly over the sphere
func spread(index, count int) maths.Vec {
	height := 1 - 2*(float64(index)+0.5)/float64(count)
	turn := float64(index) * math.Pi * (3 - math.Sqrt(5))
	flat := math.Sqrt(1 - height*height)
	return maths.V(flat*math.Cos(turn), height, flat*math.Sin(turn))
}

// meets is whether a ray out from the middle meets any ring
func meets(rings []*mesh.Mesh, way maths.Vec) bool {
	ray := maths.Ray{Dir: way}
	for _, ring := range rings {
		if _, hit := ring.Hit(ray, 0, 2*shell); hit {
			return true
		}
	}
	return false
}

// the swarm catches most of its star's light and lets some through:
// that is the lantern
func TestTheSwarmCatchesMostButNotAll(t *testing.T) {
	share := covered(t, 60, 120, wide, 20000)
	t.Logf("sixty rings catch %.1f%% of the light", 100*share)
	if share < 0.8 || share > 0.95 {
		t.Errorf("the swarm catches %.1f%%, want most but not all",
			100*share)
	}
}

func TestCoverageByPanels(t *testing.T) {
	if testing.Short() {
		t.Skip("a measure, not a check")
	}
	for _, wide := range []float64{0.2, 0.3, 0.4, 0.5, 0.6} {
		t.Logf("bands %.1f wide: %.1f%%", wide,
			100*covered(t, 60, 120, wide, 8000))
	}
}

// a ring's panels go round it without moving its pole; a tumble moves
// the pole, so the whole band turns end over end round the star
func TestATumbleTurnsTheBandAndASpinDoesNot(t *testing.T) {
	ring := orbitOf(3, 12)
	turning := spin{pole: ring.pole, turn: 10 * time.Second}
	later := turning.At(3 * time.Second).Orientation.Out(ring.pole)
	if later.Sub(ring.pole).Len() > 1e-9 {
		t.Errorf("a spin alone moved the pole to %v", later)
	}
	turning.across = ring.tumbleAxis(3)
	turning.tumble = 12 * time.Second
	later = turning.At(3 * time.Second).Orientation.Out(ring.pole)
	if math.Abs(later.Dot(ring.pole)) > 1e-9 {
		t.Errorf("a quarter tumble left the pole at %v, want it "+
			"turned square to where it was", later)
	}
}
