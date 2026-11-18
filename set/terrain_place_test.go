package set_test

import "testing"

// the two lines of the laser fixture that place its terrain by metres
const byMetres = "    west: 263000\n    north: 2149000\n"

// placedBy is a terrain placed on a sphere: the lines that say where and
// how much detail.
const placedBy = "    lat: -7.5\n    lon: 303\n    detail: {seed: 13}\n"

// A terrain placed by lat and lon says its place, its seed and a window in
// local metres: the west edge at nought and the north edge as deep as the
// window is.
func TestTerrainPlacedByLatAndLon(t *testing.T) {
	made, err := legacyLaser(t, byMetres, placedBy)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	latitude, longitude, placed := ground.Place()
	seed, asked := ground.Detail()
	west, north, across, down, step := ground.Window()
	if !placed || latitude != -7.5 || longitude != 303 || !asked || seed != 13 {
		t.Errorf("place %v %v %v, detail %v %v", latitude, longitude, placed,
			seed, asked)
	}
	if west != 0 || north != 1300 || across != 1800 || down != 1300 ||
		step != 4 {
		t.Errorf("window %v %v %v %v %v", west, north, across, down, step)
	}
}

// Anti: a terrain placed by west and north is not placed on a sphere, asks
// no detail, and has the window it always had.
func TestTerrainPlacedByMetresIsAsItWas(t *testing.T) {
	made, err := legacyLaser(t)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	_, _, placed := ground.Place()
	_, asked := ground.Detail()
	west, north, _, _, _ := ground.Window()
	if placed || asked || west != 263000 || north != 2149000 {
		t.Errorf("placed %v, detail %v, west %v north %v", placed, asked, west,
			north)
	}
}

// A terrain placed both ways is refused, naming both.
func TestTerrainPlacedBothWaysIsRefused(t *testing.T) {
	for name, edit := range map[string]string{
		"west and lat":  "    west: 263000\n    lat: 1\n    lon: 2\n",
		"north and lon": "    north: 2149000\n    lat: 1\n    lon: 2\n",
	} {
		_, err := legacyLaser(t, byMetres, edit)
		mustHold(t, name, err, "not both")
		mustHold(t, name, err, "lat")
	}
}

// Anti: either way alone is not refused, and a window given nowhere to be
// placed by metres is not taken for one placed by degrees.
func TestTerrainPlacedOneWayIsNotRefused(t *testing.T) {
	for name, edit := range map[string]string{
		"metres":  byMetres,
		"degrees": "    lat: 1\n    lon: 2\n",
		"neither": "",
	} {
		if _, err := legacyLaser(t, byMetres, edit); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// A lat without a lon, or a lon without a lat, is refused.
func TestTerrainHalfAPlaceIsRefused(t *testing.T) {
	for name, edit := range map[string]string{
		"lat alone": "    lat: 1\n",
		"lon alone": "    lon: 2\n",
	} {
		_, err := legacyLaser(t, byMetres, edit)
		mustHold(t, name, err, "go together")
	}
}

// Anti: the pair, in either order, is read.
func TestTerrainAPlaceInEitherOrder(t *testing.T) {
	for _, edit := range []string{"    lat: 1\n    lon: 2\n",
		"    lon: 2\n    lat: 1\n"} {
		if _, err := legacyLaser(t, byMetres, edit); err != nil {
			t.Error(err)
		}
	}
}

// Detail with no place on a sphere to lay it on is refused.
func TestTerrainDetailWithoutAPlaceIsRefused(t *testing.T) {
	_, err := legacyLaser(t, byMetres, byMetres+"    detail: {seed: 1}\n")
	mustHold(t, "detail on metres", err, "detail is for terrain placed by lat")
}

// Anti: a placed terrain may leave detail out, and then asks none.
func TestTerrainPlacedMayAskNoDetail(t *testing.T) {
	made, err := legacyLaser(t, byMetres, "    lat: 1\n    lon: 2\n")
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	if _, asked := ground.Detail(); asked {
		t.Error("detail was asked and not said")
	}
}

// A place off the sphere, and a seed that is not a whole number, are
// refused.
func TestTerrainRefusesWhatIsOutOfRange(t *testing.T) {
	for name, edit := range map[string]string{
		"lat over":    "    lat: 90.5\n    lon: 0\n",
		"lat under":   "    lat: -91\n    lon: 0\n",
		"lon over":    "    lat: 0\n    lon: 361\n",
		"lon under":   "    lat: 0\n    lon: -361\n",
		"seed signed": "    lat: 0\n    lon: 0\n    detail: {seed: -1}\n",
		"seed broken": "    lat: 0\n    lon: 0\n    detail: {seed: 1.5}\n",
		"detail key":  "    lat: 0\n    lon: 0\n    detail: {grain: 1}\n",
	} {
		_, err := legacyLaser(t, byMetres, edit)
		mustHold(t, name, err, "")
	}
}

// Anti: the ends of the ranges are in them.
func TestTerrainAcceptsTheEndsOfItsRanges(t *testing.T) {
	for name, edit := range map[string]string{
		"north pole": "    lat: 90\n    lon: 0\n",
		"south pole": "    lat: -90\n    lon: 0\n",
		"far east":   "    lat: 0\n    lon: 360\n",
		"far west":   "    lat: 0\n    lon: -360\n",
		"seed zero":  "    lat: 0\n    lon: 0\n    detail: {seed: 0}\n",
	} {
		if _, err := legacyLaser(t, byMetres, edit); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
