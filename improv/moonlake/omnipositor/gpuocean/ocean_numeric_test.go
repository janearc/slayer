//go:build draw

package gpuocean

import (
	"context"
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/ocean"
)

func TestGPUOceanSmallFrameAgainstCPU(t *testing.T) {
	if testing.Short() {
		t.Skip("ocean image gate requires the render queue")
	}
	config := ocean.DefaultConfig()
	config.Width, config.Height, config.Seconds, config.Wind = 4, 4, 1, 8
	image := ocean.RGB{Width: 4, Height: 4, Light: make([]float32, 4*4*3)}
	for index := range image.Light {
		image.Light[index] = float32(1+index%7) * 0.01
	}
	cpu, err := ocean.Render(config, image, nil)
	if err != nil {
		t.Fatal(err)
	}
	gpu, err := Render(context.Background(), config, image, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gpu.Width != cpu.Width || gpu.Height != cpu.Height {
		t.Fatal("ocean dimensions differ")
	}
	for index := range cpu.Light {
		for channel := range cpu.Light[index] {
			want, got := cpu.Light[index][channel], gpu.Light[index][channel]
			if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 1e-3+0.02*math.Abs(want) {
				t.Fatalf("pixel %d channel %d: GPU %v CPU %v", index, channel, got, want)
			}
		}
	}
}
