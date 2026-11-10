package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

func TestFrontCardinalBearingsUseEastAndNegativeZForNorth(t *testing.T) {
	for _, test := range []struct {
		bearing   string
		direction maths.Vec
	}{
		{"0", maths.ZAxis.Neg()}, {"90", maths.XAxis},
		{"180", maths.ZAxis}, {"270", maths.XAxis.Neg()}, {"360", maths.ZAxis.Neg()},
	} {
		raw := strings.Replace(frontFixture(t), "bearing: {value: 90", "bearing: {value: "+test.bearing, 1)
		read, err := set.Read("cardinal-front.yaml", []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		front := read.FireFronts()[0]
		origin := maths.V(10, 20, 30)
		got := front.Motion(origin)
		if front.Name() != "through the grove" || got.Origin != origin || !(got.Direction.Sub(test.direction).Len() < 1e-12) {
			t.Fatalf("bearing %s: %+v", test.bearing, got)
		}
	}
}
