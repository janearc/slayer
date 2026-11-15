package oceantex

import "testing"

func TestLightingConstantsExposeScenePreparation(t *testing.T) {
	scene := Scene{Sea: Sea{Wind: 8, Fetch: 1500, Toward: 0.3}, Sun: Vec{Z: 1}, Celsius: 15, Chlorophyll: 2}
	actual, err := scene.LightingConstants()
	if err != nil {
		t.Fatal(err)
	}
	want, err := scene.lighting()
	if err != nil {
		t.Fatal(err)
	}
	if actual.Sun != want.sun || actual.Sunlight != [3]float64(want.sunlight) || actual.Water != [3]float64(want.water) || actual.Index != want.index || actual.Along != want.along || actual.Across != want.across || actual.Foam != want.foam {
		t.Fatalf("lighting constants: %+v", actual)
	}
}

func TestLightingConstantsRefuseSunBelowHorizon(t *testing.T) {
	if _, err := (Scene{Sun: Vec{Z: -1}}).LightingConstants(); err == nil {
		t.Fatal("sun below horizon accepted")
	}
}
