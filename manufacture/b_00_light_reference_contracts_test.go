package manufacture_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

type authoredLightRecord struct {
	Kind                         string
	Seed                         uint64
	Pose, Index                  int
	At, Toward                   maths.Vec
	Beyond, Strength, Half, Soft float64
	Shares                       []float64
	First, Last                  int
	Power                        []float64
}

// These are picked lamp specifications, not measured sources. Physical
// emission and source scaling are checked by the independent integral tests.
func TestAuthoredLampSourcesKeepTheirPlacementBeamSpectrumAndFlicker(t *testing.T) {
	data, err := os.ReadFile("testdata/light-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []authoredLightRecord
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 18 {
		t.Fatalf("incomplete authored source record: %d", len(rows))
	}
	poses := []maths.Subspace{{},
		{Origin: maths.V(3, -2, 7), Scale: .5, Orientation: maths.Rotation(maths.V(1, 2, 3), .7)},
		{Origin: maths.V(-4, 5, -6), Scale: 3, Orientation: maths.Rotation(maths.V(2, -1, 1), -.4)}}
	close := func(label string, got, want float64) {
		t.Helper()
		if !(math.Abs(got-want) <= 1e-12*math.Max(math.Abs(want), 1e-100)) {
			t.Fatalf("%s %.17g, authored %.17g", label, got, want)
		}
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Pose < 0 || row.Pose >= len(poses) || row.Index != 0 || len(row.Shares) != 5 {
			t.Fatal("invalid source record")
		}
		key := fmt.Sprintf("%s/%d/%d", row.Kind, row.Seed, row.Pose)
		if seen[key] {
			t.Fatalf("duplicate source %s", key)
		}
		seen[key] = true
		kind, err := manufacture.KindNamed(row.Kind)
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.Make(kind, manufacture.Sow(row.Seed))
		if err != nil {
			t.Fatal(err)
		}
		lights, err := made.Lights(poses[row.Pose])
		if err != nil {
			t.Fatal(err)
		}
		if len(lights) != 1 {
			t.Fatalf("%s has %d lights", row.Kind, len(lights))
		}
		source := lights[0]
		if !(source.At().Sub(row.At).Len() <= 1e-12) ||
			!(source.Toward().Sub(row.Toward).Len() <= 1e-12) {
			t.Fatalf("%s placement changed", key)
		}
		light := source.Light()
		half, soft := light.Angles()
		close(key+" exclusion", source.Beyond().InMetres(), row.Beyond)
		close(key+" strength", light.Strength(), row.Strength)
		close(key+" half-angle", half.InRadians(), row.Half)
		close(key+" soft-edge", soft.InRadians(), row.Soft)
		for i, at := range []time.Duration{0, time.Second / 3, time.Second, 7 * time.Second, 13 * time.Second} {
			close(key+" time share", light.Share(at), row.Shares[i])
		}
		spectrum := light.Bands()
		first, last := spectrum.Span()
		if int(first) != row.First || int(last) != row.Last || len(row.Power) != int(last-first+1) {
			t.Fatalf("%s spectral span changed", key)
		}
		for band := first; band <= last; band++ {
			close(key+" band power", spectrum.Power(band), row.Power[int(band-first)])
		}
	}
	for _, kind := range []string{"campfire", "spotlight", "maser"} {
		for _, seed := range []uint64{5, 11} {
			for pose := range poses {
				if !seen[fmt.Sprintf("%s/%d/%d", kind, seed, pose)] {
					t.Fatal("missing source case")
				}
			}
		}
	}
}

// A manufactured bonfire is the authored log pile. Its name alone does not
// add fire, emission or an implicit lamp.
func TestManufacturedLogPyresHaveNoImplicitAttachedLamp(t *testing.T) {
	row, err := manufacture.KindNamed("bonfire")
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []uint64{5, 11} {
		made, err := manufacture.Make(row, manufacture.Sow(seed))
		if err != nil {
			t.Fatal(err)
		}
		lights, err := made.Lights(maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		if len(lights) != 0 {
			t.Fatal("unlit log pile acquired a lamp")
		}
	}
}
