package embers

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

// This records authored attribute packing and release appearance. It is
// separate from the published finalizer vectors and the RK4 force oracle.
func TestAuthoredParticleAttributesRemainStable(t *testing.T) {
	spec := fixture()
	spec.Rate = 2
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	particles := source.At(4125 * time.Millisecond)
	file, err := os.Open("testdata/appearance.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	row := 0
	for scan.Scan() {
		if strings.HasPrefix(scan.Text(), "#") {
			continue
		}
		var id int64
		var age time.Duration
		var x, y, z, heat float64
		if n, err := fmt.Sscan(scan.Text(), &id, &age, &x, &y, &z, &heat); err != nil || n != 6 {
			t.Fatalf("sample %d: %v", row, err)
		}
		if row >= len(particles) {
			t.Fatal("release lost a recorded particle")
		}
		p := particles[row]
		if p.ID() != id || p.Age() != age || !(p.Position().Sub(maths.V(x, y, z)).Len() <= 1e-10) || !(math.Abs(p.Heat().InKelvin()-heat) <= 1e-8) {
			t.Fatalf("release sample %d changed: %+v", row, p)
		}
		row++
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if row != len(particles) || row < 5 {
		t.Fatalf("incomplete release fixture %d/%d", row, len(particles))
	}
}
