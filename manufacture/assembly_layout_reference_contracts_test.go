package manufacture

import (
	"bufio"
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"math"
	"os"
	"testing"
)

// This holds authored placement and phase choices, not a combustion model.
func TestAuthoredFireLayoutsKeepTheirRecordedSources(t *testing.T) {
	fields := map[string]*flame.Field{}
	for _, pattern := range []string{"cluster", "line"} {
		spec := burnerFixture()
		spec.Smoke = nil
		spec.Layout = FireLayout{Pattern: pattern, Count: 3, Variation: .3}
		if pattern == "cluster" {
			spec.Layout.Radius = measure.Metres(.8)
		} else {
			spec.Layout.Length = measure.Metres(4)
		}
		made, err := Burn(spec)
		if err != nil {
			t.Fatal(err)
		}
		volumes, _, err := made.Place(maths.Subspace{}, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		field, ok := volumes[0].Medium.(*flame.Field)
		if !ok {
			t.Fatal("missing source field")
		}
		fields[pattern] = field
	}
	file, err := os.Open("testdata/fire-layout-reference.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	rows := 0
	seen := map[string]bool{}
	for scan.Scan() {
		var pattern string
		var index int
		var seed uint64
		var x, y, z, height float64
		if n, err := fmt.Sscan(scan.Text(), &pattern, &index, &seed, &x, &y, &z, &height); err != nil || n != 7 {
			t.Fatalf("invalid source row %q: %v", scan.Text(), err)
		}
		field := fields[pattern]
		if field == nil || index < 0 || index >= 3 || seen[fmt.Sprintf("%s%d", pattern, index)] {
			t.Fatal("invalid/duplicate layout reference")
		}
		seen[fmt.Sprintf("%s%d", pattern, index)] = true
		piece := field.Pieces()[index]
		description, _, _, _ := piece.Body.Samples()
		if description.Seed != seed || !(piece.At.Sub(maths.V(x, y, z)).Len() <= 1e-12) || !(math.Abs(description.Height-height) <= 1e-12) {
			t.Fatalf("%s patch %d changed authored placement, phase or height", pattern, index)
		}
		rows++
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if rows != 6 {
		t.Fatalf("incomplete source reference: %d", rows)
	}
}
