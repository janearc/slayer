// Package terrain reads measured elevation rasters and builds terrain from the
// required tiles. Tiled access avoids loading an entire large elevation model
// when a scene needs only a small region.
package terrain

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

// the tiff tags a raster is read from
const (
	tagWidth       = 256
	tagHeight      = 257
	tagBits        = 258
	tagCompression = 259
	tagSamples     = 277
	tagTileWidth   = 322
	tagTileHeight  = 323
	tagTileOffsets = 324
	tagTileCounts  = 325
	tagFormat      = 339
	tagScale       = 33550
	tagTiepoint    = 33922
	tagNoData      = 42113
)

// the values those tags must have for a raster this reads
const (
	uncompressed = 1
	lzwed        = 5
	floating     = 3
	floatBits    = 32
)

var errRaster = errors.New("terrain: not a raster this reads")

// A Raster is an elevation model on disk: its size in samples, its
// tiles, where its first sample's corner is, how far apart samples are,
// and what it says where it has no height.
type Raster struct {
	file            *os.File
	order           binary.ByteOrder
	big, placed     bool
	width, height   int
	tileW, tileH    int
	offsets, counts []uint64
	compression     int
	west, north     float64
	spacing         float64
	noData          float32
	cache           map[int][]float32
}

// Open is the raster in a file, its tags read and checked, its samples
// not read until asked for.
func Open(path string) (*Raster, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	raster := &Raster{file: file, cache: map[int][]float32{},
		noData: float32(math.Inf(-1))}
	if err := raster.readTags(); err != nil {
		file.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return raster, nil
}

// Close lets go of the file.
func (raster *Raster) Close() error { return raster.file.Close() }

// Size is the raster's width and height in samples.
func (raster *Raster) Size() (int, int) { return raster.width, raster.height }

// Corner is where the raster's first sample's corner is, as an easting
// and a northing in its projection's metres, and how far apart its
// samples are.
func (raster *Raster) Corner() (west, north, spacing float64) {
	return raster.west, raster.north, raster.spacing
}

// readTags reads the header and the first directory's tags.
func (raster *Raster) readTags() error {
	head := make([]byte, 8)
	if _, err := raster.file.ReadAt(head, 0); err != nil {
		return err
	}
	switch string(head[:2]) {
	case "II":
		raster.order = binary.LittleEndian
	case "MM":
		raster.order = binary.BigEndian
	default:
		return fmt.Errorf("%w: no byte order", errRaster)
	}
	first := int64(raster.order.Uint32(head[4:]))
	switch raster.order.Uint16(head[2:]) {
	case classic:
	case bigTiff:
		raster.big = true
		wide := make([]byte, 16)
		if _, err := raster.file.ReadAt(wide, 0); err != nil {
			return err
		}
		first = int64(raster.order.Uint64(wide[8:]))
	default:
		return fmt.Errorf("%w: not a tiff or a bigtiff", errRaster)
	}
	tags, err := raster.directory(first)
	if err != nil {
		return err
	}
	return raster.take(tags)
}

// a tiff's version, after its byte order: classic, with four-byte
// offsets, or bigtiff, with eight
const (
	classic = 42
	bigTiff = 43
)
