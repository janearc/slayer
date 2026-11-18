package stage_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

const authoredBeam = `      column:
        at: [1, 2, 3]
        aim: [0, 2, 0]
        radius: 0.5
        reach: 8
        divergence: 0.8
        temperature: 5200
        strength: 4
        source: eye
`

func withBeam(t *testing.T, filename, unit string) set.World {
	t.Helper()
	raw, err := os.ReadFile("testdata/yard.yaml")
	if err != nil {
		t.Fatal(err)
	}
	world := "  world:\n"
	add := world + "    unit: " + unit + "\n    beams:\n" + authoredBeam
	text := strings.Replace(string(raw), world, add, 1)
	if text == string(raw) {
		t.Fatal("yard fixture has no world block")
	}
	read, err := set.Read(filename, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	said, has := read.World()
	if !has {
		t.Fatal("fixture has no world")
	}
	return said
}

func findBeam(world *thing.World) *emit.Beam {
	for _, lamp := range world.Lamps {
		if beam, ok := lamp.(*emit.Beam); ok {
			return beam
		}
	}
	return nil
}

func TestWorldStagesAuthoredBeamAtWorldScaleAndAsBlackBody(t *testing.T) {
	said := withBeam(t, "beam-yard.yaml", "2")
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	beam := findBeam(world)
	if beam == nil {
		t.Fatalf("stager dropped authored beam; lamps %T", world.Lamps)
	}
	point := beam.Point()
	if point.At != maths.V(1, 3, -2) || point.Toward != maths.V(0, 0, -1) ||
		math.Abs(point.Strength.Luminance()-4) > 1e-10 ||
		point.Spectrum.Power(bands.Of(555)) <= 0 ||
		beam.Radius().InMetres() != 0.5 || beam.Reach().InMetres() != 8 ||
		beam.Slope() <= 0 {
		t.Errorf("staged beam lost authored scale, black-body light or envelope: %+v, %+v", point, beam)
	}
}

func TestAddShiftsBeamOriginWithoutChangingItsDirectionOrEnvelope(t *testing.T) {
	said := withBeam(t, "shifted-beam.yaml", "1")
	world := thing.NewWorld()
	shift := maths.V(5, 7, 11)
	if err := stage.Add(world, said, shift); err != nil {
		t.Fatal(err)
	}
	beam := findBeam(world)
	if beam == nil || beam.Point().At != maths.V(6, 10, 9) ||
		beam.Point().Toward != maths.V(0, 0, -1) ||
		beam.Radius().InMetres() != 0.5 || beam.Reach().InMetres() != 8 {
		t.Fatalf("shifted beam is %+v", beam)
	}
}
