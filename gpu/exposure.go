package gpu

// An exposure: a shutter that stays open while the eye moves, and a lens
// with an aperture, both made in the kernel at the primary ray.
//
// Design. A film frame used to have no depth of field, and its motion
// blur cost one whole render for each glimpse. Here each sample of one
// render takes its own time within the shutter and its own place on the
// aperture, so N samples a pixel give both at about the cost of N samples.
//
// What moves into the kernel is the making of the primary ray, and
// nothing after it. trace.metal's `exposed` makes the ray where the header
// says the shutter is open or the aperture is; otherwise the ray is made
// by `look` as before, and the picture is the one it always was.
//
// The header gains five rows. `close_origin`, `close_right` and
// `close_back` are the eye where the shutter closes; the eye where it
// opens is the header's own. `optics` is the lens: its aperture's reach,
// its focus distance, and the width and height it sees sharp at the focus.
//
// `exposure` is whether the shutter is open, whether the aperture is,
// how many samples lie along a pixel's side, and the frame's seed.
//
// The shutter. A sample's time is a share of the open shutter, from
// nought up to one. The samples of a pixel are counted along its rows;
// each takes one of as many equal shares of the shutter as the pixel has
// samples, so a pixel's samples cover the shutter evenly.
//
// Which share a sample takes is its count turned by a whole number the
// pixel's hash gives, and where in the share is the sample's own hash.
// Neighbouring pixels so take different times, and a moving edge is noise
// and not a row of copies.
//
// The hash is the kernel's `unit_of`, seeded by the frame's seed, which
// is made from the frame's time. The same frame is the same picture every
// time it is drawn, and two frames of a film do not share their noise.
//
// The eye at a time lies between the two: its place along the straight
// line, and its axes the two eyes' mixed and made square again. That is
// exact for an eye that slides and a small-angle approximation for one
// that turns: a turn of thirty degrees within one shutter is drawn less
// than a seventh of a degree from where an even turn would put it.
//
// One sample a pixel is one time a pixel: noise, not blur. Four along a
// side, sixteen a pixel, is where a blur reads as one; a fast move wants
// more.
//
// Only the eye moves. The world is encoded at one instant, so a thing
// that moves while the shutter is open is not blurred here. That stays
// with glimpses: several draws, each at its own time, averaged.
//
// The lens. A thin lens (ninjafilm/lens.Thin) is not sent as rays. The
// kernel makes each ray as Thin.Ray does: from a point on the aperture
// toward the point at the focus the pixel's middle sees sharp.
//
// The point on the aperture is the same one Thin takes: Roberts' r2
// sequence at the sample's index, row by row across the whole picture,
// carried onto the round opening by the concentric map. The index times
// the sequence's step is worked in 64-bit whole numbers, since a float
// has too few figures for a picture's millionth sample.
//
// So the kernel's rays are the tracer's to a float's precision, and a
// picture through a thin lens costs no ray input: 24 bytes a sample
// saved. Any other sensor that is not a pinhole still has its rays made
// on the cpu and sent, and a shutter over one of those is refused by name.
//
// The lens's corner falloff and its exposure stay on the host, on the
// finished frame, as before.
//
// Where the cpu can do the same. The tracer has no shutter of its own. A
// sensor that makes each sample's ray at the kernel's time for it gives
// the tracer the same rays, and the tests hold the kernel to that at the
// package's exact tolerance. Through a thin lens the tracer is Thin's own.

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	thin "github.com/janearc/slayer/ninjafilm/lens"
)

// plastic is the plastic number, the real root of x cubed equals x plus
// one, as ninjafilm/lens has it: Roberts' r2 sequence steps by its
// reciprocal along one side of the square and by that squared along the
// other. Defined by that equation.
const plastic = 1.32471795724474602596

// r2Across and r2Up are the sequence's two steps, as lens.Thin works
// them: one over the plastic number, and one over its square.
const (
	r2Across = wholeOne / plastic
	r2Up     = wholeOne / (plastic * plastic)
)

// turnBits is defined as the bits of the kernel's widest whole number,
// which a share of the square is written in so that a step times any
// index wraps round the square exactly.
const turnBits = 64

// eighthTurn and quarterTurn are the concentric map's angles, defined as
// pi over four and pi over two.
const (
	eighthTurn  = math.Pi / wholeFour
	quarterTurn = math.Pi / wholeTwo
)

// shutterTurn and shutterFirst are the third index the kernel's hash is
// asked with for a pixel's turn and for its first sample's place in a
// share; each later sample asks with the next. Defined as nought and one,
// so no two asks of a pixel share an index.
const (
	shutterTurn  = wholeZero
	shutterFirst = wholeOne
)

// errShutterOverRays is the refusal of an open shutter over a lens whose
// rays the cpu makes.
var errShutterOverRays = errors.New("gpu: a shutter is open over a lens " +
	"whose rays are made on the cpu, which have one eye; draw it through " +
	"a pinhole or a thin lens, or with the shutter closed")

