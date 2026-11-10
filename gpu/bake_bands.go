//go:build darwin

package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
)

// packedBandWriter writes contiguous scalar/vector grid bands into scene bytes.
// Dimensions and the complete reservation are checked before any dispatch.
type packedBandWriter struct {
	bytes                               []byte
	first, plane, depth, channels, next int
}

// newPackedBandWriter checks its entire packed range without a product
// overflow.
func newPackedBandWriter(destination []byte, first int, cells [3]int,
	channels int,
) (*packedBandWriter, error) {
	available := len(destination) / 4
	if len(destination)%4 != 0 || first < 0 || first > available || channels < 1 {
		return nil, fmt.Errorf("gpu: invalid packed bake reservation")
	}
	remaining := (available - first) / channels
	for _, count := range cells {
		if count < 1 || count > remaining {
			return nil, fmt.Errorf("gpu: packed bake exceeds its reserved grid")
		}
		remaining /= count
	}
	return &packedBandWriter{bytes: destination, first: first, plane: cells[0] * cells[1],
		depth: cells[2], channels: channels}, nil
}

// consume keeps global plane order and copies each scalar's exact float32 bits.
func (writer *packedBandWriter) consume(first, down int, samples []float32) error {
	if first != writer.next || down <= 0 || first < 0 || down > writer.depth-first ||
		len(samples) != writer.plane*down*writer.channels {
		return fmt.Errorf("gpu: invalid or out-of-order packed bake band")
	}
	begin := writer.first + first*writer.plane*writer.channels
	for index, value := range samples {
		binary.LittleEndian.PutUint32(writer.bytes[(begin+index)*4:], math.Float32bits(value))
	}
	writer.next += down
	return nil
}
