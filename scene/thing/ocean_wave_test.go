package thing

import "testing"

// oceanWaveRecipe is a wave recipe with a surface wake whose numbers are
// the picked ones, so that a change to any of them shows.
func oceanWaveRecipe(t *testing.T) OceanWaveConfig {
	t.Helper()
	return OceanWaveConfig{
		WindMetresPerSecond: picked(t, "ocean-wave-wind"),
		Samples:             pickedCount(t, "ocean-wave-samples"),
		SceneClock:          true,
		SurfaceWake: &OceanSurfaceWakeConfig{
			AmplitudeMetres: picked(t, "ocean-wave-amplitude"),
			LengthMetres:    picked(t, "ocean-wave-length"),
		},
	}
}

// What a caller changes in the recipe it handed over does not reach the
// staged motion, and what it changes in the recipe it was handed back does
// not either: the motion keeps its own wake.
func TestOceanWaveMotionKeepsItsOwnWake(t *testing.T) {
	recipe := oceanWaveRecipe(t)
	motion := NewOceanWaveMotion(recipe)
	changed := picked(t, "ocean-wave-changed")
	want := picked(t, "ocean-wave-amplitude")
	recipe.SurfaceWake.AmplitudeMetres = changed
	if got := motion.Config().SurfaceWake.AmplitudeMetres; got != want {
		t.Errorf("a change to the handed-over wake reached the motion: %v, "+
			"want %v", got, want)
	}
	motion.Config().SurfaceWake.AmplitudeMetres = changed
	motion.Config().SurfaceWake.LengthMetres = changed
	got := motion.Config().SurfaceWake
	if got.AmplitudeMetres != want ||
		got.LengthMetres != picked(t, "ocean-wave-length") {
		t.Errorf("a change to the returned wake reached the motion: %+v", got)
	}
}

// The anti-test: the copy is a copy and not a loss. It holds every number
// of the recipe, a recipe with no wake has none, and a motion that is nil
// gives the empty recipe.
func TestOceanWaveMotionCopiesWhatItWasGiven(t *testing.T) {
	recipe := oceanWaveRecipe(t)
	got := NewOceanWaveMotion(recipe).Config()
	if got.SurfaceWake == recipe.SurfaceWake {
		t.Error("the returned wake is the one handed over")
	}
	if *got.SurfaceWake != *recipe.SurfaceWake ||
		got.WindMetresPerSecond != recipe.WindMetresPerSecond ||
		got.Samples != recipe.Samples || got.SceneClock != recipe.SceneClock {
		t.Errorf("the copy differs: %+v, want %+v", got, recipe)
	}
	recipe.SurfaceWake = nil
	if NewOceanWaveMotion(recipe).Config().SurfaceWake != nil {
		t.Error("a recipe with no wake gained one")
	}
	var none *OceanWaveMotion
	if got := none.Config(); got.SurfaceWake != nil || got.Samples != nought {
		t.Errorf("a nil motion gives %+v", got)
	}
}
