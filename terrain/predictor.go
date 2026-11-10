package terrain

import (
	"fmt"
	"io"
	"math"
)

// decodeFloating reverses TIFF floating-point prediction, one scanline at a
// time. Cumulative byte differences span the entire row, including plane
// boundaries. Its four planes are most-significant first, independent of the
// TIFF directory byte order. See libtiff 4.7.2 tif_predict.c, fpAcc and fpDiff,
// and the retained source/provenance described in doc/terrain-streaming.md.
func (raster *Raster) decodeFloating(input io.Reader, count int) ([]float32, error) {
	if raster.tileW < 1 || count < 0 || count%raster.tileW != 0 {
		return nil, fmt.Errorf("terrain: invalid floating predictor scanline")
	}
	row := make([]byte, raster.tileW*sampleBytes)
	samples := make([]float32, count)
	for start := 0; start < count; start += raster.tileW {
		if _, err := io.ReadFull(input, row); err != nil {
			return nil, err
		}
		for i := 1; i < len(row); i++ {
			row[i] += row[i-1]
		}
		for x := 0; x < raster.tileW; x++ {
			word := uint32(row[x])<<24 |
				uint32(row[raster.tileW+x])<<16 |
				uint32(row[2*raster.tileW+x])<<8 |
				uint32(row[3*raster.tileW+x])
			samples[start+x] = math.Float32frombits(word)
		}
	}
	return samples, nil
}
