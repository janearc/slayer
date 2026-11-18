package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func spectralPlane(t *testing.T) *shape.HeightField {
	t.Helper()
	field, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	slopes := make([]float64, 9)
	for index := range slopes {
		slopes[index] = 0.25
	}
	field, err = field.WithApproximateSpectralNormals(slopes,
		make([]float64, 9))
	if err != nil {
		t.Fatal(err)
	}
	return field
}

func TestMetalEncodesApproximateClearHeightFieldNormals(t *testing.T) {
	material := &thing.Material{Name: "water", Through: 1, Index: 1.333}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: spectralPlane(t), Material: material})
	if _, err := encode(world, maths.Subspace{}, nil, 0, 1, 1); err != nil {
		t.Fatalf("spectral clear HeightField was refused: %v", err)
	}
}

func TestMetalStillEncodesOpaqueSpectralAndExactClearHeightFields(t *testing.T) {
	opaque := thing.NewWorld()
	opaque.Add(thing.Thing{Shape: spectralPlane(t),
		Material: &thing.Material{Colour: maths.Grey(.4)}})
	if _, err := encode(opaque, maths.Subspace{}, nil, 0, 1, 1); err != nil {
		t.Fatalf("opaque spectral field was refused: %v", err)
	}

	exact, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	clear := thing.NewWorld()
	clear.Add(thing.Thing{Shape: exact,
		Material: &thing.Material{Name: "water", Through: 1, Index: 1.333}})
	if _, err := encode(clear, maths.Subspace{}, nil, 0, 1, 1); err != nil {
		t.Fatalf("exact-normal clear field was refused: %v", err)
	}
}
