//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestFireMatchesCPUAndRespectsTrees(t *testing.T) {
	config := lights.DefaultFireConfig()
	config.Stops = 0
	frame := lights.Image{Width: 2, Height: 1, Light: []float64{1, 2, 3, 4, 5, 6}}
	input := lights.FireInput{Width: 2, Height: 1, Cues: lights.FireCues{Level: 1, Lit: 0, Since: []float64{0}}, Frames: [3]lights.FireFrames{{Count: 1, Images: map[int]lights.Image{0: frame}}}, CloseTrees: []bool{false, true}}
	want, err := lights.RenderFire(config, input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Fire(config, input)
	if err != nil {
		t.Fatal(err)
	}
	for index := range got.Light {
		if math.Abs(got.Light[index]-want.Light[index]) > 1e-5 {
			t.Fatalf("channel %d: GPU %v CPU %v", index, got.Light[index], want.Light[index])
		}
	}
	if got.Light[0] <= 0 || got.Light[3] != 0 {
		t.Fatalf("fire mask: %v", got.Light)
	}
}

func TestFireRejectsMissingSelectedFrame(t *testing.T) {
	config := lights.DefaultFireConfig()
	input := lights.FireInput{Width: 1, Height: 1, Cues: lights.FireCues{Level: 1, Lit: 0, Since: []float64{0}}, Frames: [3]lights.FireFrames{{Count: 1}}}
	if _, err := Fire(config, input); err == nil {
		t.Fatal("missing fire frame accepted")
	}
}
