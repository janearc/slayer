package session

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/terrain"
)

// The synthetic hillside the slope tests stand on: 20 points each way, 5 m
// apart, 10 m over sea level at its west edge and rising a tenth of a metre
// for each metre east.
const (
	slopeAcross   = 20
	slopeStep     = 5.0
	slopeGradient = 0.1
	slopeBase     = 10.0
)

// risingStage is a stage over the synthetic hillside.
func risingStage(t *testing.T) *staging {
	t.Helper()
	window, err := terrain.BuildWindow(0, 100, slopeAcross, slopeAcross,
		slopeStep, func(column, _ int) (float64, error) {
			return slopeBase + slopeGradient*slopeStep*float64(column), nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return &staging{ground: window, lowest: slopeBase}
}

func TestTheGroundUnderABuildingIsTheTerrainInTheBuildingsOwnFrame(t *testing.T) {
	stage := risingStage(t)
	place := maths.V(30, slopeGradient*30, 40)
	unturned := stage.groundUnder(place, 0)
	if got := unturned(10, 7); math.Abs(got-slopeGradient*10) > 1e-9 {
		t.Errorf("10 m east of the building the ground is %v over its "+
			"own, want %v", got, slopeGradient*10)
	}
	turned := stage.groundUnder(place, math.Pi/2)
	if got := turned(10, 0); math.Abs(got) > 1e-9 {
		t.Errorf("along a building turned a quarter the ground is %v over "+
			"its own, want level", got)
	}
	if got := turned(0, 10); math.Abs(got-slopeGradient*10) > 1e-9 {
		t.Errorf("across it the ground is %v over its own, want %v", got,
			slopeGradient*10)
	}
}
