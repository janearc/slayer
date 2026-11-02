package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Clouds sit in their layer and nowhere else, a cover of a half clouds
// about two thirds of the sphere, as Clouds says, and they are thickest
// low in the layer. over twenty seeds a half clouds 0.64 to 0.71.
func TestCloudsCoverTheirShare(t *testing.T) {
	c := Clouds{Radius: 10, Low: 0.5, High: 1.5, Cover: 0.5, Thick: 1, Seed: 3}
	if c.Density(maths.V(10.2, 0, 0)) != 0 || c.Density(maths.V(11.6, 0, 0)) != 0 {
		t.Error("cloud outside the layer")
	}
	clouded, n := 0, 0
	for i := range 400 {
		d := scatter(i)
		if c.Density(d.Scale(10.8)) > 0 {
			clouded++
		}
		n++
	}
	if share := float64(clouded) / float64(n); share < 0.55 || share > 0.8 {
		t.Errorf("%.2f of the sphere is clouded for a cover of a half", share)
	}
	// where there is cloud, the layer is thicker low than at the top
	for i := range 400 {
		d := scatter(i)
		low, top := c.Density(d.Scale(10.8)), c.Density(d.Scale(11.45))
		if low > 0 && !(top < low) {
			t.Fatalf("the top of the cloud is %v, the body %v", top, low)
		}
	}
}

// scatter is the i-th of a spread of directions over the sphere, by
// hash, so a test samples the whole sphere and not one patch of it.
func scatter(i int) maths.Vec {
	k := int64(i)
	return maths.V(noise.Hash(1, k)-0.5, noise.Hash(2, k)-0.5, noise.Hash(3, k)-0.5).Unit()
}
