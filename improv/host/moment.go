package host

import (
	"fmt"
	"image"
	"math"

	"github.com/janearc/slayer/maths/noise"
)

// The clips a host has, by name.
const (
	resting  = "resting"
	speaking = "speaking"
	angry    = "angry"
	flash    = "flash"
	quiet    = "quiet"
)

// The numbers the package is made of, defined.
const (
	none = 0
	one  = 1
	two  = 2
	turn = 2 * math.Pi
)

// The numbers the package picked: the quasar slice, 2026-11-12.
const (
	// hardHit is the loudness, as a share of the loudest moment, over
	// which a kick is a hard hit and the host flashes.
	hardHit = 0.8
	// hush is the loudness under which the music is in a break and the
	// host goes quiet and holds still.
	hush = 0.12
	// lifted is the most the picture is lifted on a kick, as a share of
	// white added.
	lifted = 0.18
	// jump is how far up the host jumps on a kick, as a share of the
	// box's height.
	jump = 0.25
	// drifts are the host's drift, in turns a second, across and down:
	// slow, and not in step, so the path does not repeat soon.
	driftAcross = 0.07
	driftDown   = 0.11
	// flickerDepth is how much the host dims at the most on a flicker,
	// and flickerRate how many times a second the flicker changes.
	flickerDepth = 0.25
	flickerRate  = 12
)

// moment is what is happening at an instant: the tune's part, how strong
// a kick is there (one on the kick, falling to nought), how loud the music
// is as a share of its loudest, and whether the host is speaking.
type moment struct {
	Part     string
	Kick     float64
	Loudness float64
	Speaking bool
}

// pick is the host's clip at a moment: quiet in a break, flash on a hard
// hit, speaking while a sentence is up, resting otherwise.
func pick(at moment) string {
	switch {
	case at.Loudness < hush:
		return quiet
	case at.Kick > hardHit && at.Loudness > hardHit:
		return flash
	case at.Speaking:
		return speaking
	}
	return resting
}

// Box is where the host may be: a rectangle of the frame, and the host's
// size in it.
type Box struct {
	area image.Rectangle
	size image.Point
}

// NewBox is a box of the frame the host moves in, for a host of a size.
// It refuses a host that does not fit the box.
func NewBox(area image.Rectangle, size image.Point) (Box, error) {
	if size.X < one || size.Y < one || size.X > area.Dx() ||
		size.Y > area.Dy() {
		return Box{}, fmt.Errorf("host: a host %v does not fit a box "+
			"%v: give a box larger than the host", size, area)
	}
	return Box{area: area, size: size}, nil
}

// place is where the host's top left is at a moment seconds in: drifting
// on a seeded path inside the box, jumping up on a kick, held at the
// box's middle foot in a break. It never leaves the box.
func (box Box) place(at moment, seconds float64, seed uint64) image.Point {
	room := box.area.Size().Sub(box.size)
	if at.Loudness < hush {
		return box.area.Min.Add(image.Pt(room.X/two, room.Y))
	}
	phase := seeded(seed)
	across := (one + math.Sin(turn*(driftAcross*seconds+phase))) / two
	down := (one + math.Sin(turn*(driftDown*seconds+two*phase))) / two
	down = math.Max(none, down-jump*at.Kick)
	return box.area.Min.Add(image.Pt(int(across*float64(room.X)),
		int(down*float64(room.Y))))
}

// Flicker is the host's brightness at seconds in, a share of full: it
// changes flickerRate times a second, seeded, never under one less the
// flicker's depth.
func Flicker(seconds float64, seed uint64) float64 {
	tick := uint64(math.Floor(seconds * flickerRate))
	return one - flickerDepth*seeded(seed^(tick*noise.Golden))
}

// kickLift is how much white is added to the picture at a moment: a share
// of the kick, in a part that lifts; none elsewhere.
func kickLift(at moment, lifts map[string]bool) float64 {
	if !lifts[at.Part] || at.Loudness < hush {
		return none
	}
	return lifted * at.Kick
}

// seeded is a share from nought up to one, the same for the same seed:
// the seed stepped once and stirred by slayer's one hash.
func seeded(seed uint64) float64 {
	return noise.Stirred(seed + noise.Golden)
}
