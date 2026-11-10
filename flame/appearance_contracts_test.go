package flame

import (
	"bufio"
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

// These samples hold deliberately authored shape and noise choices. They are
// an appearance regression, not a measurement or a physical validation. The
// independent thermal/transport checks live in the other contract tests.
func TestAuthoredFlameAppearanceAtFixedPoints(t *testing.T) {
	file, err := os.Open("testdata/appearance.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	rows := 0
	for scan.Scan() {
		if strings.HasPrefix(scan.Text(), "#") {
			continue
		}
		var radius, x, y, z, density, heat float64
		var hot bool
		var at time.Duration
		if n, err := fmt.Sscan(scan.Text(), &radius, &hot, &at, &x, &y, &z, &density, &heat); err != nil || n != 8 {
			t.Fatalf("sample %d: %v", rows, err)
		}
		spec := Fire{Radius: radius, Height: radius * 2.4, Absorption: .8, Hottest: measure.Kelvin(1100), Edge: measure.Kelvin(800), Seed: 7, HotTongues: hot, Intermittency: .2}
		source, err := NewBody(spec, 1)
		if err != nil {
			t.Fatal(err)
		}
		source = source.Tick(at).(*Body)
		point := maths.V(x, y, z)
		if got := source.Density(point); !(math.Abs(got-density) <= 1e-10) {
			t.Fatalf("sample %d absorption %g, recorded %g", rows, got, density)
		}
		if got := source.Heat(point).InKelvin(); !(math.Abs(got-heat) <= 1e-8) {
			t.Fatalf("sample %d heat %g, recorded %g", rows, got, heat)
		}
		rows++
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if rows != 120 {
		t.Fatalf("incomplete appearance fixture: %d samples", rows)
	}
}

func TestContinuousFlameOccupancyIncludesItsBorder(t *testing.T) {
	source := body(t, campfire)
	for i := range 32 {
		point := maths.V(float64(i)*.01, .05, -.03)
		for _, up := range []float64{0, source.tonguesStart()} {
			if got := source.tongued(point, up); got != 1 {
				t.Fatalf("continuous occupancy at %g is %g", up, got)
			}
		}
	}
}
