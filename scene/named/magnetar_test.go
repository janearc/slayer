package named_test

import (
	"errors"
	"io"
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// Every parameter reads its own default back, and the scene builds with
// each given alone.
func TestEveryMagnetarParameterReads(t *testing.T) {
	scene := mustGet(t, "magnetar")
	for _, one := range scene.Parameters() {
		with := map[string]any{one.Name(): one.Default()}
		if err := scene.Check(with); err != nil {
			t.Errorf("%s at its default %v: %v", one.Name(),
				one.Default(), err)
		}
	}
}

// anti-tests: near misses are refused by name: no star, no field, an
// axis past the equator or before the pole, a tail shown slower than
// true, an eye behind the middle, a view there is not, a key there is
// not.
func TestTheMagnetarRefusesNearMisses(t *testing.T) {
	scene := mustGet(t, "magnetar")
	for _, case_ := range []struct {
		with map[string]any
		want error
	}{
		{map[string]any{"radius": float64(none)}, named.ErrBounds},
		{map[string]any{"field": float64(none)}, named.ErrBounds},
		{map[string]any{"tilt": 90.5}, named.ErrBounds},
		{map[string]any{"tilt": -0.5}, named.ErrBounds},
		{map[string]any{"flare-speed": 0.5}, named.ErrBounds},
		{map[string]any{"distance": -0.5}, named.ErrBounds},
		{map[string]any{"view": "sideways"}, named.ErrChoice},
		{map[string]any{"spin": 1.5}, named.ErrNoParameter},
	} {
		if err := scene.Check(case_.with); !errors.Is(err, case_.want) {
			t.Errorf("%v: %v; want %v", case_.with, err, case_.want)
		}
	}
}

// The script reads and every line stands: each figure in it is its
// named value at the digits written.
func TestEveryLineOfTheScriptStands(t *testing.T) {
	lines := named.MagnetarScript()
	if len(lines) == none {
		t.Fatal("the script says nothing")
	}
	filmed := none
	for _, line := range lines {
		if err := line.Check(); err != nil {
			t.Error(err)
		}
		if line.Film {
			filmed++
		}
	}
	if filmed == none {
		t.Error("the short film says none of the script")
	}
}

// anti-tests: a line is refused when a figure has no value behind it,
// is not its value at the digits written, is written in words, is
// listed but not said, names a value nobody holds, or goes under a shot
// there is not.
func TestALooseFigureIsRefused(t *testing.T) {
	for _, line := range []named.Line{
		{Say: "It is 20 km across.", Shot: "field"},
		{Say: "I am 901 times too wide.", Shot: "beside",
			Figures: map[string]string{"901": "zappy-times-larger"}},
		{Say: "It is twenty km across.", Shot: "field"},
		{Say: "It is wide.", Shot: "field",
			Figures: map[string]string{"20": "magnetar-diameter-km"}},
		{Say: "It is 20 km across.", Shot: "field",
			Figures: map[string]string{"20": "magnetar-width-guessed"}},
		{Say: "It is 20 km across.", Shot: "sideways",
			Figures: map[string]string{"20": "magnetar-diameter-km"}},
	} {
		if err := line.Check(); !errors.Is(err, named.ErrScript) {
			t.Errorf("%q: %v; want ErrScript", line.Say, err)
		}
	}
}

// A figure may be rounded to the digits it is written with, and stands
// on a registered value as well as a computed one.
func TestARoundedFigureStands(t *testing.T) {
	for _, line := range []named.Line{
		{Say: "Inside 950 km.", Shot: "shells",
			Figures: map[string]string{"950": "reach-atoms-km"}},
		{Say: "Every 7.56 seconds.", Shot: "flare",
			Figures: map[string]string{
				"7.56": "sgr1806-tail-period-seconds"}},
	} {
		if err := line.Check(); err != nil {
			t.Errorf("%q: %v", line.Say, err)
		}
	}
}

// the distances the eye is asked to stand at, in km, and how near it
// must stand to them; picked by the author of this scene, 2026-11-12
const (
	askedNear   = 500
	askedBeside = 40
	standSlack  = 1e-9
)

// The eye stands at the distance asked, in the view asked, beside zappy
// as near the real star as asked.
func TestTheEyeStandsWhereAsked(t *testing.T) {
	for view, asked := range map[string]float64{"near": askedNear,
		"beside": askedBeside} {
		built := mustBuild(t, "magnetar", map[string]any{"view": view,
			"distance": asked})
		if got := built.Eye().Origin.Len(); got < asked-standSlack ||
			got > asked+standSlack {
			t.Errorf("%s at %g km: the eye is %g km out", view, asked, got)
		}
	}
}

// anti-test: a star whose surface field is weaker than the quantum
// field has no quantum hoop, and builds with one thing fewer.
func TestAWeakStarHasFewerHoops(t *testing.T) {
	full := len(mustBuild(t, "magnetar", nil).World().Things)
	weak := len(mustBuild(t, "magnetar",
		map[string]any{"field": 0.1}).World().Things)
	if weak != full-once {
		t.Errorf("a weak star has %d things, the real one %d", weak, full)
	}
}

// the distances a world is asked to stand at that the dipole cannot
// draw, in km: past the light cylinder, and inside the star; picked by
// the author of this scene, 2026-11-12
const pastTheCylinder, insideTheStar = 4e5, 5

// anti-test: a world past the light cylinder, where the field law does
// not hold, or inside the star, is refused by name.
func TestAWorldPastTheLightCylinderIsRefused(t *testing.T) {
	scene := mustGet(t, "magnetar")
	for _, out := range []float64{pastTheCylinder, insideTheStar} {
		_, err := scene.Build(map[string]any{"world": true,
			"world-at": out}, io.Discard)
		if !errors.Is(err, named.ErrBounds) {
			t.Errorf("a world at %g km: %v; want ErrBounds", out, err)
		}
	}
}
