package manufacture

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/geom"
	"github.com/janearc/slayer/maths"
)

func TestAssemblyPreservesLimbAndFormRefusals(t *testing.T) {
	skin, err := sleeved([]joint{{ofGirth: 1}, {ofGirth: .2}}, 4, body)
	if err != nil {
		t.Fatal(err)
	}
	for _, make := range []func() (growth, error){
		func() (growth, error) { return stemCarrying(0, 0, []sleeve{skin}, ring{}, 1, 0, []sleeve{skin}) },
		func() (growth, error) { return stemCarrying(1, 0, []sleeve{skin}, ring{}, 0, 0, []sleeve{skin}) },
		func() (growth, error) { return trunkCarrying(sleeve{}, 1, 1, spruceHabit(), growth{}) },
		func() (growth, error) { return (whorled{}).branching(skin, skin, 1, atQuick) },
	} {
		if _, err := make(); !errors.Is(err, errLimb) {
			t.Fatalf("assembly lost the invalid-limb cause: %v", err)
		}
	}
	if _, err := shaped(nil, leastAround, nil); !errors.Is(err, errGood) {
		t.Fatalf("missing form cause: %v", err)
	}
	if _, err := boxes([]block{{wide: 0, tall: 1, deep: 1}}); !errors.Is(err, errBox) {
		t.Fatalf("invalid box cause: %v", err)
	}
}

func TestRowAssembliesRefuseInvalidCountsAndGlassSizes(t *testing.T) {
	pyre := pyreRow()
	pyre.logs = span{}
	if _, err := pyre.made(Sow(1)); !errors.Is(err, errRing) {
		t.Fatalf("empty pyre ring cause: %v", err)
	}
	for _, field := range []string{"walls", "panes"} {
		row := homeRow()
		if field == "walls" {
			row.wide = span{}
		} else {
			row.storey = span{.2, .2}
		}
		if _, err := row.made(Sow(1)); !errors.Is(err, errBox) {
			t.Fatalf("invalid %s lost its box cause: %v", field, err)
		}
	}
}

func TestUncheckedAssemblyDirectionsReturnTheRunRefusal(t *testing.T) {
	where := whorls{count: 2, lowest: 2, set: spruceHabit()}
	where.set.low = measure.Radians(math.NaN())
	if _, err := where.at(size{length: measure.Metres(1), girth: measure.Metres(.1)}, 0, 0, Sow(1)); !errors.Is(err, errNoRun) {
		t.Fatalf("unchecked whorl direction cause: %v", err)
	}
	spec := opticalSpec()
	spec.Altitude = measure.Radians(math.NaN())
	if _, err := (optical{spec: spec}).madeAt(Sow(1), "quick"); !errors.Is(err, errNoRun) {
		t.Fatalf("unchecked optical direction cause: %v", err)
	}
	stand := stalk{tall: measure.Metres(1), thick: measure.Metres(.01)}
	head := horn{long: measure.Metres(.2), back: measure.Metres(.05), front: measure.Metres(.1)}
	if _, err := headed(stand, head, measure.Radians(math.NaN()), 6, nil); !errors.Is(err, errNoRun) {
		t.Fatalf("unchecked lamp direction cause: %v", err)
	}
	if _, err := headed(stand, head, measure.Radians(0), 2, nil); !errors.Is(err, errCut) {
		t.Fatalf("lamp tessellation cause: %v", err)
	}
}

func TestFireAssemblyPreservesSourceAndExtentRefusals(t *testing.T) {
	spec := burnerFixture()
	spec.Sparks = &embers.Spec{}
	_, cause := embers.New(*spec.Sparks)
	if cause == nil {
		t.Fatal("empty spark source accepted")
	}
	if _, err := Burn(spec); err == nil || err.Error() != cause.Error() {
		t.Fatalf("spark cause lost: %v", err)
	}
	spec = burnerFixture()
	spec.Smoke.Top = 1e200
	if _, err := Burn(spec); err == nil || !strings.Contains(err.Error(), "nonfinite fire extent") {
		t.Fatalf("unrepresentable fire extent: %v", err)
	}
	spec = burnerFixture()
	spec.Flame.Absorption = -1
	_, cause = flame.NewBody(spec.Flame, 1)
	if cause == nil {
		t.Fatal("negative absorption accepted")
	}
	if _, _, err := (burner{spec: spec}).Place(maths.Subspace{}, 1, 0); !errors.Is(err, cause) {
		t.Fatalf("unchecked recipe lost its flame cause: %v", err)
	}
	spec = burnerFixture()
	spec.Layout = FireLayout{Variation: .1}
	if _, err := sourcePatches(spec); err == nil || !strings.Contains(err.Error(), "single fire takes no layout parameters") {
		t.Fatalf("ignored single-fire layout setting: %v", err)
	}
}

func TestRecursiveConiferStopsBeyondItsRequestedDepth(t *testing.T) {
	if got := coniferRecursive(geom.Vec3{}, ConiferParams{Depth: 1}, 2); len(got) != 0 {
		t.Fatalf("out-of-depth call made %d triangles", len(got))
	}
}

func TestFernAssemblyReturnsTheFirstRecipeRefusal(t *testing.T) {
	if _, err := fernGrowth(0, 0); !errors.Is(err, errWhorls) {
		t.Fatalf("leaflet refusal must precede the later ring refusal: %v", err)
	}
}
