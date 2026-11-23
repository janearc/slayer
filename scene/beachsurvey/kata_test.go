package beachsurvey

import (
	"math"
	"testing"
)

func TestKataMappedCoastKeepsIslandAndLand(t *testing.T) {
	needsSurveys(t)
	window, err := heights("kata-beach")
	if err != nil {
		t.Fatal(err)
	}
	across, down, step := window.Size()
	if across != 181 || down != 181 || step != 20 {
		t.Fatalf("unexpected Kata grid %d × %d at %g m", across, down, step)
	}
	const lat, lon, half = 7.816, 98.298, 1800.0
	for _, point := range []struct {
		name     string
		lat, lon float64
		land     bool
	}{
		{"Ko Pu", 7.8198, 98.2862, true},
		{"offshore", 7.816, 98.2828, false},
		{"mainland", 7.816, 98.306, true},
		{"concave mainland A", 7.81330204, 98.29618449, true},
		{"concave mainland B", 7.81294231, 98.29636604, true},
		{"concave mainland C", 7.81276244, 98.29654759, true},
	} {
		east := (point.lon - lon) * 111195 * math.Cos(lat*math.Pi/180)
		north := (point.lat - lat) * 111195
		x, y := int(math.Round((east+half)/step)), int(math.Round((half-north)/step))
		if got := window.At(x, y) > 0; got != point.land {
			t.Errorf("%s: land %v, want %v", point.name, got, point.land)
		}
	}
}
