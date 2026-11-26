package manufacture

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

var errDispatchExtent = errors.New("specialized extent refusal")
var errDispatchSight = errors.New("specialized sight refusal")

type dispatchSpecialist struct{ madeCalls, extentCalls, sightCalls *int }

func (row dispatchSpecialist) made(seed) (good, error) { (*row.madeCalls)++; return good{}, nil }
func (row dispatchSpecialist) extent(seed) (maths.Box, error) {
	(*row.extentCalls)++
	return maths.Box{}, errDispatchExtent
}
func (row dispatchSpecialist) seen(seed, maths.Subspace, strategies.Eye) (strategies.Seen, error) {
	(*row.sightCalls)++
	return strategies.Seen{}, errDispatchSight
}

type dispatchBoundOnly struct{ madeCalls, extentCalls *int }

func (row dispatchBoundOnly) made(seed) (good, error) { (*row.madeCalls)++; return good{}, nil }
func (row dispatchBoundOnly) extent(seed) (maths.Box, error) {
	(*row.extentCalls)++
	return maths.Box{}, errDispatchExtent
}

func TestExtentDispatchPreservesSpecializedMethodsAndErrorOrder(t *testing.T) {
	made, extent, sight := 0, 0, 0
	row := kind{name: "specialized", how: dispatchSpecialist{&made, &extent, &sight}}
	if _, err := extentOf(row, seed{}); !errors.Is(err, errGood) {
		t.Fatalf("unsown extent: %v", err)
	}
	if _, err := seenOf(row, seed{}, maths.Local(), strategies.Eye{}); !errors.Is(err, errGood) {
		t.Fatalf("unsown sight: %v", err)
	}
	if _, err := extentOf(kind{}, seed{}); !errors.Is(err, errKind) {
		t.Fatalf("missing maker precedence: %v", err)
	}
	if made != 0 || extent != 0 || sight != 0 {
		t.Fatal("unsown/missing recipe was invoked")
	}
	if _, err := extentOf(row, sow(1)); !errors.Is(err, errDispatchExtent) {
		t.Fatalf("lost specialized extent: %v", err)
	}
	// A specialized sight owns its own eye handling; dispatch must not add an
	// earlier check that would replace its distinct refusal.
	if _, err := seenOf(row, sow(1), maths.Local(), strategies.Eye{}); !errors.Is(err, errDispatchSight) {
		t.Fatalf("lost specialized sight: %v", err)
	}
	if made != 0 || extent != 1 || sight != 1 {
		t.Fatalf("calls made=%d extent=%d sight=%d", made, extent, sight)
	}
}

func TestDefaultSightChecksEyeBeforeSpecializedExtent(t *testing.T) {
	made, extent := 0, 0
	row := kind{name: "bound-only", how: dispatchBoundOnly{&made, &extent}}
	if _, err := seenOf(row, sow(1), maths.Local(), strategies.Eye{}); !errors.Is(err, strategies.ErrEye) {
		t.Fatalf("blind eye: %v", err)
	}
	if made != 0 || extent != 0 {
		t.Fatal("blind eye evaluated the recipe")
	}
	if _, err := seenOf(row, sow(1), maths.Local(), lookingDown()); !errors.Is(err, errDispatchExtent) {
		t.Fatalf("lost extent refusal: %v", err)
	}
	if made != 0 || extent != 1 {
		t.Fatalf("calls made=%d extent=%d", made, extent)
	}
}

// A pointer had the same value-receiver object methods before dispatch was
// centralized. It remains explicitly declared, with the same refusals.
func TestDeclaredGreedyPointerKeepsValueBehavior(t *testing.T) {
	value, pointer := can{}, &can{}
	if !declaredGreedy(value) || !declaredGreedy(pointer) {
		t.Fatal("value or pointer lost its declaration")
	}
	_, valueErr := makerExtent(value, seed{})
	_, pointerErr := makerExtent(pointer, seed{})
	if !errors.Is(valueErr, geometry.ErrTubeLength) || !errors.Is(pointerErr, geometry.ErrTubeLength) || valueErr.Error() != pointerErr.Error() {
		t.Fatalf("unsown value=%v pointer=%v", valueErr, pointerErr)
	}
}
