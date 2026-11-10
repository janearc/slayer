package terrain

import (
	"context"
	"fmt"
	"math"
)

// HeightIndex retains complete native-sample extrema in bounded blocks.
// It is a visibility envelope, not a filtered terrain or a representation's
// geometric error. No-data contributes nothing; corrupt samples remain errors.
type HeightIndex struct {
	columns, rows, block, across     int
	west, north, eastStep, southStep float64
	minimum, maximum                 []float32
	present                          []bool
}

// BuildHeightIndex reads every native sample once, including intervening peaks
// which a sparse render grid could miss. The caller owns the serial source
// during construction. The returned index retains no raster or decoded tile.
// Payload bytes cover two float32 arrays and one availability byte per block;
// reader/cache/metadata storage and object overhead remain separate.
func (raster *Raster) BuildHeightIndex(ctx context.Context, block int, limitBytes int64) (*HeightIndex, error) {
	if ctx == nil || raster == nil || raster.closed || !raster.placed || block < 1 || limitBytes < 0 ||
		raster.reference.Model != 2 || raster.reference.AngularUnits != 9102 {
		return nil, fmt.Errorf("terrain: invalid geographic height-index request")
	}
	west, north, err := raster.SampleLocation(0, 0)
	if err != nil {
		return nil, err
	}
	across, down := (raster.width-1)/block+1, (raster.height-1)/block+1
	const bytesPerBlock = 2*sampleBytes + gridAvailabilityBytes
	if int64(across) > min(limitBytes, addressableBytes)/bytesPerBlock/int64(down) {
		return nil, fmt.Errorf("terrain: height index exceeds its byte budget")
	}
	count := across * down
	index := &HeightIndex{columns: raster.width, rows: raster.height,
		block: block, across: across, west: west, north: north,
		eastStep: raster.spacing, southStep: raster.rowSpacing(),
		minimum: make([]float32, count), maximum: make([]float32, count), present: make([]bool, count)}
	for tile := range raster.offsets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		samples, err := raster.tile(tile)
		if err != nil {
			return nil, err
		}
		firstColumn := tile % raster.across() * raster.tileW
		firstRow := tile / raster.across() * raster.tileH
		for row := 0; row < raster.tileH && firstRow+row < raster.height; row++ {
			for column := 0; column < raster.tileW && firstColumn+column < raster.width; column++ {
				value := samples[row*raster.tileW+column]
				if value == raster.noData || math.IsNaN(float64(value)) {
					continue
				}
				if math.IsInf(float64(value), 0) {
					return nil, fmt.Errorf("terrain: non-finite native sample in height index")
				}
				cell := (firstRow+row)/block*across + (firstColumn+column)/block
				if !index.present[cell] {
					index.minimum[cell], index.maximum[cell], index.present[cell] = value, value, true
				} else {
					index.minimum[cell] = min(index.minimum[cell], value)
					index.maximum[cell] = max(index.maximum[cell], value)
				}
			}
		}
	}
	return index, nil
}

// Range encloses all native samples that can contribute to a geographic
// rectangle's bilinear queries. Whole intersecting blocks and a sample halo
// make the bound conservative. Present means some source data, not guaranteed
// complete coverage; callers still handle missing heights explicitly.
func (index *HeightIndex) Range(west, south, east, north float64) (low, high float64, present bool, err error) {
	if index == nil || index.block < 1 || !finiteCoordinate(west) || !finiteCoordinate(east) ||
		!finiteCoordinate(south) || !finiteCoordinate(north) || west > east || south > north {
		return 0, 0, false, fmt.Errorf("terrain: invalid height-index region")
	}
	left, right := (west-index.west)/index.eastStep, (east-index.west)/index.eastStep
	top, bottom := (index.north-north)/index.southStep, (index.north-south)/index.southStep
	if right < 0 || bottom < 0 || left > float64(index.columns-1) || top > float64(index.rows-1) {
		return 0, 0, false, nil
	}
	firstColumn := int(max(0, math.Floor(left)-1)) / index.block
	lastColumn := int(min(float64(index.columns-1), math.Ceil(right)+1)) / index.block
	firstRow := int(max(0, math.Floor(top)-1)) / index.block
	lastRow := int(min(float64(index.rows-1), math.Ceil(bottom)+1)) / index.block
	for row := firstRow; row <= lastRow; row++ {
		for column := firstColumn; column <= lastColumn; column++ {
			cell := row*index.across + column
			if !index.present[cell] {
				continue
			}
			if !present {
				low, high, present = float64(index.minimum[cell]), float64(index.maximum[cell]), true
			} else {
				low, high = min(low, float64(index.minimum[cell])), max(high, float64(index.maximum[cell]))
			}
		}
	}
	return low, high, present, nil
}
