package terrain

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// tiffTag is one directory entry as a test writes it: its tag, its type,
// its value count and the value bytes in the file's byte order.
type tiffTag struct {
	id, kind uint16
	count    uint64
	raw      []byte
}

// tiffSpec describes a small raster file. Tags default to a one-band float32
// uncompressed raster; a test replaces or drops tags to build a bad file.
type tiffSpec struct {
	order           tiffOrder
	big             bool
	tags            map[uint16]tiffTag
	tiles           [][]byte
	offsets, counts []uint64
	width, height   int
	tileW, tileH    int
}

// newTiffSpec starts a spec whose tiles hold the heights from at, padded with
// zero beyond the raster's edge, in the given byte order.
func newTiffSpec(order tiffOrder, big bool, width, height, tileW, tileH int,
	at func(column, row int) float32,
) *tiffSpec {
	spec := &tiffSpec{order: order, big: big, tags: map[uint16]tiffTag{},
		width: width, height: height, tileW: tileW, tileH: tileH}
	across, down := (width+tileW-1)/tileW, (height+tileH-1)/tileH
	for tileRow := range down {
		for tileColumn := range across {
			var raw []byte
			for y := range tileH {
				for x := range tileW {
					column, row := tileColumn*tileW+x, tileRow*tileH+y
					value := float32(0)
					if column < width && row < height {
						value = at(column, row)
					}
					raw = order.AppendUint32(raw, math.Float32bits(value))
				}
			}
			spec.tiles = append(spec.tiles, raw)
		}
	}
	spec.shorts(tagWidth, uint16(width))
	spec.shorts(tagHeight, uint16(height))
	spec.shorts(tagBits, 32)
	spec.shorts(tagCompression, 1)
	spec.shorts(tagSamples, 1)
	spec.shorts(tagTileWidth, uint16(tileW))
	spec.shorts(tagTileHeight, uint16(tileH))
	spec.shorts(tagFormat, 3)
	return spec
}

// shorts sets a tag of 16-bit whole numbers.
func (spec *tiffSpec) shorts(id uint16, values ...uint16) *tiffSpec {
	var raw []byte
	for _, value := range values {
		raw = spec.order.AppendUint16(raw, value)
	}
	spec.tags[id] = tiffTag{id: id, kind: 3, count: uint64(len(values)), raw: raw}
	return spec
}

// doubles sets a tag of 64-bit floats.
func (spec *tiffSpec) doubles(id uint16, values ...float64) *tiffSpec {
	var raw []byte
	for _, value := range values {
		raw = spec.order.AppendUint64(raw, math.Float64bits(value))
	}
	spec.tags[id] = tiffTag{id: id, kind: 12, count: uint64(len(values)), raw: raw}
	return spec
}

// ascii sets a text tag, with its closing zero byte.
func (spec *tiffSpec) ascii(id uint16, text string) *tiffSpec {
	spec.tags[id] = tiffTag{id: id, kind: 2, count: uint64(len(text) + 1),
		raw: append([]byte(text), 0)}
	return spec
}

// place sets the pixel scales and a tiepoint at the first sample.
func (spec *tiffSpec) place(west, north, across, down float64) *tiffSpec {
	spec.doubles(tagScale, across, down, 0)
	return spec.doubles(tagTiepoint, 0, 0, 0, west, north, 0)
}

// geo sets a GeoKey directory of key, location, count, value rows.
func (spec *tiffSpec) geo(rows ...[4]uint16) *tiffSpec {
	values := []uint16{1, 1, 0, uint16(len(rows))}
	for _, row := range rows {
		values = append(values, row[:]...)
	}
	return spec.shorts(34735, values...)
}

// degrees declares a geographic degree grid with the given pixel registration.
func (spec *tiffSpec) degrees(registration uint16) *tiffSpec {
	return spec.geo([4]uint16{1024, 0, 1, 2}, [4]uint16{1025, 0, 1, registration},
		[4]uint16{2048, 0, 1, 4326}, [4]uint16{2054, 0, 1, 9102})
}

