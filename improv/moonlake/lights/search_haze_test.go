package lights

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"testing"
)

func effectDigest(values []float64) string {
	raw := make([]byte, len(values)*8)
	for i, value := range values {
		binary.LittleEndian.PutUint64(raw[i*8:], math.Float64bits(value))
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func searchFixture() (Image, Image, SearchConfig, SearchInput) {
	sky := Image{24, 18, make([]float64, 24*18*3)}
	front := Image{24, 18, make([]float64, len(sky.Light))}
	open, closeTrees := make([]bool, 24*18), make([]bool, 24*18)
	for i := range open {
		open[i] = i/24 < 8+i%3 && i%11 != 0
		closeTrees[i] = i%13 == 0
	}
	for i := range sky.Light {
		sky.Light[i] = float64(i%17+1) / 1024
	}
	cam := NewCamera(24, 18, CameraSettings{109.65, -0.5, 70.41, 150})
	eye := float64(1129.6 - 1076)
	farTree := Vec{1727.5, -617.5, 1195.5 - (1076 + eye)}
	return sky, front, SearchConfig{0.05, 8}, SearchInput{
		Open: open, CloseTrees: closeTrees, Camera: cam, Moon: [3]float64{0.7, 1.1, 1.9},
		FarTree: farTree, Azimuth: 109.65,
		ShoreRow: 11.25, Level: 0.8, Gate: 0.7, Phase: 0.23, MovePhase: 0.37, Move: "sweep",
	}
}

func TestSearchCapturedOutputs(t *testing.T) {
	for _, test := range []struct{ move, sky string }{
		{"sweep", "1d6f1071b877a1cb47af4ab339f02a6e4298de84e49e56ff476c02196be7801c"},
		{"chase", "385cffb8e753ee4148bee7879efd75e825e23aec6e6713bc51acd4db90d4d530"},
		{"fan", "6a1d83974973d8990a2a89ea3eedb3d2c4af2e8c2a1f9b13c7dcefc97f32f505"},
		{"viewer", "2b8223e546680412dba3c434be29bef5cefe1f28cdf3e46b14a6e5cc07a6c12f"},
		{"unknown", "1d6f1071b877a1cb47af4ab339f02a6e4298de84e49e56ff476c02196be7801c"},
	} {
		t.Run(test.move, func(t *testing.T) {
			sky, front, config, input := searchFixture()
			input.Move = test.move
			AddSearchlights(sky, front, config, input)
			if got := effectDigest(sky.Light); got != test.sky {
				t.Fatalf("sky=%s want %s", got, test.sky)
			}
			if got := effectDigest(front.Light); got != "3b8b40b17f089bb594741d74ef45006e9f7bdbf1e4c59306cbaceeb5a29285c3" {
				t.Fatalf("front=%s", got)
			}
		})
	}
}

func hazeFixture() (Image, Image, BackHazeInput) {
	front, _, _, search := searchFixture()
	trees := Image{24, 18, make([]float64, len(front.Light))}
	for i := range trees.Light {
		if i/3%11 == 0 {
			trees.Light[i] = 1
		}
	}
	return front, trees, BackHazeInput{Camera: search.Camera, Moon: search.Moon,
		Positions: Backlights(1129.6 - 1076), Levels: [2]float64{0.8, 0.4}, Gain: 0.6, Seconds: 1.25}
}

func TestHazeCapturedOutputs(t *testing.T) {
	for _, test := range []struct {
		levels [2]float64
		want   string
	}{
		{[2]float64{0.8, 0.4}, "02bbef6973d443696e27841968992809825dde9b1da752895e3ebe9664fa35e4"},
		{[2]float64{1, 0}, "8f6c9188b02713f24eb2f9bf9f05fa4e46aba39c0ce38a30963b3cadb2730425"},
		{[2]float64{0, 1}, "7936d5199f42e8758f767244ceec4f3856e5c37a2fc97821299d52b379bfa20e"},
		{[2]float64{0.009, 0.009}, "e2c28e6b4310b156a91de17489d918ff1a428bbe9dd28dd3d45bec012933bc8e"},
	} {
		front, trees, input := hazeFixture()
		input.Levels = test.levels
		AddBackHaze(front, trees, input)
		if got := effectDigest(front.Light); got != test.want {
			t.Fatalf("levels=%v digest=%s want %s", test.levels, got, test.want)
		}
	}
}

func TestSearchMasksGateAndInstanceIsolation(t *testing.T) {
	for _, move := range []string{"sweep", "chase", "fan", "viewer"} {
		t.Run(move, func(t *testing.T) {
			sky, front, config, input := searchFixture()
			input.Move = move
			original := slices.Clone(sky.Light)
			open, closeTrees := slices.Clone(input.Open), slices.Clone(input.CloseTrees)
			input.Gate = 0.009 / input.Level
			AddSearchlights(sky, front, config, input)
			if !slices.Equal(sky.Light, original) || slices.ContainsFunc(front.Light, func(v float64) bool { return v != 0 }) {
				t.Fatal("subthreshold gate changed images")
			}
			input.Gate = 0.7
			for i := range input.Open {
				input.Open[i] = false
				input.CloseTrees[i] = true
			}
			AddSearchlights(sky, front, config, input)
			if !slices.Equal(sky.Light, original) || slices.ContainsFunc(front.Light, func(v float64) bool { return v != 0 }) {
				t.Fatal("occlusion did not suppress light")
			}
			copy(input.Open, open)
			copy(input.CloseTrees, closeTrees)
			AddSearchlights(sky, front, config, input)
			wantSky, wantFront := slices.Clone(sky.Light), slices.Clone(front.Light)
			otherSky, otherFront, otherConfig, otherInput := searchFixture()
			otherConfig.Gain, otherConfig.Count, otherInput.Phase = 0.7, 3, 0.91
			AddSearchlights(otherSky, otherFront, otherConfig, otherInput)
			copy(sky.Light, original)
			clear(front.Light)
			AddSearchlights(sky, front, config, input)
			if !slices.Equal(sky.Light, wantSky) || !slices.Equal(front.Light, wantFront) {
				t.Fatal("another invocation changed output")
			}
			if !slices.Equal(input.Open, open) || !slices.Equal(input.CloseTrees, closeTrees) {
				t.Fatal("input masks mutated")
			}
		})
	}
}

func TestSearchViewerFrontRouting(t *testing.T) {
	sky, front, config, input := searchFixture()
	input.Move, input.ShoreRow = "viewer", 2.25
	original := slices.Clone(sky.Light)
	AddSearchlights(sky, front, config, input)
	if !slices.ContainsFunc(front.Light, func(v float64) bool { return v > 0 }) {
		t.Fatal("viewer beam did not reach front")
	}
	for y := 3; y < sky.Height; y++ {
		start := sky.At(0, y)
		end := start + sky.Width*3
		if !slices.Equal(sky.Light[start:end], original[start:end]) {
			t.Fatal("below-shore light entered sky")
		}
	}
	for i := range input.CloseTrees {
		input.CloseTrees[i] = true
	}
	copy(sky.Light, original)
	clear(front.Light)
	AddSearchlights(sky, front, config, input)
	if !slices.Equal(sky.Light, original) || slices.ContainsFunc(front.Light, func(v float64) bool { return v != 0 }) {
		t.Fatal("close trees failed to cut viewer beam")
	}
}

func TestHazeOcclusionDarkMaskBehindAndIsolation(t *testing.T) {
	for _, mode := range []string{"dark", "covered", "behind", "off", "zero-gain"} {
		t.Run(mode, func(t *testing.T) {
			front, trees, input := hazeFixture()
			original := slices.Clone(front.Light)
			switch mode {
			case "dark":
				clear(trees.Light)
			case "covered":
				for i := range trees.Light {
					trees.Light[i] = 1
				}
			case "behind":
				input.Camera.Forward = input.Camera.Forward.Scale(-1)
			case "off":
				input.Levels = [2]float64{0.009, 0.009}
			case "zero-gain":
				input.Gain = 0
			}
			mask := slices.Clone(trees.Light)
			AddBackHaze(front, trees, input)
			if !slices.Equal(front.Light, original) || !slices.Equal(trees.Light, mask) {
				t.Fatal("inactive or occluded haze mutated images")
			}
		})
	}
	front, trees, input := hazeFixture()
	original, mask := slices.Clone(front.Light), slices.Clone(trees.Light)
	AddBackHaze(front, trees, input)
	want := slices.Clone(front.Light)
	for i := 0; i < len(front.Light); i += 3 {
		if trees.Light[i] > 0 && !slices.Equal(front.Light[i:i+3], original[i:i+3]) {
			t.Fatal("tree pixel lit")
		}
	}
	other, otherTrees, otherInput := hazeFixture()
	otherInput.Gain, otherInput.Seconds = 2, 8
	AddBackHaze(other, otherTrees, otherInput)
	copy(front.Light, original)
	AddBackHaze(front, trees, input)
	if !slices.Equal(front.Light, want) || !slices.Equal(trees.Light, mask) {
		t.Fatal("inputs mutated or invocation state leaked")
	}
}
