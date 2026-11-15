//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func flatLayers() []MarchLayer {
	layers := make([]MarchLayer, 3)
	for index, extent := range []float64{4000, 2000, 1000} {
		layers[index] = MarchLayer{Field: &oceantex.Field{N: 2, Extent: extent, Height: make([]float64, 4), PushX: make([]float64, 4), PushY: make([]float64, 4)}}
	}
	return layers
}

func TestMarchFlatAndUpwardRays(t *testing.T) {
	rays := []MarchRay{{Ray: oceantex.Ray{At: oceantex.Vec{Z: 10}, Dir: oceantex.Vec{Z: -1}}, PixelAngle: 0.01}, {Ray: oceantex.Ray{At: oceantex.Vec{Z: 10}, Dir: oceantex.Vec{Z: 1}}, PixelAngle: 0.01}}
	got, err := March(flatLayers(), rays)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(got[0].Distance)-10) > 1e-4 || got[0].Sky {
		t.Fatalf("downward ray: %+v", got[0])
	}
	if !got[1].Sky {
		t.Fatalf("upward ray: %+v", got[1])
	}
}

func TestMarchRespondsToWaveHeight(t *testing.T) {
	layers := flatLayers()
	for index := range layers[0].Field.Height {
		layers[0].Field.Height[index] = 2
	}
	layers[0].Reach = 2
	ray := MarchRay{Ray: oceantex.Ray{At: oceantex.Vec{Z: 10}, Dir: oceantex.Vec{Z: -1}}, PixelAngle: 0.01}
	got, err := March(layers, []MarchRay{ray})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(got[0].Distance)-8) > 1e-3 || got[0].Sky {
		t.Fatalf("wave height: %+v", got[0])
	}
}

func TestMarchRejectsMissingLayer(t *testing.T) {
	if _, err := March(flatLayers()[:2], []MarchRay{{}}); err == nil {
		t.Fatal("missing cascade accepted")
	}
}

func TestMarchRejectsUnorderedLayers(t *testing.T) {
	layers := flatLayers()
	layers[0], layers[2] = layers[2], layers[0]
	ray := MarchRay{Ray: oceantex.Ray{At: oceantex.Vec{Z: 10}, Dir: oceantex.Vec{Z: -1}}, PixelAngle: 0.01}
	if _, err := March(layers, []MarchRay{ray}); err == nil {
		t.Fatal("unordered layers accepted")
	}
}

func TestMarchRejectsUnboundedWave(t *testing.T) {
	layers := flatLayers()
	layers[0].Field.Height[0] = 2
	ray := MarchRay{Ray: oceantex.Ray{At: oceantex.Vec{Z: 10}, Dir: oceantex.Vec{Z: -1}}, PixelAngle: 0.01}
	if _, err := March(layers, []MarchRay{ray}); err == nil {
		t.Fatal("incorrect wave reach accepted")
	}
}
