package searching

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestCatalogueAxes checks three independently specified cardinal directions.
func TestCatalogueAxes(t *testing.T) {
	for _, example := range []struct {
		lon, lat float64
		want     maths.Vec
	}{
		{0, 0, maths.V(2, 0, 0)}, {90, 0, maths.V(0, 0, -2)}, {0, 90, maths.V(0, 2, 0)},
	} {
		got := (cataloguePoint{Longitude: example.lon, Latitude: example.lat, Parsecs: 2}).position()
		if got.Sub(example.want).Len() > 1e-12 {
			t.Fatalf("position %v, want %v", got, example.want)
		}
	}
}

// TestCatalogueTravel checks the geometric midpoint and held endpoints,
// and proves no state is accumulated when frames are visited out of order.
func TestCatalogueTravel(t *testing.T) {
	plan := CataloguePlan{Near: 1, Far: 10000}
	for _, example := range []struct{ at, want float64 }{{1, 10000}, {.5, 100}, {0, 1}, {2, 10000}, {-1, 1}, {.5, 100}} {
		if math.Abs(plan.distance(example.at)-example.want) > 1e-8 {
			t.Fatalf("travel at %g", example.at)
		}
	}
}

// TestCatalogueRefusesBadPositions keeps corrupt data out of Metal.
func TestCatalogueRefusesBadPositions(t *testing.T) {
	data := catalogue{Kind: "gaia-dr3-display", Source: "local", SHA: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Adapter: "test", Method: "fixture", Points: []cataloguePoint{{ID: "a", Epoch: 2016, Parsecs: 10}}}
	if err := data.check(); err != nil {
		t.Fatal(err)
	}
	data.Points = append(data.Points, data.Points[0])
	if data.check() == nil {
		t.Fatal("duplicate catalogue identity accepted")
	}
	data.Points = data.Points[:1]
	data.Points[0].Parsecs = math.NaN()
	if data.check() == nil {
		t.Fatal("non-finite distance accepted")
	}
}
