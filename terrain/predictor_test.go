package terrain

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
)

func TestFloatingPredictorRowsUseCanonicalBytePlanes(t *testing.T) {
	// IEEE float32 1,2 and -1,0.5, arranged most-significant plane first
	// then differenced across each entire scanline modulo 256.
	encoded := []byte{0x3f, 1, 0x40, 0x80, 0, 0, 0, 0,
		0xbf, 0x80, 0x41, 0x80, 0, 0, 0, 0}
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		r := Raster{tileW: 2, order: order, predictor: 3}
		got, err := r.decode(bytes.NewReader(encoded), 4)
		if err != nil || !slices.Equal(got, []float32{1, 2, -1, 0.5}) {
			t.Fatalf("planes %v, %v", got, err)
		}
		if _, err := r.decode(bytes.NewReader(encoded[:15]), 4); err == nil {
			t.Fatal("truncated scanline accepted")
		}
	}
}

func TestFloatingPredictorMatchesIndependentLibtiffFiles(t *testing.T) {
	for _, name := range []string{"fixture-float-predictor-le.tif", "fixture-float-predictor-be.tif"} {
		r, err := Open("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		for row := 0; row < 16; row++ {
			for column := 0; column < 16; column++ {
				got, has, err := r.sample(column, row)
				want := float64(row*16+column) - 0.5
				if err != nil || !has || got != want {
					t.Fatalf("%s at %d,%d: %v %v %v; want %v",
						name, column, row, got, has, err, want)
				}
			}
		}
		r.Close()
	}
}

func TestFloatingPredictorAccountsForScanlineBuffer(t *testing.T) {
	budget, err := DefaultRasterBudget()
	if err != nil {
		t.Fatal(err)
	}
	// 16*16 float32 samples plus one 16*4-byte scanline.
	budget.TileBytes = 1087
	r, err := OpenWithBudget("testdata/fixture-float-predictor-le.tif", budget)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, _, err := r.sample(0, 0); err == nil {
		t.Fatal("predictor row buffer exceeded tile budget")
	}
}
