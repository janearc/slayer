package manufacture_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

type summitContract struct {
	Seed          uint64
	Tolerance     float64
	MaximumFaces  int `yaml:"maximum-faces"`
	Axis, Tangent maths.Vec
	Profiles      []summitProfile
}

type summitProfile struct {
	Kind, Role          string
	Aperture, Near, Far float64
	Copies              int
	Group               bool
	Centres             []maths.Vec
	Minimum, Maximum    maths.Vec
}

type summitProbe struct {
	contract summitContract
}

// polygon checks closure before measuring its placement in the reserved site.
func (probe summitProbe) polygon(test *testing.T, polygon *mesh.Mesh,
	pose maths.Subspace, bounds maths.Box,
) {
	test.Helper()
	if !polygon.Closed() || polygon.FaceCount() > probe.contract.MaximumFaces {
		test.Fatal("open or excessive summit prototype")
	}
	limits := meshContract{Tolerance: probe.contract.Tolerance,
		MaximumFaces: probe.contract.MaximumFaces}
	for _, point := range polygon.Points() {
		limits.assertPoint(test, pose.Out(point), bounds)
	}
}
