package set_test

import "testing"

// A terrain that says its reflectance gives it back, red, green and blue.
func TestTerrainSaysItsReflectance(t *testing.T) {
	made, err := legacyLaser(t, byMetres,
		byMetres+"    reflectance: [0.04, 0.035, 0.03]\n")
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	red, green, blue, said := ground.Reflectance()
	// picked: the three numbers the reflectance line says
	if !said || red != 0.04 || green != 0.035 || blue != 0.03 { // picked
		t.Errorf("reflectance %v %v %v, said %v", red, green, blue, said)
	}
}

// Anti: a terrain that says no reflectance says so, and gives back none.
func TestTerrainWithoutReflectanceSaysNone(t *testing.T) {
	made, err := legacyLaser(t)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	if red, green, blue, said := ground.Reflectance(); said || red != 0 ||
		green != 0 || blue != 0 {
		t.Errorf("reflectance %v %v %v, said %v", red, green, blue, said)
	}
}

// A reflectance past all the light, below none, or not three numbers is
// refused.
func TestTerrainRefusesAReflectanceOutOfRange(t *testing.T) {
	for name, edit := range map[string]string{
		"over":  "    reflectance: [1.2, 0.5, 0.5]\n",
		"under": "    reflectance: [0.1, -0.1, 0.1]\n",
		"two":   "    reflectance: [0.1, 0.1]\n",
		"word":  "    reflectance: dark\n",
	} {
		_, err := legacyLaser(t, byMetres, byMetres+edit)
		mustHold(t, name, err, "")
	}
}
