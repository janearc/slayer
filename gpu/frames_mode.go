package gpu

import (
	"encoding/binary"
	"errors"
)

// Mode is how a Frames drawer traces, one of the per-call functions'
// ways: Software as SampledMeasured, NativeShadows as SampledNative,
// NativeMeshes as SampledAccelerated, CachedLighting as SampledCached,
// FastLighting as SampledFast and FasterLighting as SampledFaster. The
// zero Mode is none and is refused.
type Mode uint8

// the modes, in the order of the per-call functions; modes is one past
// the last, so a Mode below it and above noMode is known
const (
	noMode Mode = iota
	Software
	NativeShadows
	NativeMeshes
	CachedLighting
	FastLighting
	FasterLighting
	modes
)

// known is whether the mode is one of the six.
func (mode Mode) known() bool { return mode > noMode && mode < modes }

// framesRefusal is a refusal of the Frames drawer, held as a constant so
// that no one can assign it.
type framesRefusal string

// Error is the refusal's words.
func (r framesRefusal) Error() string { return string(r) }

// ErrFramesClosed is a draw on a Frames drawer after its Close.
const ErrFramesClosed framesRefusal = "gpu: the frames drawer is closed; " +
	"make another with NewFrames"

// errFramesFailed is a draw on a Frames drawer after one of its draws
// failed. The drawer has let its device state go: a kernel that timed
// out may still be running and is never used again (see metal.SetLimit),
// so a caller does not draw the same work again in this process.
var errFramesFailed = errors.New("gpu: the frames drawer failed earlier " +
	"and draws no more")

// errNoWorld is a draw asked of no world.
var errNoWorld = errors.New("gpu: a frame needs a world to draw")

// errNoMode is a drawer asked for with a mode that is none of the six.
var errNoMode = errors.New("gpu: no such drawing mode; use Software, " +
	"NativeShadows, NativeMeshes, CachedLighting, FastLighting or " +
	"FasterLighting")

// fieldWords is the numbers in a field of a structure the kernel reads,
// and fieldBytes its bytes: four words, by structure's own layout.
const (
	fieldWords = 4
	fieldBytes = fieldWords * wordBytes
)

// patch rewrites the named fields of one row of the structure in place,
// as row writes them, and leaves every other field as it is. A name the
// structure lacks, or a field given more than four numbers, is a mistake
// in the code and panics, as row does.
func (made structure) patch(row []byte, given values) {
	for name, numbers := range given {
		at := fieldOffset(made, name)
		if len(numbers) > fieldWords {
			made.wrong("%s holds %d, is given %d", name, fieldWords,
				len(numbers))
		}
		member := made.fields[at/fieldBytes]
		var four [fieldWords]float64
		copy(four[:], numbers)
		for index, value := range four {
			binary.LittleEndian.PutUint32(row[at+index*wordBytes:],
				member.bits(value))
		}
	}
}
