package bodies_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the helpers the worked tests share: finding a part in a world by the
// name it was given, and holding a number to one worked by hand.

// solidNamed is the thing in the world whose material has the name.
func solidNamed(t *testing.T, w *thing.World, name string) *thing.Thing {
	t.Helper()
	for _, th := range w.Things {
		if th.Material.Name == name {
			return th
		}
	}
	t.Fatalf("there is no thing named %q in the world", name)
	return nil
}

// volumeNamed is the volume in the world whose material has the name.
func volumeNamed(t *testing.T, w *thing.World, name string) *thing.Volume {
	t.Helper()
	for _, v := range w.Volumes {
		if v.Material.Name == name {
			return v
		}
	}
	t.Fatalf("there is no volume named %q in the world", name)
	return nil
}

// near holds got to want within a share of want.
func near(t *testing.T, what string, got, want, share float64) {
	t.Helper()
	if !(math.Abs(got-want) <= share*math.Abs(want)) {
		t.Errorf("%s is %v, want %v within %v of it", what, got, want, share)
	}
}

// nearVec holds got to want within a distance.
func nearVec(t *testing.T, what string, got, want maths.Vec, within float64) {
	t.Helper()
	if !(got.Sub(want).Len() <= within) {
		t.Errorf("%s is %v, want %v within %v", what, got, want, within)
	}
}

// nearLight holds each band of got to want within a distance.
func nearLight(t *testing.T, what string, got, want maths.Light, within float64) {
	t.Helper()
	if !(math.Abs(got.R-want.R) <= within && math.Abs(got.G-want.G) <= within &&
		math.Abs(got.B-want.B) <= within) {
		t.Errorf("%s is %v, want %v within %v", what, got, want, within)
	}
}

// same holds got to want, field for field, with no give: for the parts
// whose numbers are worked in halves and quarters, which a float holds
// exactly.
func same(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s is\n  %#v\nwant\n  %#v", what, got, want)
	}
}
