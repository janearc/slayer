package flame

import (
	"bufio"
	"fmt"
	"github.com/janearc/slayer/maths"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

// Fixed authored samples hold rags that the broad appearance grid misses.
func TestAuthoredSootRagsAndContinuousBorders(t *testing.T) {
	file, err := os.Open("testdata/ragged.txt")
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
		var seed uint64
		var hot bool
		var at time.Duration
		var x, y, z, density, heat float64
		if n, err := fmt.Sscan(scan.Text(), &seed, &hot, &at, &x, &y, &z, &density, &heat); err != nil || n != 8 {
			t.Fatalf("row %d: %v", rows, err)
		}
		spec := campfire
		spec.Seed = seed
		spec.HotTongues = hot
		source := body(t, spec).Tick(at).(*Body)
		p := maths.V(x, y, z)
		if got := source.Density(p); !(math.Abs(got-density) <= 1e-10) {
			t.Fatalf("row %d absorption %g, authored %g", rows, got, density)
		}
		if got := source.Heat(p).InKelvin(); !(math.Abs(got-heat) <= 1e-8) {
			t.Fatalf("row %d kelvin %g, authored %g", rows, got, heat)
		}
		rows++
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if rows != 15 {
		t.Fatalf("incomplete authored rag fixture: %d rows", rows)
	}
}
