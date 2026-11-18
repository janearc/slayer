package terrain

import (
	"encoding/binary"
	"testing"
)

// TestTiffBuilderRoundTrips checks that the test builder's files, in each
// byte order and width, read back with the heights they were given.
func TestTiffBuilderRoundTrips(t *testing.T) {
	for _, order := range []tiffOrder{binary.LittleEndian, binary.BigEndian} {
		for _, big := range []bool{false, true} {
			raster := openSpec(t, newTiffSpec(order, big, 5, 4, 4, 4, rampAt).
				place(100, 50, 1, 1))
			for row := range 4 {
				for column := range 5 {
					got, has, err := raster.sample(column, row)
					if err != nil || !has || got != float64(rampAt(column, row)) {
						t.Fatalf("%v big=%v %d,%d: %v %v %v", order, big, column, row, got, has, err)
					}
				}
			}
		}
	}
}
