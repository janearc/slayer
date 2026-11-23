package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// clearCloudOrigin is the middle of space.
func clearCloudOrigin() maths.Vec { return maths.V(nought, nought, nought) }

// TestClearCloudRequiresFiniteBoundedDensityField holds that a clear cloud is
// refused a missing field, a field of no finite bounds and one wider than the
// explicit maximum, and that one it takes leaves the density as it was.
func TestClearCloudRequiresFiniteBoundedDensityField(t *testing.T) {
	if _, err := NewClearCloud(nil); err == nil {
		t.Fatal("nil field accepted")
	}
	endless := Puff{Radius: math.Inf(unity), Thick: unity}
	if _, err := NewClearCloud(endless); err == nil {
		t.Fatal("nonfinite bounds accepted")
	}
	wide := Puff{Radius: chosen.clearCloudWidth, Thick: unity}
	if _, err := NewClearCloud(wide); err == nil {
		t.Fatal("field wider than the explicit maximum accepted")
	}
	field := Puff{Radius: picked(t, "clear-cloud-field-radius"),
		Thick: picked(t, "clear-cloud-field-thick")}
	cloud, err := NewClearCloud(field)
	if err != nil {
		t.Fatal(err)
	}
	outside := maths.V(picked(t, "clear-cloud-outside-reach"), nought, nought)
	if cloud.Density(clearCloudOrigin()) != field.Thick ||
		cloud.Density(outside) != nought {
		t.Fatal("wrapper changed the density field")
	}
	if _, ok := any(cloud).(interface{ ClearMediumOverlay() }); !ok {
		t.Fatal("cloud did not carry explicit mixed-medium opt-in")
	}
}