// write builds the file in a temporary directory and returns its path.
func (spec *tiffSpec) write(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "made.tif")
	if err := os.WriteFile(path, spec.bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// bytes lays the file out: header, tile payloads, large tag values, directory.
func (spec *tiffSpec) bytes() []byte {
	head, inline := 8, 4
	if spec.big {
		head, inline = 16, 8
	}
	body := make([]byte, head)
	var offsets, counts []uint64
	for _, tile := range spec.tiles {
		offsets = append(offsets, uint64(len(body)))
		counts = append(counts, uint64(len(tile)))
		body = append(body, tile...)
	}
	if spec.offsets != nil {
		offsets = spec.offsets
	}
	if spec.counts != nil {
		counts = spec.counts
	}
	tags := map[uint16]tiffTag{}
	for id, tag := range spec.tags {
		tags[id] = tag
	}
	kind := uint16(4)
	if spec.big {
		kind = 16
	}
	number := func(value uint64) []byte {
		if spec.big {
			return spec.order.AppendUint64(nil, value)
		}
		return spec.order.AppendUint32(nil, uint32(value))
	}
	pack := func(values []uint64) []byte {
		var raw []byte
		for _, value := range values {
			raw = append(raw, number(value)...)
		}
		return raw
	}
	_, stripped := tags[tagRowsPerStrip]
	offsetTag, countTag := uint16(tagTileOffsets), uint16(tagTileCounts)
	if stripped {
		offsetTag, countTag = tagStripOffsets, tagStripCounts
	}
	tags[offsetTag] = tiffTag{id: offsetTag, kind: kind, count: uint64(len(offsets)), raw: pack(offsets)}
	tags[countTag] = tiffTag{id: countTag, kind: kind, count: uint64(len(counts)), raw: pack(counts)}
	ids := make([]int, 0, len(tags))
	for id := range tags {
		ids = append(ids, int(id))
	}
	slices.Sort(ids)
	var table []byte
	for _, id := range ids {
		tag := tags[uint16(id)]
		table = spec.order.AppendUint16(table, tag.id)
		table = spec.order.AppendUint16(table, tag.kind)
		table = append(table, number(tag.count)...)
		value := tag.raw
		if len(value) > inline {
			table = append(table, number(uint64(len(body)))...)
			body = append(body, value...)
			continue
		}
		table = append(table, value...)
		table = append(table, make([]byte, inline-len(value))...)
	}
	at := uint64(len(body))
	if spec.big {
		body = spec.order.AppendUint64(body, uint64(len(ids)))
	} else {
		body = spec.order.AppendUint16(body, uint16(len(ids)))
	}
	body = append(body, table...)
	body = append(body, make([]byte, inline)...)
	if spec.order == binary.BigEndian {
		copy(body, "MM")
	} else {
		copy(body, "II")
	}
	if spec.big {
		spec.order.PutUint16(body[2:], bigTiff)
		spec.order.PutUint16(body[4:], 8)
		spec.order.PutUint64(body[8:], at)
	} else {
		spec.order.PutUint16(body[2:], classic)
		spec.order.PutUint32(body[4:], uint32(at))
	}
	return body
}

// openSpec writes the spec, opens it and fails the test on a refusal.
func openSpec(t *testing.T, spec *tiffSpec) *Raster {
	t.Helper()
	raster, err := Open(spec.write(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raster.Close() })
	return raster
}

// rampAt is the height column + 10 row, exact in a float32.
func rampAt(column, row int) float32 { return float32(column + 10*row) }

// tiffOrder is a byte order that can also append, as the encoding package's
// two orders do.
type tiffOrder interface {
	binary.ByteOrder
	binary.AppendByteOrder
}

// tiffLittle is the little-endian order as a tiffOrder.
var tiffLittle tiffOrder = binary.LittleEndian
