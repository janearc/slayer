package set_test

import "testing"

// the laser fixture's terrain placed by lat and lon, with a near window
// said after it: the lines of the terrain's west and north become them.
const (
	nearPlaced = "    lat: 37.79\n    lon: -122.41\n"
	nearBlock  = "    near: {lat: 37.76, lon: -122.45, across: 1000, " +
		"down: 800, step: 2}\n"
)

// A terrain says a near window: its place, its size and its step, and the
// terrain's own window is not changed by it.
func TestTerrainSaysANearWindow(t *testing.T) {
	made, err := legacyLaser(t, byMetres, nearPlaced+nearBlock)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	near, said := ground.Near()
	latitude, longitude, placed := near.Place()
	_, north, across, down, step := near.Window()
	if !said || !placed || latitude != 37.76 || longitude != -122.45 ||
		across != 1000 || down != 800 || north != 800 || step != 2 {
		t.Errorf("near %v %v %v, %v %v, window %v %v %v %v", said, placed,
			latitude, longitude, north, across, down, step, north)
	}
	_, _, farAcross, _, farStep := ground.Window()
	if farAcross != 1800 || farStep != 4 {
		t.Errorf("terrain across %v step %v", farAcross, farStep)
	}
}

// Anti: a terrain that says no near window has none.
func TestTerrainWithoutANearWindowHasNone(t *testing.T) {
	made, err := legacyLaser(t, byMetres, nearPlaced)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	if _, said := ground.Near(); said {
		t.Error("a near window where none was said")
	}
}

// A near window that leaves out any of its keys, or is in a terrain placed
// by metres, or is no finer than the terrain, is refused.
func TestTerrainNearWindowRefusals(t *testing.T) {
	for name, trial := range map[string]struct{ edit, want string }{
		"no step": {nearPlaced + "    near: {lat: 37.76, lon: -122.45, " +
			"across: 1000, down: 800}\n", "no step"},
		"by metres": {byMetres + nearBlock, "by lat and lon"},
		"as coarse": {nearPlaced + "    near: {lat: 37.76, lon: " +
			"-122.45, across: 1000, down: 800, step: 4}\n", "finer"},
		"unknown key": {nearPlaced + "    near: {lat: 37.76, lon: " +
			"-122.45, across: 1000, down: 800, step: 2, sea: 1}\n",
			"not a key"},
	} {
		_, err := legacyLaser(t, byMetres, trial.edit)
		mustHold(t, name, err, trial.want)
	}
}
