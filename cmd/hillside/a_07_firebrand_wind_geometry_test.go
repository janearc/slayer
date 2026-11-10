package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Cardinal source bearings have exact horizontal downwind vectors. Compare
// actual staged particle poses with the public source model; its force law
// is independently checked by the embers package's RK4 regression.
func TestFrontFirebrandsUseHorizontalWindFromTheSpecifiedBearing(t *testing.T) {
	for _, sample := range []struct {
		from  int
		drift maths.Vec
	}{
		{0, maths.V(0, 0, 5)}, {90, maths.V(-5, 0, 0)},
		{180, maths.V(0, 0, -5)}, {270, maths.V(5, 0, 0)},
	} {
		t.Run(fmt.Sprintf("from%d", sample.from), func(t *testing.T) {
			given := frontOnFixture(t)
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			night, err := os.ReadFile("../../doc/fire-night.yaml")
			if err != nil {
				t.Fatal(err)
			}
			start := strings.Index(string(night), "      sparks:\n")
			if start < 0 {
				t.Fatal("night scene lost sparks")
			}
			end := strings.Index(string(night[start:]), "      light-cells:")
			if end < 0 {
				t.Fatal("night scene lost following light-cells")
			}
			block := string(night[start : start+end])
			text := strings.Replace(string(raw), "      flame:", block+"      flame:", 1)
			text = strings.Replace(text, "kind: fern, count: 4", "kind: fern, count: 1", 1)
			weather := fmt.Sprintf("  weather:\n    wind:\n      from: {value: %d, unit: deg}\n      speed: {value: 5, unit: m/s}\n", sample.from)
			text = strings.Replace(text, "  film:", weather+"  film:", 1)
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			read := readBoundarySet(t, given.set)
			from, speed := read.Wind()
			if from != float64(sample.from) || speed != 5 {
				t.Fatal("fixture wind was not read")
			}
			given.exposure = 1
			stage := staging{given: given, recipes: read, world: thing.NewWorld(),
				places: map[string]maths.Vec{"the tree": {}},
				fuels:  map[string][]groveTree{"front fuel": {{at: maths.V(10, 2, 5)}}}}
			if err := stage.prepareFronts(read); err != nil {
				t.Fatal(err)
			}
			if len(stage.fronts) != 1 || len(stage.fronts[0].sparks) != 1 {
				t.Fatal("front lost target source")
			}
			front := stage.fronts[0]
			shape := front.sparks[0].sparks.shape
			if shape == nil || shape.FaceCount() != 8 || !shape.Closed() {
				t.Fatal("firebrand needs a real closed eight-face shared mesh")
			}
			// This ray crosses the interior of x+y+z=1 at z=.625.
			// It checks actual geometry independently of pointer sharing.
			ray := maths.Ray{At: maths.V(.125, .25, 2), Dir: maths.V(0, 0, -1)}
			if distance, hit := shape.Hit(ray, 0, 4); !hit || !(math.Abs(distance-1.375) <= 1e-12) {
				t.Fatalf("firebrand mesh lost its real surface: hit=%v distance=%g", hit, distance)
			}
			groups := front.plan.Groups()
			if len(groups) != 1 || len(groups[0]) != 1 {
				t.Fatal("fixture lost single planned target")
			}
			planned := groups[0][0]
			row, has := read.Recipe("wood fire")
			if !has {
				t.Fatal("fixture lost recipe")
			}
			spec, has := row.Fire()
			if !has || spec.Sparks == nil {
				t.Fatal("fixture lost sparks")
			}
			referenceSpec := *spec.Sparks
			referenceSpec.Seed = planned.Seed
			referenceSpec.Air = referenceSpec.Air.Add(sample.drift)
			reference, err := embers.New(referenceSpec)
			if err != nil {
				t.Fatal(err)
			}
			at := 1500 * time.Millisecond
			want := reference.At(at)
			got := front.sparks[0].sparks.thingsIn(at, front.sparks[0].duration)
			if len(want) == 0 || len(got) != len(want) {
				t.Fatalf("living particles: got%d want%d", len(got), len(want))
			}
			for i, particle := range want {
				if got[i].Material.Colour != maths.Grey(.03) {
					t.Fatalf("firebrand lost its authored .03 surface reflectance: %v", got[i].Material.Colour)
				}
				position := got[i].At(0).Origin
				expected := planned.At.Add(particle.Position())
				if distance := position.Sub(expected).Len(); !(distance < 1e-9) {
					t.Fatalf("particle%d moved %g m from horizontal-wind reference", particle.ID(), distance)
				}
			}
			// The front replaces its prototype source for each target. Also
			// check the constructor used directly by ordinary fire recipes.
			baseSpec := spec
			baseSpec.Flame.Seed = 14340413156236405150
			base, err := stage.prepareSparks(baseSpec, maths.Subspace{Origin: planned.At})
			if err != nil {
				t.Fatal(err)
			}
			if base.shape == nil || base.shape != shape || stage.firebrandShape != shape {
				t.Fatal("ordinary source did not reuse the actual shared geometry")
			}
			referenceSpec.Seed = 14340413156236405150
			reference, err = embers.New(referenceSpec)
			if err != nil {
				t.Fatal(err)
			}
			want, got = reference.At(at), base.thingsAt(at)
			if len(want) == 0 || len(got) != len(want) {
				t.Fatal("ordinary source lost its living particles")
			}
			for i, particle := range want {
				expected := planned.At.Add(particle.Position())
				if gap := got[i].At(0).Origin.Sub(expected).Len(); !(gap < 1e-9) {
					t.Fatalf("ordinary particle %d moved %g m from horizontal wind", particle.ID(), gap)
				}
			}
		})
	}
}
