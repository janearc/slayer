package ocean

import "testing"

func TestSkyProjectionExposesActualNormalization(t *testing.T) {
	config := DefaultConfig()
	image := RGB{Width: 1, Height: 1, Light: []float32{1, 1, 1}}
	sky, err := NewSky(config, image, nil)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := sky.Projection()
	if err != nil {
		t.Fatal(err)
	}
	if projection.Focal != sky.focal || projection.Normalization != sky.disc || projection.Forward != sky.forward || projection.Right != sky.right || projection.Up != sky.up {
		t.Fatalf("projection: %+v", projection)
	}
}

func TestNilSkyProjectionIsRefused(t *testing.T) {
	var sky *Sky
	if _, err := sky.Projection(); err == nil {
		t.Fatal("nil sky accepted")
	}
}
