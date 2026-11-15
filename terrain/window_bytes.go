package terrain

import (
	"encoding/binary"
	"fmt"
	"math"
)

// A window as bytes, so a run can keep the ground it cut and smoothed
// and the next read it back: its west, north and step, its counts across
// and down, and every height, each eight bytes, little-endian, a float's
// every bit. Read back it is the same window to the bit.

// windowHead is how many numbers come before the heights. Defined: west,
// north, step, across and down.
const windowHead = 5

// numberBytes is the bytes each number takes. Defined.
const numberBytes = 8

// Bytes is the window as bytes.
func (window Window) Bytes() []byte {
	made := make([]byte, 0, (windowHead+len(window.heights))*numberBytes)
	for _, number := range []float64{window.west, window.north,
		window.step} {
		made = binary.LittleEndian.AppendUint64(made, math.Float64bits(number))
	}
	made = binary.LittleEndian.AppendUint64(made, uint64(window.across))
	made = binary.LittleEndian.AppendUint64(made, uint64(window.down))
	for _, height := range window.heights {
		made = binary.LittleEndian.AppendUint64(made, math.Float64bits(height))
	}
	return made
}

// WindowFromBytes is the window Bytes gave, or a refusal of bytes whose
// length is not the one their counts say.
func WindowFromBytes(raw []byte) (Window, error) {
	if len(raw) < windowHead*numberBytes || len(raw)%numberBytes != 0 {
		return Window{}, fmt.Errorf("terrain: %d bytes is not a window",
			len(raw))
	}
	next := func() uint64 {
		number := binary.LittleEndian.Uint64(raw)
		raw = raw[numberBytes:]
		return number
	}
	window := Window{west: math.Float64frombits(next()),
		north: math.Float64frombits(next()),
		step:  math.Float64frombits(next())}
	across, down := next(), next()
	points := uint64(len(raw) / numberBytes)
	if across == 0 || down == 0 || across > points || across*down != points {
		return Window{}, fmt.Errorf("terrain: a window of %d by %d with "+
			"%d heights", across, down, points)
	}
	window.across, window.down = int(across), int(down)
	window.heights = make([]float64, points)
	for i := range window.heights {
		window.heights[i] = math.Float64frombits(next())
	}
	return window, nil
}
