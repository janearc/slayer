//ninja:vow dojo this checks the private cap construction at its axis
package manufacture

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/slit-cylinder.yaml
var slitCylinderRaw []byte

func TestGroundObservatorySlitCylinderStaysClosedAtItsAxis(test *testing.T) {
	var data struct {
		Outer, Height, Opening, Lift float64
		Inners                       []float64
		Sides                        []int
	}
	if err := yamlfile.Read("slit-cylinder.yaml", slitCylinderRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	for _, inner := range data.Inners {
		shape := slitCylinder{outer: data.Outer, inner: inner,
			height: data.Height, opening: data.Opening * maths.Degree}
		for _, sides := range data.Sides {
			skin := shape.skin(cut{around: sides})
			if _, err := skin.meshed(); err != nil {
				test.Fatal("cap is not a valid mesh", inner, sides, err)
			}
			pose := maths.NewSubspace(maths.YAxis.Scale(data.Lift),
				maths.Identity, spaceOne)
			polygon, err := skin.placed(pose).meshed()
			if err != nil || !polygon.Closed() {
				test.Fatal("translated cap is not closed", inner, sides, err)
			}
		}
	}
}
