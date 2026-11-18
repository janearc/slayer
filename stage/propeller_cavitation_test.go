package stage_test

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

const cavitationSet = `scene:
  world:
    vapor-cavities:
      screw:
        colour: {srgb_linear: [0.82, 0.84, 0.86]}
        origin: [-1500, 0, -55]
        axis: [0.342, -0.94, 0]
        blades: 5
        tip-radius: 2.1
        rpm: 300
        slipstream-mps: 20
        length-m: 40
        core-radius-m: 0.2
        density-per-m: 0.8
        fragment-scale-m: 0.5
        fragment-contrast: 0
        fragment-seed: 705
        pressure-deficit-pa: 12000
        recovery-pa-per-m: 600
        visual-onset: true
        clear-water: true
        source: eye
`

func TestVaporCavitationSetBuildsOneBoundedClearCloud(t *testing.T) {
	parsed, err := set.Read("vapor.yaml", []byte(cavitationSet))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := parsed.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 1 {
		t.Fatalf("built %d volumes, want one compact field", len(world.Volumes))
	}
	volume := world.Volumes[0]
	cloud, ok := volume.Medium.(interface {
		medium.Medium
		medium.Timed
		medium.Spanner
		ClearMediumOverlay()
	})
	if !ok {
		t.Fatalf("built %T, want a timed marked clear cloud", volume.Medium)
	}
	bounds := cloud.Bounds()
	if bounds.Max.Z-bounds.Min.Z > 30 {
		t.Fatalf("the pressure-recovered field exceeds its finite axial envelope: %+v", bounds)
	}
	field := cloud.Tick(time.Second)
	// One second is five shaft turns; four metres of convection is one
	// turn of delay. The first tip is therefore on the authored radial axis.
	axis := maths.V(.342, 0, .94).Unit()
	radial := maths.V(-.94, 0, .342).Unit()
	point := maths.V(-1500, -55, 0).Add(axis.Scale(4)).Add(radial.Scale(2.1))
	if density := field.Density(point); density <= 0 {
		t.Fatalf("the rendered field has no density near its authored placement: %g", density)
	}
}

func TestVaporCavitationSetRefusesAnInvalidAxis(t *testing.T) {
	bad := strings.Replace(cavitationSet, "axis: [0.342, -0.94, 0]", "axis: [0, 0, 0]", 1)
	parsed, err := set.Read("vapor.yaml", []byte(bad))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := parsed.World()
	if _, err := stage.World(said); err == nil || !strings.Contains(err.Error(), "axis") {
		t.Fatalf("zero axis build error = %v", err)
	}
}

func TestVaporCavitationSetOffRemainsOptIn(t *testing.T) {
	still := strings.Replace(cavitationSet, "visual-onset: true", "visual-onset: false", 1)
	parsed, err := set.Read("vapor.yaml", []byte(still))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := parsed.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	field := world.Volumes[0].Medium.(medium.ClearCloud)
	if density := field.Density(maths.V(-1498, -55, 4)); density != 0 {
		t.Fatalf("disabled visual onset produced density %g", density)
	}
}
