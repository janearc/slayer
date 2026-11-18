package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// TestSpacecraftServiceHardware checks actual ray hits in the new silhouettes,
// with neighbouring empty-space controls. Values are independent review
// locations in metres, not reads of the construction's proportion table.
func TestSpacecraftServiceHardware(t *testing.T) {
	cases := []struct {
		design               string
		aperture             float64
		material             string
		at, clear, direction maths.Vec
	}{
		{"hubble", 2.4, "thermal shield", maths.V(0, 14, -5),
			maths.V(2.5, 14, -5), maths.V(0, 0, 1)},
		{"hubble", 2.4, "thermal shield", maths.V(2.64, 5, 0),
			maths.V(3.3, 5, 0), maths.V(0, -1, 0)},
		{"spitzer", .85, "spacecraft structure", maths.V(.855, 1, 0),
			maths.V(.90, 1, 0), maths.V(0, -1, 0)},
		{"webb", 6.6, "telescope housing", maths.V(0, 5, 8),
			maths.V(3, 5, 8), maths.V(0, 0, -1)},
	}
	for _, sample := range cases {
		parts, _ := makeSpacecraft(t, spacecraftProfile{Design: sample.design,
			Aperture: sample.aperture}, 5, "demo")
		for _, probe := range []struct {
			at   maths.Vec
			want bool
		}{
			{sample.at, true}, {sample.clear, false},
		} {
			got := detailHit(parts, sample.material, maths.Ray{At: probe.at, Dir: sample.direction})
			if got != probe.want {
				t.Errorf("%s %s from %+v: hit %t, want %t",
					sample.design, sample.material, probe.at, got, probe.want)
			}
		}
	}
}

// detailHit considers only the named component family, so a hidden bus or
// mirror behind it cannot accidentally satisfy a missing-hardware probe.
func detailHit(parts []thing.Thing, material string, ray maths.Ray) bool {
	for _, part := range parts {
		if part.Material.Name == material {
			if solid, ok := part.Shape.(*mesh.Mesh); ok {
				if _, hit := solid.Hit(ray, 0, math.Inf(1)); hit {
					return true
				}
			}
		}
	}
	return false
}
