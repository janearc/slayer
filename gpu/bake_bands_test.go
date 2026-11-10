//go:build darwin

package gpu

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// The noncubic fixture has three six-cell planes. Two arrive together,
// followed by a short final band. Offsets below count float32 words.
const bandPlane, bandCells, bandFirst = 6, 18, 3

// bandWords includes signed zero, subnormal, infinity and a NaN payload.
// The remaining distinct words expose channel and plane reordering.
func bandWords(count int) []uint32 {
	words := make([]uint32, count)
	for index := range words {
		words[index] = 0x3f000001 + uint32(index)
	}
	copy(words, []uint32{0, 0x80000000, 1, 0x7f800000,
		0xff800000, 0x7fc12345, 0x3f800001, 0xbf000001})
	return words
}

func bandSamples(words []uint32) []float32 {
	samples := make([]float32, len(words))
	for index, word := range words {
		samples[index] = math.Float32frombits(word)
	}
	return samples
}

// expectBandWords builds expected bytes from authored words, independently
// of either production writer's float32 conversion and addressing state.
func expectBandWords(destination []byte, first int, words []uint32) {
	for index, word := range words {
		binary.LittleEndian.PutUint32(destination[4*(first+index):], word)
	}
}

func sameBandBytes(test *testing.T, got, want []byte) {
	test.Helper()
	if !bytes.Equal(got, want) {
		test.Fatal("band changed bits, order or a neighbouring reservation")
	}
}

func TestPackedBandsKeepExactBitsAndNeighbourReservations(test *testing.T) {
	for _, channels := range []int{1, 3} {
		checkPackedBands(test, channels)
	}
}

func checkPackedBands(test *testing.T, channels int) {
	test.Helper()
	destination, expected := bandNeighbour(test, channels)
	writer, err := newPackedBandWriter(destination, bandFirst,
		[3]int{2, 3, 3}, channels)
	if err != nil {
		test.Fatal(err)
	}
	streamPackedBands(test, writer, expected, channels)
}

func bandNeighbour(test *testing.T, channels int) ([]byte, []byte) {
	test.Helper()
	// A second RGB job lies beyond a two-word gap, with guards outside both.
	neighbour := bandFirst + bandCells*channels + 2
	destination := bytes.Repeat([]byte{0xa5}, 4*(neighbour+6+4))
	expected := bytes.Clone(destination)
	next, err := newPackedBandWriter(destination, neighbour,
		[3]int{1, 2, 1}, 3)
	if err != nil {
		test.Fatal(err)
	}
	words := []uint32{0x41000001, 0x41000002, 0x41000003,
		0x41000004, 0x41000005, 0x41000006}
	if err := next.consume(0, 1, bandSamples(words)); err != nil {
		test.Fatal(err)
	}
	expectBandWords(expected, neighbour, words)
	sameBandBytes(test, destination, expected)
	return destination, expected
}

func streamPackedBands(test *testing.T, writer *packedBandWriter,
	expected []byte, channels int,
) {
	test.Helper()
	words := bandWords(bandCells * channels)
	samples := bandSamples(words)
	cut := 2 * bandPlane * channels
	if err := writer.consume(0, 2, samples[:cut]); err != nil {
		test.Fatal(err)
	}
	expectBandWords(expected, bandFirst, words[:cut])
	sameBandBytes(test, writer.bytes, expected)
	rejectDuplicateBand(test, writer, samples[:bandPlane*channels], expected)
	if err := writer.consume(2, 1, samples[cut:]); err != nil {
		test.Fatal(err)
	}
	expectBandWords(expected, bandFirst+cut, words[cut:])
	sameBandBytes(test, writer.bytes, expected)
}

func rejectDuplicateBand(test *testing.T, writer *packedBandWriter,
	samples []float32, expected []byte,
) {
	test.Helper()
	if err := writer.consume(0, 1, samples); err == nil {
		test.Fatal("duplicate first band accepted")
	}
	sameBandBytes(test, writer.bytes, expected)
}

func TestPackedBandRefusalsDoNotWritePartialBytes(test *testing.T) {
	destination := bytes.Repeat([]byte{0xa5}, 4*(bandFirst+bandCells+4))
	writer, err := newPackedBandWriter(destination, bandFirst,
		[3]int{2, 3, 3}, 1)
	if err != nil {
		test.Fatal(err)
	}
	rejectMalformedBands(test, writer.consume, destination, 1)
}

func rejectMalformedBands(test *testing.T,
	consume func(int, int, []float32) error,
	destination []byte, channels int,
) {
	test.Helper()
	before := bytes.Clone(destination)
	plane := bandPlane * channels
	for _, band := range [][3]int{
		{1, 1, plane}, {-1, 1, plane}, {0, 0, 0},
		{0, 4, 4 * plane}, {0, 1, plane - 1},
	} {
		err := consume(band[0], band[1], make([]float32, band[2]))
		if err == nil {
			test.Fatal("malformed band accepted", band)
		}
		sameBandBytes(test, destination, before)
	}
}

func TestFlameBandsSplitExactWordsWithoutTouchingNeighbours(test *testing.T) {
	writer, destination := flameBandFixture(test)
	words := bandWords(4 * bandCells)
	samples := bandSamples(words)
	expected := bytes.Clone(destination)
	for _, band := range [][2]int{{0, 2}, {2, 1}} {
		first, down := band[0], band[1]
		low, high := first*bandPlane, (first+down)*bandPlane
		if err := writer.consume(first, down, samples[4*low:4*high]); err != nil {
			test.Fatal(err)
		}
		for cell := low; cell < high; cell++ {
			density, shine := words[4*cell:4*cell+1], words[4*cell+1:4*cell+4]
			expectBandWords(expected, bandFirst+cell, density)
			expectBandWords(expected, bandFirst+bandCells+3*cell, shine)
		}
		sameBandBytes(test, destination, expected)
	}
}

func TestFlameBandRefusalsDoNotWritePartialBytes(test *testing.T) {
	writer, destination := flameBandFixture(test)
	rejectMalformedBands(test, writer.consume, destination, 4)
}

func flameBandFixture(test *testing.T) (*flameBandWriter, []byte) {
	test.Helper()
	destination := bytes.Repeat([]byte{0xa5}, 4*(bandFirst+4*bandCells+7))
	encoded := &scene{}
	encoded.boxes.bytes = destination
	job := flameBake{grid: boxed{cells: [3]int{2, 3, 3}},
		density: bandFirst, shine: bandFirst + bandCells}
	writer := &flameBandWriter{encoded: encoded, job: job}
	if err := writer.validReservation(); err != nil {
		test.Fatal(err)
	}
	return writer, destination
}
