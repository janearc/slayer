package gpulights

import (
	"errors"
	"math"
)

const splatTileSize = 32
const maxSplatTileReferences = 1 << 24

type splatTiles struct {
	columns, rows int
	offsets       []uint32
	indices       []uint32
}

// binSplats sorts splats into square tiles of splatTileSize pixels. Each splat
// goes in every tile its reach touches, three sigma out, or 2.5 if SkyOnly,
// and a splat wholly off the frame goes in none. It refuses more than
// maxSplatTileReferences entries in all.
func binSplats(width, height int, splats []Splat) (splatTiles, error) {
	columns, rows := (width+splatTileSize-1)/splatTileSize, (height+splatTileSize-1)/splatTileSize
	counts := make([]uint32, columns*rows)
	bounds := func(splat Splat) (int, int, int, int, bool) {
		radiusScale := 3.0
		if splat.SkyOnly {
			radiusScale = 2.5
		}
		radius := int(math.Ceil(radiusScale*splat.Sigma)) + 1
		centerX, centerY := int(splat.X), int(splat.Y)
		left, right := max(0, centerX-radius), min(width-1, centerX+radius)
		top, bottom := max(0, centerY-radius), min(height-1, centerY+radius)
		if left > right || top > bottom {
			return 0, 0, 0, 0, false
		}
		return left / splatTileSize, right / splatTileSize, top / splatTileSize, bottom / splatTileSize, true
	}
	total := 0
	for _, splat := range splats {
		left, right, top, bottom, visible := bounds(splat)
		if !visible {
			continue
		}
		for row := top; row <= bottom; row++ {
			for column := left; column <= right; column++ {
				if total >= maxSplatTileReferences {
					return splatTiles{}, errors.New("gpulights: splat tile reference limit exceeded")
				}
				counts[row*columns+column]++
				total++
			}
		}
	}
	offsets := make([]uint32, len(counts)+1)
	for index, count := range counts {
		offsets[index+1] = offsets[index] + count
	}
	indices := make([]uint32, total)
	cursor := append([]uint32(nil), offsets[:len(counts)]...)
	for index, splat := range splats {
		left, right, top, bottom, visible := bounds(splat)
		if !visible {
			continue
		}
		for row := top; row <= bottom; row++ {
			for column := left; column <= right; column++ {
				tile := row*columns + column
				indices[cursor[tile]] = uint32(index)
				cursor[tile]++
			}
		}
	}
	return splatTiles{columns: columns, rows: rows, offsets: offsets, indices: indices}, nil
}
