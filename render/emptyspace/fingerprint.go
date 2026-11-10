package emptyspace

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"math"

	"github.com/janearc/slayer/maths"
)

// fingerprint validates fields and hashes their actual scalar bits in one pass.
// Negative/non-finite inputs cannot certify empty transport support.
func fingerprint(bounds maths.Box, cells [3]int, fields []Field,
	brickSide int,
) ([sha256.Size]byte, bool, error) {
	digest := sha256.New()
	digest.Write([]byte("slayer-emptyspace-v1\x00"))
	for _, value := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z,
		bounds.Max.X, bounds.Max.Y, bounds.Max.Z} {
		writeWord(digest, math.Float64bits(value))
	}
	for _, count := range []int{cells[0], cells[1], cells[2], brickSide, len(fields)} {
		writeWord(digest, uint64(count))
	}
	any := false
	for _, field := range fields {
		writeWord(digest, uint64(field.Channels))
		nonzero, err := writeSamples(digest, field.Values)
		if err != nil {
			return [sha256.Size]byte{}, false, err
		}
		any = any || nonzero
	}
	var stamp [sha256.Size]byte
	copy(stamp[:], digest.Sum(nil))
	return stamp, any, nil
}

// writeWord fixes metadata encoding independently of host integer/byte order.
func writeWord(digest hash.Hash, value uint64) {
	var bytes [8]byte
	binary.LittleEndian.PutUint64(bytes[:], value)
	digest.Write(bytes[:])
}

// writeSamples hashes in SHA-256 blocks instead of allocating a slice per
// scalar.
func writeSamples(digest hash.Hash, values []float32) (bool, error) {
	var bytes [sha256.BlockSize]byte
	used, nonzero := 0, false
	for _, value := range values {
		if !finite(float64(value)) || value < 0 {
			return false, fmt.Errorf("emptyspace: field has negative or non-finite samples")
		}
		nonzero = nonzero || value != 0
		binary.LittleEndian.PutUint32(bytes[used:], math.Float32bits(value))
		used += 4
		if used == len(bytes) {
			digest.Write(bytes[:])
			used = 0
		}
	}
	digest.Write(bytes[:used])
	return nonzero, nil
}