// shutter is a frame's open shutter: the eye where it closes, how many
// samples lie along a pixel's side, and the frame's time, which seeds
// the samples' times. The zero shutter is closed.
type shutter struct {
	closing *maths.Subspace
	side    int
	at      time.Duration
}

// turned is a share of the square, nought up to one, as the kernel's
// widest whole number: the share times two to the sixty-fourth.
func turned(share float64) uint64 {
	scaled := new(big.Float).SetMantExp(big.NewFloat(share), turnBits)
	whole, _ := scaled.Uint64()
	return whole
}

// seedOf is a frame's seed: its time in nanoseconds, stirred as SplitMix64
// stirs a number, and cut to the header's thirty-two bits.
func seedOf(at time.Duration) uint32 {
	stirred := uint64(at) + splitGolden
	stirred = (stirred ^ stirred>>splitFirst) * splitOnce
	stirred = (stirred ^ stirred>>splitSecond) * splitTwice
	return uint32(stirred ^ stirred>>splitThird)
}

// SplitMix64's numbers, defined by Steele, Lea and Flood's generator, as
// the kernel's unit_of has them.
const (
	splitGolden = 0x9e3779b97f4a7c15
	splitOnce   = 0xbf58476d1ce4e5b9
	splitTwice  = 0x94d049bb133111eb
	splitFirst  = 30
	splitSecond = 27
	splitThird  = 31
)

// thinOf is the lens as a thin lens, and whether it is one, by value or
// by pointer.
func thinOf(lens film.Sensor) (thin.Thin, bool) {
	if pointer, ok := lens.(*thin.Thin); ok && pointer != nil {
		return *pointer, true
	}
	glass, ok := lens.(thin.Thin)
	return glass, ok
}

// kernelMade is whether the kernel makes the lens's rays itself: a
// pinhole's or a thin lens's.
func kernelMade(lens film.Sensor) bool {
	_, pinhole := pinholeOf(lens)
	_, glass := thinOf(lens)
	return pinhole || glass
}

// over is whether the shutter is open over an eye that moves: whether
// the eye where it closes is another eye, as the header holds the two. A
// shutter open over a still eye is closed: every time in it sees the same.
func (open shutter) over(eye maths.Subspace) bool {
	if open.closing == nil {
		return false
	}
	from := cameraValues(eye, film.Pinhole{}, wholeOne, wholeOne)
	to := cameraValues(*open.closing, film.Pinhole{}, wholeOne, wholeOne)
	for _, name := range []string{"origin", "right", "up", "back"} {
		for index, value := range from[name] {
			if float32(value) != float32(to[name][index]) {
				return true
			}
		}
	}
	return false
}

// viewValues is the header's fields that the eye, the lens and the
// shutter make, for a picture width by height samples. A camera move,
// another focus or another shutter changes these and nothing else. With
// the shutter closed and no aperture the exposure's rows are all nought.
func viewValues(eye maths.Subspace, lens film.Sensor, open shutter, width,
	height int) values {
	pinhole, _ := pinholeOf(lens)
	glass, opened := thinOf(lens)
	if opened {
		pinhole = film.Pinhole{Wide: glass.Wide()}
	}
	given := cameraValues(eye, pinhole, width, height)
	given["optics"] = opticsOf(eye, glass, opened)
	given["close_origin"], given["close_right"] = nil, nil
	given["close_back"] = nil
	given["exposure"] = wholes(wholeZero, flag(opened))
	if !open.over(eye) {
		return given
	}
	closed := cameraValues(*open.closing, pinhole, width, height)
	given["close_origin"] = closed["origin"]
	given["close_right"] = closed["right"]
	given["close_back"] = closed["back"]
	given["exposure"] = wholes(wholeOne, flag(opened),
		max(open.side, wholeOne), int(int32(seedOf(open.at))))
	return given
}

// opticsOf is a thin lens as the header holds it, or nothing for a lens
// that is not one. The aperture's reach is scaled as the eye scales a
// ray's start, as film.Look takes it out of the eye's space.
func opticsOf(eye maths.Subspace, glass thin.Thin, opened bool) []float64 {
	if !opened {
		return nil
	}
	reach, focus, across, up := glass.Opening()
	scale := eye.Scale
	if scale == wholeZero {
		scale = wholeOne
	}
	return []float64{reach * scale, focus, across, up}
}

// exposureNumbers is the kernel's declaration of the numbers an exposure
// is made with: the r2 sequence's steps and its half as shares of the
// square, the concentric map's angles, and the hash's indices.
func exposureNumbers() string {
	return fmt.Sprintf(`constexpr constant ulong r2_across = %dUL;
constexpr constant ulong r2_up = %dUL;
constexpr constant ulong r2_half = %dUL;
constexpr constant float eighth_turn = %.9g;
constexpr constant float quarter_turn = %.9g;
constexpr constant float hash_whole = %.9g;
constexpr constant int shutter_turn = %d;
constexpr constant int shutter_first = %d;
`, turned(r2Across), turned(r2Up), turned(beamHalf), eighthTurn,
		quarterTurn, wholeOne/hashUnit, shutterTurn, shutterFirst)
}
