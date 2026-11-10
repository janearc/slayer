package terrain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// a tag's value as the directory holds it: its type, how many, and its
// bytes, read from beside the entry or from where it points
type entry struct {
	kind, count int
	raw         []byte
}

// sizes of the tiff types a raster's tags use, in bytes; 16 is
// bigtiff's eight-byte whole number
var sizes = map[int]int{1: 1, 2: 1, 3: 2, 4: 4, 11: 4, 12: 8, 16: 8}

// layout is how a directory is laid out: the bytes of its count, of
// each entry, and of the value an entry holds in itself.
func (raster *Raster) layout() (counted, each, inline int) {
	if raster.big {
		return 8, 20, 8
	}
	return 2, 12, 4
}

// word is an unsigned number of a directory's width at a place.
func (raster *Raster) word(raw []byte, wide int) uint64 {
	switch wide {
	case 8:
		return raster.order.Uint64(raw)
	case 4:
		return uint64(raster.order.Uint32(raw))
	}
	return uint64(raster.order.Uint16(raw))
}

// directory reads the tags of the directory at an offset.
func (raster *Raster) directory(at uint64) (map[int]entry, error) {
	counted, each, inline := raster.layout()
	head, err := raster.readMetadata(at, counted)
	if err != nil {
		return nil, err
	}
	many := raster.word(head, counted)
	room := raster.budget.MetadataBytes - raster.metadataBytes
	size, err := allocation(many, uint64(each), room)
	if err != nil {
		return nil, fmt.Errorf("terrain: TIFF directory byte budget: %w", err)
	}
	table, err := raster.readMetadata(at+uint64(counted), size)
	if err != nil {
		return nil, err
	}
	tags := map[int]entry{}
	for index := range int(many) {
		row := table[each*index : each*(index+1)]
		tag := int(raster.order.Uint16(row))
		kind := int(raster.order.Uint16(row[2:]))
		count := raster.word(row[4:], inline)
		width, known := sizes[kind]
		if !known {
			continue
		}
		raw := row[4+inline : 4+2*inline]
		size, err := allocation(count, uint64(width),
			raster.budget.MetadataBytes)
		if err != nil {
			return nil, fmt.Errorf("terrain: TIFF tag %d byte budget: %w", tag, err)
		}
		if size > inline {
			offset := raster.word(row[4+inline:], inline)
			raw, err = raster.readMetadata(offset, size)
			if err != nil {
				return nil, err
			}
		}
		tags[tag] = entry{kind: kind, count: int(count), raw: raw}
	}
	return tags, nil
}

// numbers is a tag's values as unsigned whole numbers.
func (raster *Raster) numbers(value entry) []uint64 {
	out := make([]uint64, 0, value.count)
	for i := range value.count {
		switch value.kind {
		case 3:
			out = append(out, uint64(raster.order.Uint16(value.raw[2*i:])))
		case 4:
			out = append(out, uint64(raster.order.Uint32(value.raw[4*i:])))
		case 16:
			out = append(out, raster.order.Uint64(value.raw[8*i:]))
		case 1:
			out = append(out, uint64(value.raw[i]))
		}
	}
	return out
}

// reals is a tag's values as doubles.
func (raster *Raster) reals(value entry) []float64 {
	out := make([]float64, 0, value.count)
	for i := range value.count {
		if value.kind == 12 {
			bits := raster.order.Uint64(value.raw[8*i:])
			out = append(out, math.Float64frombits(bits))
		}
	}
	return out
}

// one is a tag's single whole number, or minus one where there is none.
func (raster *Raster) one(tags map[int]entry, tag int) int {
	value, found := tags[tag]
	if !found {
		return -1
	}
	all := raster.numbers(value)
	if len(all) == 0 {
		return -1
	}
	return int(all[0])
}

