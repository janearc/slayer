package manufacture

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestSpecializedMakersRequireTheirSeedAndSupportedRepresentation(t *testing.T) {
	for _, make := range []func() (good, error){
		func() (good, error) { return (shootKind{style: 1}).madeAt(seed{}, "quick") },
		func() (good, error) { return (optical{spec: opticalSpec()}).madeAt(seed{}, "quick") },
	} {
		if _, err := make(); !errors.Is(err, errGood) {
			t.Fatalf("unsown seed cause: %v", err)
		}
	}
	if _, err := (shootKind{style: 4}).madeAt(Sow(1), "quick"); err == nil ||
		!strings.Contains(err.Error(), "unsupported shoot prototype key") {
		t.Fatalf("unsupported shoot style: %v", err)
	}
	if _, err := (optical{spec: opticalSpec()}).madeAt(Sow(1), "unknown"); err == nil ||
		!strings.Contains(err.Error(), `unknown complexity "unknown"`) {
		t.Fatalf("unsupported optical representation: %v", err)
	}
	row, err := KindNamed("can")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeWithComplexity(row, Sow(1), "hull"); err == nil ||
		!strings.Contains(err.Error(), "can has no hull representation") {
		t.Fatalf("unsupported simple representation: %v", err)
	}
}

func TestShootChoiceFailuresRetainTheirActualParserOrNameCause(t *testing.T) {
	for _, parse := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing name", true: "malformed data"}[parse], func(t *testing.T) {
			previous := shootNumbersError
			t.Cleanup(func() { shootNumbersError = previous })
			var cause error
			if parse {
				_, cause = numbers.Read("broken-shoot-choices.yml", []byte("numbers: ["))
				if cause == nil {
					t.Fatal("malformed choices parsed")
				}
				shootNumbersError = cause
			}
			defer func() {
				got, ok := recover().(error)
				if !ok || (parse && !errors.Is(got, cause)) ||
					(!parse && !strings.Contains(got.Error(), "absent-shoot-parameter")) {
					t.Fatalf("choice failure lost its cause: %v", got)
				}
			}()
			shootValue("absent-shoot-parameter")
		})
	}
}

// changedShootChoice models invalid authored data at a fresh startup. Tests
// remain sequential and restore both the choices and their immutable cache.
func changedShootChoice(t *testing.T, name string, value float64) {
	t.Helper()
	previous := shootNumbers
	shootNumbers = slices.Clone(shootNumbers)
	t.Cleanup(func() { shootNumbers = previous })
	number, err := shootNumbers.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	number.Value = value
	previousCache := map[any]any{}
	shootPrototypes.Range(func(key, value any) bool {
		previousCache[key] = value
		return true
	})
	shootPrototypes.Clear()
	t.Cleanup(func() {
		shootPrototypes.Clear()
		for key, value := range previousCache {
			shootPrototypes.Store(key, value)
		}
	})
}

func TestPrototypeRefusesInvalidAuthoredTubeAndNeedleGeometry(t *testing.T) {
	t.Run("woody radius", func(t *testing.T) {
		changedShootChoice(t, "shoot-base-radius", 0)
		if _, err := prototype(1, 0, 0, "quick"); !errors.Is(err, errTubeFoot) {
			t.Fatalf("zero woody radius lost its tube cause: %v", err)
		}
	})
	t.Run("needle width", func(t *testing.T) {
		changedShootChoice(t, "redwood-needle-width", 0)
		if _, err := prototype(1, 0, 0, "quick"); err == nil || !strings.Contains(err.Error(), "has no area") {
			t.Fatalf("degenerate needle lost its mesh cause: %v", err)
		}
	})
}

func TestRecursiveFoliageRequiresTerminalTubeAxes(t *testing.T) {
	box := maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}
	for _, test := range []struct {
		parts []part
		want  string
	}{
		{nil, "tree has no terminal shoots"},
		{[]part{{of: leaf, terminal: true, shape: crownShape{box}}}, "foliage needs a shoot axis"},
	} {
		if _, err := recursiveFoliage(good{parts: test.parts}, Sow(1), 1, "quick"); err == nil ||
			!strings.Contains(err.Error(), test.want) {
			t.Fatalf("invalid scaffold: want %q, got %v", test.want, err)
		}
	}
}

func TestGroupedBoundsSurviveGatheringAndPlacement(t *testing.T) {
	triangle, err := mesh.New([]maths.Vec{maths.Vec{}, maths.XAxis, maths.YAxis},
		[][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle}})
	if err != nil {
		t.Fatal(err)
	}
	made := good{groups: map[slot]*shape.MeshGroup{leaf: group},
		dressed: looks{leaf: plain("test foliage", .1, .4, .1)}}
	want := maths.Box{Min: maths.Vec{}, Max: maths.V(1, 1, 0)}
	if got := made.Bounds(); got != want {
		t.Fatalf("group bounds: %v, want %v", got, want)
	}
	made.extent = &want
	gathered, err := gather(made)
	if err != nil || gathered.Bounds() != want {
		t.Fatalf("gathered extent: %v, %v", gathered.Bounds(), err)
	}
	moved, err := gathered.placed(maths.Subspace{Origin: maths.V(10, 2, 3), Scale: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got := moved.Bounds(); got.Min != maths.V(10, 2, 3) || got.Max != maths.V(12, 4, 3) || made.Bounds() != want {
		t.Fatalf("placed extent: %v, original %v", got, made.Bounds())
	}
}

func TestFoliageGroupFailureReturnsNoPartialSpecimen(t *testing.T) {
	triangle, err := mesh.New([]maths.Vec{maths.Vec{}, maths.XAxis, maths.YAxis},
		[][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	box := maths.Box{Min: maths.Vec{}, Max: maths.V(1, 1, 0)}
	original := good{extent: &box, dressed: looks{body: plain("test wood", .2, .1, .1)}}
	for _, members := range [][2][]shape.MeshMember{
		{nil, {{Mesh: triangle}}},
		{{{Mesh: triangle}}, nil},
	} {
		got, err := original.withFoliageGroups(members[0], members[1])
		if err == nil || !strings.Contains(err.Error(), "invalid mesh group member count") {
			t.Fatalf("invalid foliage group cause: %v", err)
		}
		if got.extent != nil || got.dressed != nil || len(got.groups) != 0 {
			t.Fatal("group refusal returned a partially assembled specimen")
		}
	}
}
