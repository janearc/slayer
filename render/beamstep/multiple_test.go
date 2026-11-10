package beamstep_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/beamstep"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/multiple.yaml
var multipleRaw []byte

func TestBeamRefinementHandlesWideningAndMixedWidths(test *testing.T) {
	fixture := readReference(test)
	var data struct {
		Cases []struct {
			Name  string `json:"name"`
			Beams []struct {
				East               float64 `json:"x" yaml:"x"`
				Radius, Divergence float64
			} `json:"beams"`
			Ray  maths.Ray `json:"ray"`
			Far  float64   `json:"far"`
			Cell cell      `json:"cell"`
		} `json:"cases"`
	}
	if err := yamlfile.Read("testdata/multiple.yaml", multipleRaw, &data); err != nil {
		test.Fatal(err)
	}
	for _, sample := range data.Cases {
		var lamps []thing.Lamp
		for _, launch := range sample.Beams {
			point := fixture.Source
			point.At.X = launch.East
			beam, err := emit.NewBeam(point, measure.Metres(launch.Radius), measure.Metres(fixture.Reach), measure.Degrees(launch.Divergence))
			if err != nil {
				test.Fatal(err)
			}
			lamps = append(lamps, beam)
		}
		refine := beamstep.Of(lamps).Ray(sample.Ray, sample.Far)
		if got := refine.Splits(sample.Cell.Enter, sample.Cell.Leave); got != sample.Cell.Splits {
			test.Fatal("wrong mixed-beam refinement", got, sample.Cell)
		}
	}
}