// take keeps what the tags say, refusing a raster that is not one band
// of float32 heights in tiles or explicitly sized strips, uncompressed or
// lzw, placed by positive pixel scales and a tiepoint at its first sample.
// Projected legacy grids retain square scales; geographic grids retain both.
func (raster *Raster) take(tags map[int]entry) error {
	raster.width, raster.height = raster.one(tags, tagWidth),
		raster.one(tags, tagHeight)
	raster.tileW, raster.tileH = raster.one(tags, tagTileWidth),
		raster.one(tags, tagTileHeight)
	_, tiledWidth := tags[tagTileWidth]
	_, tiledHeight := tags[tagTileHeight]
	if !tiledWidth && !tiledHeight {
		rows := raster.one(tags, tagRowsPerStrip)
		if rows > 0 {
			raster.stripped = true
			raster.tileW, raster.tileH = raster.width, min(rows, raster.height)
		}
	}
	raster.compression = raster.one(tags, tagCompression)
	if _, present := tags[tagFillOrder]; present && raster.one(tags, tagFillOrder) != 1 {
		order := raster.one(tags, tagFillOrder)
		return fmt.Errorf("%w: fill order %d", errRaster, order)
	}
	raster.predictor = raster.one(tags, tagPredictor)
	if _, present := tags[tagPredictor]; !present {
		raster.predictor = 1
	}
	if raster.predictor != 1 && raster.predictor != 3 {
		return fmt.Errorf("%w: predictor %d", errRaster, raster.predictor)
	}

	switch {
	case raster.width < 1 || raster.height < 1:
		return fmt.Errorf("%w: no size", errRaster)
	case raster.tileW < 1 || raster.tileH < 1:
		return fmt.Errorf("%w: not tiled", errRaster)
	case raster.one(tags, tagBits) != floatBits ||
		raster.one(tags, tagFormat) != floating:
		return fmt.Errorf("%w: not float32 samples", errRaster)
	case raster.one(tags, tagSamples) > 1:
		return fmt.Errorf("%w: more than one band", errRaster)
	case raster.compression != uncompressed && raster.compression != lzwed:
		return fmt.Errorf("%w: compression %d", errRaster,
			raster.compression)
	}
	expected, err := raster.tiles()
	if err != nil {
		return err
	}
	raster.offsets = raster.numbers(tags[tagTileOffsets])
	raster.counts = raster.numbers(tags[tagTileCounts])
	if raster.stripped {
		raster.offsets = raster.numbers(tags[tagStripOffsets])
		raster.counts = raster.numbers(tags[tagStripCounts])
	}
	if len(raster.offsets) != expected || len(raster.counts) != expected {
		return fmt.Errorf("%w: %d tile offsets and %d counts for %d tiles",
			errRaster, len(raster.offsets), len(raster.counts), expected)
	}
	if err := raster.readReference(tags); err != nil {
		return err
	}
	return raster.place(tags)
}

// place keeps where the raster stands and what marks no height. a
// raster with neither a scale nor a tiepoint is not placed: its samples
// can be read, and it refuses a window. one with them wrong is refused.
func (raster *Raster) place(tags map[int]entry) error {
	_, scaled := tags[tagScale]
	_, tied := tags[tagTiepoint]
	if scaled || tied {
		scale := raster.reals(tags[tagScale])
		tie := raster.reals(tags[tagTiepoint])
		if len(scale) < 2 || len(tie) < 6 || !(scale[0] > 0) || !(scale[1] > 0) ||
			(raster.reference.Model != 2 && scale[0] != scale[1]) || tie[0] != 0 || tie[1] != 0 ||
			!finiteCoordinate(scale[0]) || !finiteCoordinate(scale[1]) || !finiteCoordinate(tie[3]) ||
			!finiteCoordinate(tie[4]) {
			return fmt.Errorf("%w: not placed by compatible positive scales and a "+
				"tiepoint at its first sample", errRaster)
		}
		raster.spacing, raster.west, raster.north = scale[0], tie[3], tie[4]
		raster.rowStep = scale[1]
		raster.placed = true
		coverage, _ := raster.Coverage()
		if !finiteCoordinate(coverage.East) || !finiteCoordinate(coverage.South) {
			return fmt.Errorf("%w: projected extent overflows", errRaster)
		}
	}
	if value, found := tags[tagNoData]; found {
		text := strings.Trim(string(value.raw), "\x00 ")
		none, err := strconv.ParseFloat(text, 32)
		if err != nil {
			return fmt.Errorf("%w: no-data %q", errRaster, text)
		}
		raster.noData = float32(none)
	}
	return nil
}

// tiles checks a grid's product before using it as a directory length.
// Every tile needs one offset and one byte count in uint64 storage.
func (raster *Raster) tiles() (int, error) {
	rows := (raster.height-1)/raster.tileH + 1
	limit := raster.budget.MetadataBytes / tileReferenceBytes
	count, err := allocation(uint64(raster.across()), uint64(rows), limit)
	if err != nil {
		return 0, fmt.Errorf("terrain: tile directory capacity: %w", err)
	}
	return count, nil
}

// tileReferenceBytes is defined by one uint64 offset and one uint64 count.
const tileReferenceBytes = 2 * 64 / byteBits

// across avoids overflow from adding two large source dimensions.
func (raster *Raster) across() int {
	return (raster.width-1)/raster.tileW + 1
}
