package gpuocean

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/oceantex"
)

type WaveLayer struct {
	Field          *oceantex.Field
	Reach          float64
	ResolvedAlong  float64
	ResolvedAcross float64
}

// MarchLayers keeps the Field and Reach of each of three wave layers, dropping
// their resolved variances. It refuses other than three, a layer with no
// Field, and a Reach that is negative or not finite.
func MarchLayers(layers []WaveLayer) ([]MarchLayer, error) {
	if len(layers) != 3 {
		return nil, errors.New("gpuocean: expected three wave layers")
	}
	result := make([]MarchLayer, len(layers))
	for index, layer := range layers {
		if layer.Field == nil || !rayFinite(layer.Reach) || layer.Reach < 0 {
			return nil, errors.New("gpuocean: invalid wave layer")
		}
		result[index] = MarchLayer{Field: layer.Field, Reach: layer.Reach}
	}
	return result, nil
}

// Waves builds a wave layer for each cascade oceantex.WaveCascades gives for
// sea at the given seconds and seed. It refuses a time that is negative or
// not finite.
func Waves(sea oceantex.Sea, seconds float64, seed uint64) ([]WaveLayer, error) {
	if !rayFinite(seconds) || seconds < 0 {
		return nil, errors.New("gpuocean: invalid wave time")
	}
	cascades, err := oceantex.WaveCascades(sea, seconds, seed)
	if err != nil {
		return nil, fmt.Errorf("gpuocean: waves: %w", err)
	}
	layers := make([]WaveLayer, len(cascades))
	for index, cascade := range cascades {
		layers[index] = WaveLayer{Field: cascade.Field, Reach: cascade.Reach, ResolvedAlong: cascade.ResolvedAlong, ResolvedAcross: cascade.ResolvedAcross}
	}
	return layers, nil
}
