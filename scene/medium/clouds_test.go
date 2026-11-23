package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// cloudsLayer is the layer of cloud the cover test samples, with the cover
// and the seed the test chose for it.
func cloudsLayer(t *testing.T) Clouds {
	t.Helper()
	return Clouds{
		Radius: picked(t, "clouds-radius"),
		Low:    picked(t, "clouds-low"),
		High:   picked(t, "clouds-high"),
		Cover:  picked(t, "clouds-cover"),
		Thick:  picked(t, "clouds-thick"),
		Seed:   uint64(pickedCount(t, "clouds-seed")),
	}
}

// Clouds sit in their layer and nowhere else, a cover of a half clouds
// about two thirds of the sphere, as Clouds says, and they are thickest
// low in the layer. over twenty seeds a half clouds 0.64 to 0.71.
func TestCloudsCoverTheirShare(t *testing.T) {
	cloudy := cloudsLayer(t)
	below := maths.V(picked(t, "clouds-below-layer"), nought, nought)
	above := maths.V(picked(t, "clouds-above-layer"), nought, nought)
	if cloudy.Density(below) != nought || cloudy.Density(above) != nought {
		t.Error("cloud outside the layer")
	}
	samples := pickedCount(t, "clouds-samples")
	bodyHeight := picked(t, "clouds-body-height")
	clouded, sampled := nought, nought
	for place := range samples {
		direction := scatter(t, place)
		if cloudy.Density(direction.Scale(bodyHeight)) > nought {
			clouded++
		}
		sampled++
	}
	share := float64(clouded) / float64(sampled)
	shareLeast := picked(t, "clouds-share-least")
	shareMost := picked(t, "clouds-share-most")
	if share < shareLeast || share > shareMost {
		t.Errorf("%.2f of the sphere is clouded for a cover of a half", share)
	}
	// where there is cloud, the layer is thicker low than at the top
	topHeight := picked(t, "clouds-top-height")
	for place := range samples {
		direction := scatter(t, place)
		low := cloudy.Density(direction.Scale(bodyHeight))
		top := cloudy.Density(direction.Scale(topHeight))
		if low > nought && !(top < low) {
			t.Fatalf("the top of the cloud is %v, the body %v", top, low)
		}
	}
}

// scatter is the place-th of a spread of directions over the sphere, by
// hash, so a test samples the whole sphere and not one patch of it.
func scatter(t *testing.T, place int) maths.Vec {
	t.Helper()
	index := int64(place)
	seedX := uint64(pickedCount(t, "clouds-scatter-seed-x"))
	seedY := uint64(pickedCount(t, "clouds-scatter-seed-y"))
	seedZ := uint64(pickedCount(t, "clouds-scatter-seed-z"))
	return maths.V(noise.Hash(seedX, index)-halfway,
		noise.Hash(seedY, index)-halfway,
		noise.Hash(seedZ, index)-halfway).Unit()
}
