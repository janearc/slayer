package medium

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestAuthoredGrovePatchNoiseAtFixedCoordinates(t *testing.T) {
	weather := GroveWeather{Extinction: .008, Depth: 8, Coverage: .6,
		Seed: 23, Wind: maths.V(2, .1, -.5), GroundFloor: -2, Fade: true}
	grove, err := NewGrove(weather, []Canopy{
		{At: maths.V(3, 0, -5), Radius: 10, Height: 20},
		{At: maths.V(-8, 0, 7), Radius: 5, Height: 12},
	}, func(x, z float64) float64 { return -.5 + .01*x - .02*z })
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open("testdata/grove-appearance.txt")
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
		var at time.Duration
		var x, y, z, want float64
		if n, err := fmt.Sscan(scan.Text(), &at, &x, &y, &z, &want); err != nil || n != 5 {
			t.Fatalf("sample %d: %v", rows, err)
		}
		got := grove.Tick(at).Density(maths.V(x, y, z))
		if !(math.Abs(got-want) <= 1e-13) {
			t.Fatalf("sample %d density %.17g, authored %.17g", rows, got, want)
		}
		rows++
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if rows != 486 {
		t.Fatalf("incomplete authored patch record: %d", rows)
	}
}
