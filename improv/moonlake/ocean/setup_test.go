package ocean

import (
	"os"
	"path/filepath"
	"testing"
)

// brightSky is a two by two sky with a moon bright enough to normalise.
func brightSky() RGB {
	return RGB{Width: 2, Height: 2, Light: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
}

// TestGaugeWidthIsTheGaugeSetupBuilds checks the width, 70.41 millimetres.
func TestGaugeWidthIsTheGaugeSetupBuilds(t *testing.T) {
	if got := GaugeWidth(); got != 70.41 {
		t.Errorf("GaugeWidth = %v, want 70.41", got)
	}
}

// TestSetupRefusesWhatCannotBeBuilt checks an invalid sky and a frame
// height of nothing, which no sky has, are both refused.
func TestSetupRefusesWhatCannotBeBuilt(t *testing.T) {
	config := DefaultConfig()
	if _, _, err := Setup(config, RGB{}, nil); err == nil {
		t.Error("Setup accepted an empty sky")
	}
	config.FrameHeight = 0
	if _, _, err := Setup(config, brightSky(), nil); err == nil {
		t.Error("Setup accepted a frame of no height")
	}
}

// TestRenderGivesAPictureOfTheAskedSize checks a small render has the
// width and height asked and one light value for each pixel.
func TestRenderGivesAPictureOfTheAskedSize(t *testing.T) {
	config := DefaultConfig()
	config.Width, config.Height = 4, 3
	picture, err := Render(config, brightSky(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if picture.Width != 4 || picture.Height != 3 || len(picture.Light) != 12 {
		t.Errorf("picture is %dx%d with %d pixels", picture.Width, picture.Height, len(picture.Light))
	}
}

// TestRenderRefusesWhatSetupRefuses checks the error comes back whole
// and no picture comes with it.
func TestRenderRefusesWhatSetupRefuses(t *testing.T) {
	picture, err := Render(DefaultConfig(), RGB{}, nil)
	if err == nil || picture != nil {
		t.Errorf("Render gave %v and %v for an empty sky", picture, err)
	}
	config := DefaultConfig()
	if picture, err := Render(config, brightSky(), nil); err == nil || picture != nil {
		t.Errorf("Render gave %v and %v for a picture of no size", picture, err)
	}
}

// TestReadPFMRefusesAMissingFileAndAHeightThatIsNotANumber checks the
// open error and the height parse error.
func TestReadPFMRefusesAMissingFileAndAHeightThatIsNotANumber(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadPFM(filepath.Join(dir, "none.pfm")); err == nil {
		t.Error("ReadPFM accepted a missing file")
	}
	path := filepath.Join(dir, "tall.pfm")
	if err := os.WriteFile(path, []byte("PF\n1 tall\n-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPFM(path); err == nil {
		t.Error("ReadPFM accepted a height that is not a number")
	}
}
