package terrain

import (
	"fmt"
	"io"
	"math"

	"golang.org/x/image/tiff/lzw"
)

// tile reads directly from the file section, without keeping compressed bytes.
// The decoded payload is checked before allocation and read incrementally.
func (raster *Raster) tile(index int) ([]float32, error) {
	if samples, found := raster.cache.get(index); found {
		return samples, nil
	}
	room := raster.budget.TileBytes
	if raster.predictor == 3 {
		row, err := allocation(uint64(raster.tileW), sampleBytes, room)
		if err != nil {
			return nil, err
		}
		room -= int64(row)
	}
	rows := raster.tileH
	// TIFF strips are not padded to RowsPerStrip at the image's last strip.
	// Tiled images retain their complete padded tile payloads. See the installed
	// libtiff TIFFReadEncodedStrip manual and TIFFTAG_ROWSPERSTRIP in tiff.h.
	if raster.stripped {
		rows = min(rows, raster.height-index*raster.tileH)
	}
	count, err := allocation(uint64(raster.tileW), uint64(rows),
		room/sampleBytes)
	if err != nil {
		return nil, fmt.Errorf("terrain: tile %d: %w", index, err)
	}
	section, err := raster.section(raster.offsets[index], raster.counts[index])
	if err != nil {
		return nil, err
	}
	var input io.Reader = section
	if raster.compression == lzwed {
		compressed := lzw.NewReader(section, lzw.MSB, byteBits)
		defer compressed.Close()
		input = compressed
	}
	buffered := newSampleReader(input)
	samples, err := raster.decode(buffered, count)
	if err != nil {
		return nil, fmt.Errorf("terrain: tile %d: %w", index, err)
	}
	if raster.compression == lzwed {
		if err := compressedEnd(buffered); err != nil {
			return nil, fmt.Errorf("terrain: tile %d: %w", index, err)
		}
	}
	raster.cache.put(index, samples)

	return samples, nil
}

// decode reads one encoded float at a time through a fixed-size reader buffer.
// It never drains unbounded extra output from a corrupt compressed stream.
func (raster *Raster) decode(input io.Reader, count int) ([]float32, error) {
	if raster.predictor == 3 {
		return raster.decodeFloating(input, count)
	}
	samples := make([]float32, count)
	var word [sampleBytes]byte
	for index := range samples {
		if _, err := io.ReadFull(input, word[:]); err != nil {
			return nil, err
		}
		samples[index] = math.Float32frombits(raster.order.Uint32(word[:]))
	}
	return samples, nil
}

// compressedEnd checks the end code without draining unbounded extra output.
// TIFF's declared tile grid already includes the padded edge samples.
func compressedEnd(input io.Reader) error {
	var extra [1]byte
	count, err := io.ReadFull(input, extra[:])
	if count != 0 {
		return fmt.Errorf("decoded output exceeds the declared tile grid")
	}
	if err != io.EOF {
		return fmt.Errorf("missing compressed tile end: %w", err)
	}
	return nil
}
