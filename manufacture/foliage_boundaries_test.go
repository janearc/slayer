package manufacture

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
)

func TestLegacyBladesKeepTheirRanksAndUnrelatedParts(t *testing.T) {
	axis, err := newTube(measure.Metres(1), measure.Metres(.1), measure.Metres(.02))
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []int{1, 2} {
		spray := foliageSpray{envelope: axis, count: 16, seed: Sow(5), style: style}
		if skin := spray.skin(cut{around: 3}); len(skin) != 64 {
			t.Fatalf("style %d has %d blade faces", style, len(skin))
		}
		_, first, _ := spray.twig(0)
		_, second, _ := spray.twig(1)
		if style == 1 && (first != 0 || second != math.Pi) {
			t.Fatal("two-ranked legacy blades lost alternate sides")
		}
		if style == 2 && first == second {
			t.Fatal("seeded scale blades lost their spiral")
		}
	}
	untouched := part{of: leaf, shape: crownShape{maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}}}
	result := individualFoliage(good{parts: []part{untouched}}, Sow(5), 0)
	if !reflect.DeepEqual(result.parts, []part{untouched}) {
		t.Fatal("a non-tube foliage descriptor was replaced")
	}
}

func TestRecursiveFoliagePropagatesPrototypeAndBarkRefusals(t *testing.T) {
	axis, err := newTube(measure.Metres(.05), measure.Metres(.01), measure.Metres(.001))
	if err != nil {
		t.Fatal(err)
	}
	terminal := part{of: leaf, shape: axis, terminal: true, fine: cut{around: 3}}
	for _, test := range []struct {
		style int
		want  string
	}{
		{4, "unsupported shoot prototype key"},
		{1, "no faces"},
	} {
		_, err := recursiveFoliage(good{parts: []part{terminal}}, Sow(5), test.style, "quick")
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("style %d lost %q: %v", test.style, test.want, err)
		}
	}
	row := norwayRow()
	row.foliageStyle = 1
	row.grows.trunk.sides = 2
	if _, err := row.madeAt(Sow(5), "quick"); !errors.Is(err, errCut) {
		t.Fatalf("recursive tree lost scaffold cause: %v", err)
	}
}

func TestUncheckedFireSmokeKeepsItsPlumeRefusal(t *testing.T) {
	spec := burnerFixture()
	spec.Smoke = &plume.Fire{}
	_, cause := plume.New(*spec.Smoke)
	if cause == nil {
		t.Fatal("empty plume accepted")
	}
	if _, _, err := (burner{spec: spec}).Place(maths.Subspace{}, 1, 0); !errors.Is(err, cause) {
		t.Fatalf("smoke cause: %v", err)
	}
}

func TestRingAssemblyRefusesAnUncheckedNonfinitePose(t *testing.T) {
	if _, err := around(ring{count: 1, radius: measure.Metres(math.NaN())}, canRow(), Sow(5)); err == nil || !strings.Contains(err.Error(), "invalid placement") {
		t.Fatalf("nonfinite ring placement: %v", err)
	}
}
