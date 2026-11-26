//go:build darwin

package gpu

import (
	"errors"
	"image"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// These are the captured diagnostics used by sampledRegion, not a later
// change to the process environment. The refusal occurs before opening Metal.
func TestRegionRefusesCapturedDiagnostics(t *testing.T) {
	lens, err := film.PinholeOf(1)
	if err != nil {
		t.Fatal(err)
	}
	priorCounting, priorWindow := workCounting, onlyWindow
	defer func() { workCounting, onlyWindow = priorCounting, priorWindow }()
	for _, tc := range []struct {
		count  bool
		window string
	}{{true, ""}, {false, "1-2,1-2"}} {
		workCounting, onlyWindow = tc.count, tc.window
		_, _, err := SampledRegion(Software, thing.NewWorld(), maths.Subspace{}, lens,
			0, 4, 4, 1, image.Rect(1, 1, 3, 3))
		if !errors.Is(err, ErrRegionUnavailable) {
			t.Fatalf("diagnostics %+v: %v", tc, err)
		}
	}
}
