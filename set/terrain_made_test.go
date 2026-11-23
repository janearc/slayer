package set_test

import "testing"

// madeAll is made ground with every process said: the lines that say it
// under a terrain.
const madeAll = `    made:
      seed: 9
      steep: 1.5
      flutes: {depth: 0.8, width: 1.2}
      beds: {spacing: 2.5, standout: 0.3}
`

// A terrain placed by west and north that says made ground gives its seed
// and each process as written.
func TestTerrainSaysItsMadeGround(t *testing.T) {
	read, err := legacyLaser(t, byMetres, byMetres+madeAll)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := read.Ground()
	made, asked := ground.Made()
	if !asked || made.Seed() != 9 { // picked: the seed the lines say
		t.Fatalf("made ground asked %v, seed %d", asked, made.Seed())
	}
	if steep, said := made.Steep(); !said || steep != 1.5 { // picked
		t.Errorf("steep %v, said %v", steep, said)
	}
	if depth, width, said := made.Flutes(); !said || depth != 0.8 ||
		width != 1.2 { // picked
		t.Errorf("flutes %v by %v, said %v", depth, width, said)
	}
	if spacing, standout, said := made.Beds(); !said || spacing != 2.5 ||
		standout != 0.3 { // picked
		t.Errorf("beds %v and %v, said %v", spacing, standout, said)
	}
}

// Anti: made ground that says only its seed asks no flutes, no beds and
// no steepness of its own.
func TestTerrainMadeGroundSaysOnlyItsSeed(t *testing.T) {
	read, err := legacyLaser(t, byMetres,
		byMetres+"    made: {seed: 4}\n")
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := read.Ground()
	made, asked := ground.Made()
	_, steeped := made.Steep()
	_, _, fluted := made.Flutes()
	_, _, bedded := made.Beds()
	if !asked || steeped || fluted || bedded {
		t.Fatalf("asked %v, steep %v, flutes %v, beds %v", asked, steeped,
			fluted, bedded)
	}
}

// Anti: a window placed by lat and lon, with no detail and no near
// window, may say made ground, and gives it.
func TestTerrainPlacedOnASphereMaySayMadeGround(t *testing.T) {
	read, err := legacyLaser(t, byMetres,
		"    lat: 1\n    lon: 2\n    made: {seed: 3}\n")
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := read.Ground()
	if made, asked := ground.Made(); !asked || made.Seed() != 3 {
		t.Fatalf("made ground asked %v, seed %d", asked, made.Seed())
	}
}

// Anti: a terrain that says no made ground asks none.
func TestTerrainAsksNoMadeGround(t *testing.T) {
	read, err := legacyLaser(t)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := read.Ground()
	if _, asked := ground.Made(); asked {
		t.Fatal("a terrain that says no made ground asked some")
	}
}

// Made ground with no seed, a flute without its width, beside a detail
// or with a near window is refused.
func TestTerrainMadeGroundRefusals(t *testing.T) {
	for name, test := range map[string]struct{ edit, want string }{
		"no seed": {byMetres + "    made: {steep: 1}\n",
			"made ground says [seed], all of them, and this has no seed"},
		"half a flute": {byMetres +
			"    made: {seed: 1, flutes: {depth: 1}}\n",
			"this has no width"},
		"with detail": {placedBy + "    made: {seed: 1}\n",
			"made ground and detail each make the ground below the map"},
		"with a near window": {"    lat: 1\n    lon: 2\n" +
			"    near: {lat: 1, lon: 2, across: 100, down: 100, step: 1}\n" +
			"    made: {seed: 1}\n",
			"a terrain with a near window does not say it"},
	} {
		_, err := legacyLaser(t, byMetres, test.edit)
		mustHold(t, name, err, test.want)
	}
}
