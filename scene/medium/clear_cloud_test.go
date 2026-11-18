package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestClearCloudRequiresFiniteBoundedDensityField(t *testing.T) {
	if _, err := NewClearCloud(nil); err == nil {
		t.Fatal("nil field accepted")
	}
	if _, err := NewClearCloud(Puff{Radius: math.Inf(1), Thick: 1}); err == nil {
		t.Fatal("nonfinite bounds accepted")
	}
	if _, err := NewClearCloud(Puff{Radius: maxClearCloudWidth, Thick: 1}); err == nil {
		t.Fatal("field wider than the explicit maximum accepted")
	}
	cloud, err := NewClearCloud(Puff{Radius: 2, Thick: 3})
	if err != nil {
		t.Fatal(err)
	}
	if cloud.Density(maths.Vec{}) != 3 || cloud.Density(maths.V(4, 0, 0)) != 0 {
		t.Fatal("wrapper changed the density field")
	}
	if _, ok := any(cloud).(interface{ ClearMediumOverlay() }); !ok {
		t.Fatal("cloud did not carry explicit mixed-medium opt-in")
	}
}
