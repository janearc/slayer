package manufacture

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type testPartMotion struct{}

func (testPartMotion) At(at time.Duration) maths.Subspace {
	return maths.Subspace{Origin: maths.V(at.Seconds(), 0, 0)}
}

type movingPartMaker struct{ source []thing.Thing }

func (maker movingPartMaker) made(seed) (good, error) { return good{native: maker.source}, nil }
func (maker movingPartMaker) madeAtTime(_ seed, _ string, _ time.Duration) (good, error) {
	return (good{native: maker.source}).placed(maths.Subspace{Origin: maths.V(0, 3, 0)})
}

func TestDeformingPartMotionSurvivesPlacementAndReplays(t *testing.T) {
	source := []thing.Thing{{Shape: shape.Sphere{Radius: 1}, Motion: testPartMotion{}}}
	row := kind{name: "moving part", how: movingPartMaker{source: source}}
	outer := maths.Subspace{Origin: maths.V(10, 0, 0), Orientation: maths.About(maths.YAxis, 1)}
	for _, at := range []time.Duration{time.Second, 2 * time.Second, time.Second} {
		made, err := MakeAtTime(row, Sow(1), "quick", at)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.things(outer)
		if err != nil {
			t.Fatal(err)
		}
		moment := at.Seconds()
		want := outer.Out(maths.V(moment, 3, 0))
		if got := parts[0].At(0).Origin; got.Sub(want).Len() > 1e-9 {
			t.Fatalf("at %v: got %v, want %v", at, got, want)
		}
	}
	if _, ok := source[0].Motion.(testPartMotion); !ok {
		t.Fatal("source motion was mutated")
	}
}

type timedTestMaker struct{ calls *[]time.Duration }

func (maker timedTestMaker) made(seed) (good, error) { return good{}, nil }
func (maker timedTestMaker) madeAtTime(_ seed, _ string, at time.Duration) (good, error) {
	*maker.calls = append(*maker.calls, at)
	return good{}, nil
}

func TestDeformingKindIsOptInAndReceivesAbsoluteTime(t *testing.T) {
	var calls []time.Duration
	row := kind{name: "timed test", how: timedTestMaker{calls: &calls}}
	if !Deforms(row) {
		t.Fatal("timed kind was not recognized")
	}
	for _, at := range []time.Duration{time.Second, 2 * time.Second, time.Second} {
		if _, err := MakeAtTime(row, Sow(9), "quick", at); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 3 || calls[0] != time.Second || calls[1] != 2*time.Second || calls[2] != time.Second {
		t.Fatalf("wrong requested times: %v", calls)
	}
	static, err := KindNamed("can")
	if err != nil {
		t.Fatal(err)
	}
	if Deforms(static) {
		t.Fatal("static can was marked deforming")
	}
	if _, err := MakeAtTime(static, Sow(9), "quick", 0); err == nil || !strings.Contains(err.Error(), "no deforming skin") {
		t.Fatalf("static kind was silently animated: %v", err)
	}
	if _, err := MakeAtTime(row, Sow(9), "unknown", 0); err == nil {
		t.Fatal("invalid complexity was accepted")
	}
}
