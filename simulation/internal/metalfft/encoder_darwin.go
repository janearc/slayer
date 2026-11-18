//go:build darwin

package metalfft

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"

	"github.com/janearc/slayer/metal"
)

// Encoder appends a fixed grid's transform stages to one session. The
// caller owns its buffers, its commits, and the session's lifetime.
type Encoder struct {
	shape    Shape
	cells    int
	session  *metal.Session
	twiddles metal.Buffer
	failed   error
}

// New retains twiddles for the shape on a caller-owned session, which
// must already contain every function listed by Kernels.
func New(session *metal.Session, shape Shape) (*Encoder, error) {
	cells, err := shape.Cells()
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("metalfft: an encoder needs a session")
	}
	twiddles, err := makeTwiddles(session, shape)
	if err != nil {
		return nil, err
	}
	return newEncoder(session, shape, cells, twiddles), nil
}

// newEncoder assembles an encoder after validation and buffer allocation.
func newEncoder(session *metal.Session, shape Shape, cells int,
	twiddles metal.Buffer) *Encoder {
	return &Encoder{shape: shape, cells: cells, session: session,
		twiddles: twiddles}
}

// makeTwiddles stores row roots followed by column roots, once per plan.
func makeTwiddles(session *metal.Session, shape Shape) (metal.Buffer, error) {
	buffer, err := session.Buffer(shape.Width + shape.Height)
	if err != nil {
		return metal.Buffer{}, fmt.Errorf("metalfft: twiddles: %w", err)
	}
	values := buffer.Numbers()
	position := 0
	for _, dimension := range []int{shape.Width, shape.Height} {
		for index := range dimension / 2 {
			sine, cosine := math.Sincos(-2 * math.Pi *
				float64(index) / float64(dimension))
			values[position], values[position+1] = float32(cosine),
				float32(sine)
			position += complexParts
		}
	}
	return buffer, nil
}

// Append queues the transform and returns its result buffer. Output and
// scratch differ; scratch differs from input. Input may equal output for
// deliberate in-place work. No dispatch is committed here.
func (encoder *Encoder) Append(input, output, scratch metal.Buffer,
	inverse bool) (metal.Buffer, error) {
	if err := encoder.ready(input, output, scratch); err != nil {
		return metal.Buffer{}, err
	}
	shape := encoder.shape
	params := words(uint32(shape.Width), uint32(shape.Height),
		uint32(bits.TrailingZeros(uint(shape.Width))),
		uint32(bits.TrailingZeros(uint(shape.Height))))
	if err := encoder.queue(permutationKernel, shape.Width, shape.Height,
		params, input, scratch); err != nil {
		return metal.Buffer{}, err
	}
	current, next := scratch, output
	for _, axis := range []axisPass{
		{rowKernel, shape.Width, shape.Width / 2, shape.Height},
		{columnKernel, shape.Height, shape.Width, shape.Height / 2},
	} {
		for span := 2; span <= axis.length; span *= 2 {
			params := stageWords(shape, span, inverse)
			if err := encoder.queue(axis.kernel, axis.width,
				axis.height, params, current, next,
				encoder.twiddles); err != nil {
				return metal.Buffer{}, err
			}
			current, next = next, current
		}
	}
	if inverse {
		params := words(uint32(shape.Width), uint32(shape.Height),
			math.Float32bits(float32(1/float64(encoder.cells))), 0)
		if err := encoder.queue(scaleKernel, shape.Width, shape.Height,
			params, current); err != nil {
			return metal.Buffer{}, err
		}
	}
	return current, nil
}

// axisPass is the dispatch grid and full transform length for one axis.
type axisPass struct {
	kernel                string
	length, width, height int
}

// ready checks every buffer and alias before the first dispatch is queued.
func (encoder *Encoder) ready(input, output, scratch metal.Buffer) error {
	if encoder == nil || encoder.session == nil || encoder.cells == 0 {
		return fmt.Errorf("metalfft: no live encoder")
	}
	if encoder.failed != nil {
		return fmt.Errorf("metalfft: encoder failed earlier: %w",
			encoder.failed)
	}
	for _, buffer := range []metal.Buffer{input, output, scratch} {
		if len(buffer.Numbers()) != complexParts*encoder.cells {
			return fmt.Errorf("metalfft: a buffer has %d "+
				"components, "+
				"want %d", len(buffer.Numbers()),
				complexParts*encoder.cells)
		}
	}
	if sameBuffer(output, scratch) || sameBuffer(input, scratch) {
		return fmt.Errorf("metalfft: scratch must differ from input " +
			"and output")
	}
	return nil
}

// sameBuffer compares allocation identity after nonempty lengths are checked.
func sameBuffer(first, second metal.Buffer) bool {
	return &first.Numbers()[0] == &second.Numbers()[0]
}

// queue remembers a dispatch failure so a partial batch cannot be resumed.
func (encoder *Encoder) queue(kernel string, width, height int,
	params []byte, buffers ...metal.Buffer) error {
	if err := encoder.session.Dispatch(kernel, width, height, params,
		buffers...); err != nil {
		encoder.failed = fmt.Errorf("metalfft: dispatch %s: %w",
			kernel, err)
		return encoder.failed
	}
	return nil
}

// stageWords matches SpectralFFTStage's four scalar words exactly.
func stageWords(shape Shape, span int, inverse bool) []byte {
	var reverse uint32
	if inverse {
		reverse = 1
	}
	return words(uint32(shape.Width), uint32(shape.Height),
		uint32(span), reverse)
}

// words packs one 16-byte shader parameter block without Go struct padding.
func words(first, second, third, fourth uint32) []byte {
	var packed [4 * floatBytes]byte
	for index, word := range [4]uint32{first, second, third, fourth} {
		binary.LittleEndian.PutUint32(packed[index*floatBytes:], word)
	}
	return packed[:]
}
