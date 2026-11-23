package session

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/set"
)

// laterCameraSetText is a set of two cans whose camera stands off the one
// named last, so the camera cannot be placed until the later can is.
func laterCameraSetText() string {
	var text strings.Builder
	text.WriteString(`scene:
  name: later camera
  film: {width: 1, height: 1, rays: 1}
  cameras:
    between:
      title: between
      location:
        loc_relative: {object: last can}
      orientation:
        point_to: first can
  set:
    things:
`)
	for _, each := range []struct {
		name  string
		north string
	}{{"first can", "100"}, {"last can", "200"}} {
		text.WriteString("      " + each.name + `:
        kind: can
        seed: 1
        complexity: quick
        position:
          east: {value: 0, unit: m}
          north: {value: ` + each.north + `, unit: m}
          up: {value: 0, unit: m}
`)
	}
	return text.String()
}

// TestACameraOffAThingThereIsNoneOfIsRefused is the anti-test of the
// placing: a camera that stands off a thing the set does not name is
// refused where scrooge is asked, so the check above can report a camera.
func TestACameraOffAThingThereIsNoneOfIsRefused(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	t.Setenv(planTrialVariable, "")
	text := strings.Replace(laterCameraSetText(),
		"loc_relative: {object: last can}",
		"loc_relative: {object: no such can}", 1)
	read, err := set.Read("positions.yaml", []byte(text))
	if err != nil {
		return
	}
	stage := warmStaging(t, read)
	stage.sight = nil
	if err := stage.stand(read); err == nil {
		t.Error("a camera off a thing there is none of was stood")
	}
}

// TestACameraOffALaterThingIsPlacedBeforeAnyThingIsMade checks that where
// scrooge is asked, the first thing judged by a camera that stands off a
// thing named after it is stood without refusal, and every thing's place
// is known by the end.
func TestACameraOffALaterThingIsPlacedBeforeAnyThingIsMade(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	t.Setenv(planTrialVariable, "")
	read := readPositionSet(t, laterCameraSetText())
	stage := warmStaging(t, read)
	stage.sight = nil
	if err := stage.stand(read); err != nil {
		t.Fatalf("a camera off a later thing: %v", err)
	}
	for _, name := range []string{"first can", "last can"} {
		if _, known := stage.places[name]; !known {
			t.Errorf("the place of %s is not known", name)
		}
	}
}
